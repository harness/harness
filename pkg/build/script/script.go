package script

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"strings"

	"launchpad.net/goyaml"

	"github.com/drone/drone/pkg/build/buildfile"
	"github.com/drone/drone/pkg/build/git"
	"github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/plugin/deploy"
	"github.com/drone/drone/pkg/plugin/notify"
	"github.com/drone/drone/pkg/plugin/publish"
)

func ParseBuild(data []byte, params map[string]string) (*Build, error) {
	build := Build{}

	// parse the build configuration file
	err := goyaml.Unmarshal(injectParams(data, params), &build)
	return &build, err
}

func ParseBuildFile(filename string) (*Build, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return ParseBuild(data, nil)
}

// injectParams injects params into data.
func injectParams(data []byte, params map[string]string) []byte {
	for k, v := range params {
		data = bytes.Replace(data, []byte(fmt.Sprintf("{{%s}}", k)), []byte(v), -1)
	}
	return data
}

// Build stores the configuration details for
// building, testing and deploying code.
type Build struct {
	// Image specifies the Docker Image that will be
	// used to virtualize the Build process.
	Image string

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

	Deploy        *deploy.Deploy       `yaml:"deploy,omitempty"`
	Publish       *publish.Publish     `yaml:"publish,omitempty"`
	Notifications *notify.Notification `yaml:"notify,omitempty"`

	// Git specified git-specific parameters, such as
	// the clone depth and path
	Git *git.Git `yaml:"git,omitempty"`
}

// Write adds all the steps to the build script, including
// build commands, deploy and publish commands.
func (b *Build) Write(f *buildfile.Buildfile, r *repo.Repo) {
	// append build commands
	b.WriteBuild(f)

	// write publish commands
	if b.Publish != nil {
		b.Publish.Write(f, r)
	}

	// write deployment commands
	if b.Deploy != nil {
		b.Deploy.Write(f)
	}

	// write exit value
	f.WriteCmd("exit 0")
}

// WriteBuild adds only the build steps to the build script,
// omitting publish and deploy steps. This is important for
// pull requests, where deployment would be undesirable.
func (b *Build) WriteBuild(f *buildfile.Buildfile) {
	// append environment variables
	for _, env := range b.Env {
		parts := strings.Split(env, "=")
		if len(parts) != 2 {
			continue
		}
		f.WriteEnv(parts[0], parts[1])
	}

	// append build commands
	for _, cmd := range b.Script {
		f.WriteCmd(cmd)
	}
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
