package script

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v1"

	"github.com/drone/drone/plugin/deploy"
	"github.com/drone/drone/plugin/notify"
	"github.com/drone/drone/plugin/publish"
	"github.com/drone/drone/shared/build/buildfile"
	"github.com/drone/drone/shared/build/git"
	"github.com/drone/drone/shared/build/repo"

	"github.com/kr/pretty"
)

func ParseBuild(data string, params map[string]string) (*Build, error) {
	var yml *Build

	// parse the build configuration file
	err := yaml.Unmarshal(injectParams([]byte(data), params), &yml)
	if err != nil {
		return nil, err
	}

	yml.Type = "matrix"

	if yml.Matrix == nil {
		yml.Type = "build"
		matrix := Matrix{
			Image:    yml.Image,
			Script:   yml.Script,
			Services: yml.Services,
			Env:      yml.Env,
			Hosts:    yml.Hosts,
			Cache:    yml.Cache,
			Branches: yml.Branches,
			Publish:  yml.Publish,
			Deploy:   yml.Deploy,
			Git:      yml.Git,
		}

		if yml.Name == "" {
			matrix.Name = "Build 1"
		} else {
			matrix.Name = yml.Name
		}

		yml.Matrix = append(yml.Matrix, &matrix)
	}

	pretty.Log(yml)

	return yml, nil
}

func ParseBuildFile(filename string) (*Build, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseBuild(string(data), nil)
}

// injectParams injects params into data.
func injectParams(data []byte, params map[string]string) []byte {
	for k, v := range params {
		data = bytes.Replace(data, []byte(fmt.Sprintf("{{%s}}", k)), []byte(v), -1)
	}
	return data
}

type Matrix struct {
	// Image specifies the Docker Image that will be
	// used to virtualize the Build process.
	Image string

	// Name specifies a user-defined label used
	// to identify the build.
	Name string

	// Allow failures
	AllowFail bool

	// Script specifies the build and test commands.
	Script []string

	// Env specifies the environment of the build.
	Env []string

	// Hosts specifies the custom IP address and
	// hostname mappings.
	Hosts []string

	// Cache lists a set of directories that should
	// persisted between builds.
	Cache []string

	// Services specifies external services, such as
	// database or messaging queues, that should be
	// linked to the build environment.
	Services []string

	// White-list of Branches that are built.
	Branches []string

	Publish *publish.Publish `yaml:"publish,omitempty"`
	Deploy  *deploy.Deploy   `yaml:"deploy,omitempty"`

	// Git specified git-specific parameters, such as
	// the clone depth and path
	Git *git.Git `yaml:"git,omitempty"`
}

// Build stores the configuration details for
// building, testing and deploying code.
type Build struct {
	// Image specifies the Docker Image that will be
	// used to virtualize the Build process.
	Image string

	// build or matrix strategy
	Type string `yaml:"-"`

	// Name specifies a user-defined label used
	// to identify the build.
	Name string

	// Script specifies the build and test commands.
	Script []string

	// Env specifies the environment of the build.
	Env []string

	// Hosts specifies the custom IP address and
	// hostname mappings.
	Hosts []string

	// Cache lists a set of directories that should
	// persisted between builds.
	Cache []string

	// Services specifies external services, such as
	// database or messaging queues, that should be
	// linked to the build environment.
	Services []string

	// White-list of Branches that are built.
	Branches []string

	Matrix []*Matrix `yaml:"matrix,omitempty"`

	Notifications *notify.Notification `yaml:"notify,omitempty"`
	Publish       *publish.Publish     `yaml:"publish,omitempty"`
	Deploy        *deploy.Deploy       `yaml:"deploy,omitempty"`

	// Git specified git-specific parameters, such as
	// the clone depth and path
	Git *git.Git `yaml:"git,omitempty"`
}

// Write adds all the steps to the build script, including
// build commands, deploy and publish commands.
func (b *Build) Write(f *buildfile.Buildfile, r *repo.Repo, i int) {
	// append build commands
	b.WriteBuild(f, i)

	// write publish commands
	if b.Matrix[i].Publish != nil {
		b.Matrix[i].Publish.Write(f, r)
	}

	// write deployment commands
	if b.Matrix[i].Deploy != nil {
		b.Matrix[i].Deploy.Write(f, r)
	}

	// write exit value
	f.WriteCmd("exit 0")
}

// WriteBuild adds only the build steps to the build script,
// omitting publish and deploy steps. This is important for
// pull requests, where deployment would be undesirable.
func (b *Build) WriteBuild(f *buildfile.Buildfile, i int) {
	// append environment variables
	for _, env := range b.Env {
		parts := strings.Split(env, "=")
		if len(parts) != 2 {
			continue
		}
		f.WriteEnv(parts[0], parts[1])
	}

	// Write matrix variables only in matrix builds
	if b.Type == "matrix" {
		for _, env := range b.Matrix[i].Env {
			parts := strings.Split(env, "=")
			if len(parts) != 2 {
				continue
			}
			f.WriteEnv(parts[0], parts[1])
		}
	}

	// append build commands
	for _, cmd := range b.Script {
		f.WriteCmd(cmd)
	}

	// Write matrix commands only in matrix builds
	if b.Type == "matrix" {
		for _, cmd := range b.Matrix[i].Script {
			f.WriteCmd(cmd)
		}
	}
}

func (b *Build) MatchBranch(branch string) bool {
	if len(b.Branches) == 0 {
		return true
	}
	for _, item := range b.Branches {
		if item == branch {
			return true
		}
	}
	return false
}

type Publish interface {
	Write(f *buildfile.Buildfile)
}

type Deployment interface {
	Write(f *buildfile.Buildfile)
}

type Notification interface {
	Set(c Context)
}

type Context interface {
	Host() string
	Owner() string
	Name() string

	Branch() string
	Hash() string
	Status() string
	Message() string
	Author() string
	Gravatar() string

	Duration() int64
	HumanDuration() string

	//Settings
}
