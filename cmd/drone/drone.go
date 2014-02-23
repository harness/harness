package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/drone/drone/pkg/build"
	"github.com/drone/drone/pkg/build/log"
	"github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"

	"launchpad.net/goyaml"
)

// A buildParams represents build parameters which are
// injected into the yaml configuration file.
type buildParams []string

// String returns the string value of the buildParams.
func (p *buildParams) String() string {
	return fmt.Sprint(*p)
}

// Set sets the value into the buildParams.
func (p *buildParams) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		*p = append(*p, v)
	}
	return nil
}

// Map returns a map of the buildParams.
func (p *buildParams) Map() map[string]string {
	m := map[string]string{}
	for _, prm := range *p {
		if kv := strings.SplitN(prm, "=", 2); len(kv) == 2 {
			m[kv[0]] = kv[1]
		}
	}
	return m
}

var (
	// identity file (id_rsa) that will be injected
	// into the container if specified
	identity = flag.String("identity", "", "")

	// runs Drone in parallel mode if True
	parallel = flag.Bool("parallel", false, "")

	// build will timeout after N milliseconds.
	// this will default to 500 minutes (6 hours)
	timeout = flag.Duration("timeout", 300*time.Minute, "")

	// runs Drone with verbose output if True
	verbose = flag.Bool("v", false, "")

	// displays the help / usage if True
	help = flag.Bool("h", false, "")

	// build parameters
	params buildParams
)

func init() {
	// default logging
	log.SetPrefix("\033[2m[DRONE] ")
	log.SetSuffix("\033[0m\n")
	log.SetOutput(os.Stdout)
	log.SetPriority(log.LOG_NOTICE)

	flag.Var(&params, "param", "")
}

func main() {
	// Parse the input parameters
	flag.Usage = usage
	flag.Parse()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if *verbose {
		log.SetPriority(log.LOG_DEBUG)
	}

	// Must speicify a command
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(0)
	}

	switch {
	// run drone build assuming the current
	// working directory contains the drone.yml
	case args[0] == "build" && len(args) == 1:
		path, _ := os.Getwd()
		path = filepath.Join(path, ".drone.yml")
		run(path, params)

	// run drone build where the path to the
	// source directory is provided
	case args[0] == "build" && len(args) == 2:
		path := args[1]
		path = filepath.Clean(path)
		path, _ = filepath.Abs(path)
		path = filepath.Join(path, ".drone.yml")
		run(path, params)

	// run drone vet where the path to the
	// source directory is provided
	case args[0] == "vet" && len(args) == 2:
		path := args[1]
		path = filepath.Clean(path)
		path, _ = filepath.Abs(path)
		path = filepath.Join(path, ".drone.yml")
		vet(path, params)

	// run drone vet assuming the current
	// working directory contains the drone.yml
	case args[0] == "vet" && len(args) == 1:
		path, _ := os.Getwd()
		path = filepath.Join(path, ".drone.yml")
		vet(path, params)

	// print the help message
	case args[0] == "help" && len(args) == 1:
		flag.Usage()
	}

	os.Exit(0)
}

func vet(path string, params buildParams) {
	// parse the Drone yml file
	script, err := script.ParseBuildFile(path, params.Map())
	if err != nil {
		log.Err(err.Error())
		os.Exit(1)
		return
	}

	// print the Drone yml as parsed
	out, _ := goyaml.Marshal(script)
	log.Noticef("parsed yaml:\n%s", string(out))
}

func run(path string, params buildParams) {
	paramsMap := params.Map()
	// parse the Drone yml file
	s, err := script.ParseBuildFile(path, paramsMap)
	if err != nil {
		log.Err(err.Error())
		os.Exit(1)
		return
	}

	// set environment variables
	for k, v := range paramsMap {
		s.Env = append(s.Env, k+"="+v)
	}

	// get the repository root directory
	dir := filepath.Dir(path)
	code := repo.Repo{Path: dir}

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

	// track all build results
	var builders []*build.Builder

	// ssh key to import into container
	var key []byte
	if len(*identity) != 0 {
		key, err = ioutil.ReadFile(*identity)
		if err != nil {
			fmt.Printf("[Error] Could not find or read identity file %s\n", *identity)
			os.Exit(1)
			return
		}
	}

	builds := []*script.Build{s}

	// loop through and create builders
	for _, b := range builds { //script.Builds {
		builder := build.Builder{}
		builder.Build = b
		builder.Repo = &code
		builder.Key = key
		builder.Stdout = os.Stdout
		builder.Timeout = *timeout

		if *parallel == true {
			var buf bytes.Buffer
			builder.Stdout = &buf
		}

		builders = append(builders, &builder)
	}

	switch *parallel {
	case false:
		runSequential(builders)
	case true:
		runParallel(builders)
	}

	// if in parallel mode, print out the buffer
	// if we had a failure
	for _, builder := range builders {
		if builder.BuildState.ExitCode == 0 {
			continue
		}

		if buf, ok := builder.Stdout.(*bytes.Buffer); ok {
			log.Noticef("printing stdout for failed build %s", builder.Build.Name)
			println(buf.String())
		}
	}

	// this exit code is initially 0 and will
	// be set to an error code if any of the
	// builds fail.
	var exit int

	fmt.Printf("\nDrone Build Results \033[90m(%v)\033[0m\n", len(builders))

	// loop through and print results
	for _, builder := range builders {
		build := builder.Build
		res := builder.BuildState
		duration := time.Duration(res.Finished - res.Started)
		switch {
		case builder.BuildState.ExitCode == 0:
			fmt.Printf(" \033[32m\u2713\033[0m %v \033[90m(%v)\033[0m\n", build.Name, humanizeDuration(duration*time.Second))
		case builder.BuildState.ExitCode != 0:
			fmt.Printf(" \033[31m\u2717\033[0m %v \033[90m(%v)\033[0m\n", build.Name, humanizeDuration(duration*time.Second))
			exit = builder.BuildState.ExitCode
		}
	}

	os.Exit(exit)
}

func runSequential(builders []*build.Builder) {
	// loop through and execute each build
	for _, builder := range builders {
		if err := builder.Run(); err != nil {
			log.Errf("Error executing build: %s", err.Error())
			os.Exit(1)
		}
	}
}

func runParallel(builders []*build.Builder) {
	// spawn four worker goroutines
	var wg sync.WaitGroup
	for _, builder := range builders {
		// Increment the WaitGroup counter
		wg.Add(1)
		// Launch a goroutine to run the build
		go func(builder *build.Builder) {
			defer wg.Done()
			builder.Run()
		}(builder)
		time.Sleep(500 * time.Millisecond) // get weird iptables failures unless we sleep.
	}

	// wait for the workers to finish
	wg.Wait()
}

var usage = func() {
	fmt.Println(`Drone is a tool for building and testing code in Docker containers.

Usage:

	drone command [arguments]

The commands are:

   build                      build and test the repository
   version                    print the version number
   vet                        validate the yaml configuration file

  -v                          runs drone with verbose output
  -h                          display this help and exit
  --parallel                  runs drone build tasks in parallel
  --timeout=300ms             timeout build after 300 milliseconds
  -param 'key=value'          Parameter for the yaml configuration file, can be used multiple times.

Examples:
  drone build                 builds the source in the pwd
  drone build /path/to/repo   builds the source repository

Use "drone help [command]" for more information about a command.
`)
}
