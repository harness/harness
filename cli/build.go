package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/drone/drone/shared/build"
	"github.com/drone/drone/shared/build/docker"
	"github.com/drone/drone/shared/build/log"
	"github.com/drone/drone/shared/build/repo"
	"github.com/drone/drone/shared/build/script"

	"github.com/codegangsta/cli"
)

const EXIT_STATUS = 1

// NewBuildCommand returns the CLI command for "build".
func NewBuildCommand() cli.Command {
	return cli.Command{
		Name:  "build",
		Usage: "run a local build",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "i",
				Value: "",
				Usage: "identify file injected in the container",
			},
			cli.BoolFlag{
				Name:  "p",
				Usage: "runs drone build in a privileged container",
			},
			cli.BoolFlag{
				Name:  "deploy",
				Usage: "runs drone build with deployments enabled",
			},
			cli.BoolFlag{
				Name:  "publish",
				Usage: "runs drone build with publishing enabled",
			},
			cli.StringFlag{
				Name:  "docker-host",
				Value: getHost(),
				Usage: "docker daemon address",
			},
			cli.StringFlag{
				Name:  "docker-cert",
				Value: getCert(),
				Usage: "docker daemon tls certificate",
			},
			cli.StringFlag{
				Name:  "docker-key",
				Value: getKey(),
				Usage: "docker daemon tls key",
			},
		},
		Action: func(c *cli.Context) {
			buildCommandFunc(c)
		},
	}
}

// buildCommandFunc executes the "build" command.
func buildCommandFunc(c *cli.Context) {
	var privileged = c.Bool("p")
	var identity = c.String("i")
	var deploy = c.Bool("deploy")
	var publish = c.Bool("publish")
	var path string

	var dockerhost = c.String("docker-host")
	var dockercert = c.String("docker-cert")
	var dockerkey = c.String("docker-key")

	// the path is provided as an optional argument that
	// will otherwise default to $PWD/.drone.yml
	if len(c.Args()) > 0 {
		path = c.Args()[0]
	}

	switch len(path) {
	case 0:
		path, _ = os.Getwd()
		path = filepath.Join(path, ".drone.yml")
	default:
		path = filepath.Clean(path)
		path, _ = filepath.Abs(path)
		path = filepath.Join(path, ".drone.yml")
	}

	// this configures the default Docker logging levels,
	// and suffix and prefix values.
	log.SetPrefix("\033[2m[DRONE] ")
	log.SetSuffix("\033[0m\n")
	log.SetOutput(os.Stdout)
	log.SetPriority(log.LOG_DEBUG) //LOG_NOTICE
	docker.Logging = false

	var exit, _ = run(path, identity, dockerhost, dockercert, dockerkey, publish, deploy, privileged)
	os.Exit(exit)
}

// TODO this has gotten a bit out of hand. refactor input params
func run(path, identity, dockerhost, dockercert, dockerkey string, publish, deploy, privileged bool) (int, error) {
	dockerClient, err := docker.NewHostCertFile(dockerhost, dockercert, dockerkey)
	if err != nil {
		log.Err(err.Error())
		return EXIT_STATUS, err
	}

	// parse the private environment variables
	envs := getParamMap("DRONE_ENV_")

	// parse the Drone yml file
	s, err := script.ParseBuildFile(path, envs)
	if err != nil {
		log.Err(err.Error())
		return EXIT_STATUS, err
	}

	// inject private environment variables into build script
	for key, val := range envs {
		s.Env = append(s.Env, key+"="+val)
	}

	if deploy == false {
		s.Deploy = nil
	}
	if publish == false {
		s.Publish = nil
	}

	// get the repository root directory
	dir := filepath.Dir(path)
	code := repo.Repo{
		Name:   filepath.Base(dir),
		Branch: "HEAD", // should we do this?
		Path:   dir,
	}

	// does the local repository match the
	// $GOPATH/src/{package} pattern? This is
	// important so we know the target location
	// where the code should be copied inside
	// the container.
	if gopath, ok := getRepoPath(dir); ok {
		code.Dir = gopath

	} else if gopath, ok := getGoPath(dir); ok {
		// in this case we found a GOPATH and
		// reverse engineered the package path
		code.Dir = gopath

	} else {
		// otherwise just use directory name
		code.Dir = filepath.Base(dir)
	}

	// this is where the code gets uploaded to the container
	// TODO move this code to the build package
	code.Dir = filepath.Join("/var/cache/drone/src", filepath.Clean(code.Dir))

	// ssh key to import into container
	var key []byte
	if len(identity) != 0 {
		key, err = ioutil.ReadFile(identity)
		if err != nil {
			fmt.Printf("[Error] Could not find or read identity file %s\n", identity)
			return EXIT_STATUS, err
		}
	}

	// loop through and create builders
	builder := build.New(dockerClient)
	builder.Build = s
	builder.Repo = &code
	builder.Key = key
	builder.Stdout = os.Stdout
	builder.Timeout = 300 * time.Minute
	builder.Privileged = privileged

	// execute the build
	if err := builder.Run(); err != nil {
		log.Errf("Error executing build: %s", err.Error())
		return EXIT_STATUS, err
	}

	fmt.Printf("\nDrone Build Results \033[90m(%s)\033[0m\n", dir)

	// loop through and print results

	build := builder.Build
	res := builder.BuildState
	duration := time.Duration(res.Finished - res.Started)
	switch {
	case builder.BuildState.ExitCode == 0:
		fmt.Printf(" \033[32m\u2713\033[0m %v \033[90m(%v)\033[0m\n", build.Name, humanizeDuration(duration*time.Second))
	case builder.BuildState.ExitCode != 0:
		fmt.Printf(" \033[31m\u2717\033[0m %v \033[90m(%v)\033[0m\n", build.Name, humanizeDuration(duration*time.Second))
	}

	return builder.BuildState.ExitCode, nil
}

func getHost() string {
	return os.Getenv("DOCKER_HOST")
}

func getCert() string {
	if os.Getenv("DOCKER_CERT_PATH") != "" && os.Getenv("DOCKER_TLS_VERIFY") == "1" {
		return filepath.Join(os.Getenv("DOCKER_CERT_PATH"), "cert.pem")
	} else {
		return ""
	}
}

func getKey() string {
	if os.Getenv("DOCKER_CERT_PATH") != "" && os.Getenv("DOCKER_TLS_VERIFY") == "1" {
		return filepath.Join(os.Getenv("DOCKER_CERT_PATH"), "key.pem")
	} else {
		return ""
	}
}
