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

type Result struct {
	Code     int
	Name     string
	Duration time.Duration
}

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
			cli.StringFlag{
				Name:  "p",
				Value: "false",
				Usage: "runs drone build in a privileged container",
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
	var path string

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

	status_codes, _ := run(path, identity, privileged)
	var exit_code int
	for _, v := range status_codes {
		if v.Code > exit_code {
			exit_code = v.Code
		}

		switch {
		case v.Code == 0:
			fmt.Printf(" \033[32m\u2713\033[0m %v \033[90m(%v)\033[0m\n", v.Name, humanizeDuration(v.Duration*time.Second))
		case v.Code != 0:
			fmt.Printf(" \033[31m\u2717\033[0m %v \033[90m(%v)\033[0m\n", v.Name, humanizeDuration(v.Duration*time.Second))
		}
	}
	os.Exit(exit_code)
}

func run(path, identity string, privileged bool) ([]*Result, error) {
	var results []*Result

	// parse the Drone yml file
	s, err := script.ParseBuildFile(path)
	if err != nil {
		log.Err(err.Error())
		results := append(results, &Result{EXIT_STATUS, "DRONE_YAML_PARSE", 0})
		return results, err
	}

	for i, b := range s.Matrix {
		code, name, duration := build_matrix(s, b, i, path, identity, privileged)
		results = append(results, &Result{code, name, duration})
	}

	return results, nil
}

func build_matrix(s *script.Build, b *script.Matrix, i int, path, identity string, priveleged bool) (int, string, time.Duration) {
	dockerClient := docker.New()
	// remove deploy & publish sections
	// for now, until I fix bug
	s.Matrix[i].Publish = nil
	s.Matrix[i].Deploy = nil

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
	var err error
	if len(identity) != 0 {
		key, err = ioutil.ReadFile(identity)
		if err != nil {
			// loop through and print results
			fmt.Printf("[Error] Could not find or read identity file %s\n", identity)
			return EXIT_STATUS, b.Name, 0
		}
	}

	// loop through and create builders
	builder := build.New(dockerClient)
	builder.Index = i
	builder.Build = s
	builder.Repo = &code
	builder.Key = key
	builder.Stdout = os.Stdout
	// TODO ADD THIS BACK
	builder.Timeout = 300 * time.Minute
	builder.Privileged = priveleged

	// execute the build
	if err := builder.Run(); err != nil {
		res := builder.BuildState
		duration := time.Duration(res.Finished - res.Started)
		log.Errf("Error executing build: %s", err.Error())
		return EXIT_STATUS, b.Name, duration
	}

	res := builder.BuildState
	duration := time.Duration(res.Finished - res.Started)
	return builder.BuildState.ExitCode, b.Name, duration
}
