package engine

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/samalba/dockerclient"
)

// Agent provides configuration for a build agent container.
type Agent struct {
	Image      string   `yaml:"image,omitempty"`
	Entrypoint []string `yaml:"entrypoint,omitempty"`
	Cmd        []string `yaml:"cmd,omitempty"`
	Env        []string `yaml:"env,omitempty"`
}

// NewAgent parses an build agent DSN and returns a new Agent. The image,
// entrypoint, and cmd arguments will provide defaults if they are not
// set by the DSN.
//
// The DSN string is an inline YAML string, e.g.
//
//    "image: drone/drone-exec"
//
// The following keys are supported:
//
// - image (string): The Docker image
// - entrypoint (string[]): The entrypoint
// - cmd (string[]): The command
// - env (string[]): The environment
func NewAgent(dsn, image string, entrypoint, cmd []string) (*Agent, error) {
	a := &Agent{
		Image:      image,
		Entrypoint: entrypoint,
		Cmd:        cmd,
	}
	err := yaml.Unmarshal([]byte(fmt.Sprintf("{%s}", dsn)), a)
	return a, err
}

// NewDockerConfig create a Docker container configuration object. The
// task will be appended to the Docker command. The env will be appended
// to the default environment.
func (a *Agent) NewContainerConfig(task string, env []string) *dockerclient.ContainerConfig {
	return &dockerclient.ContainerConfig{
		Image:      a.Image,
		Entrypoint: a.Entrypoint,
		Cmd:        append(a.Cmd, "--", task),
		Env:        append(a.Env, env...),
		HostConfig: dockerclient.HostConfig{
			Binds:            []string{"/var/run/docker.sock:/var/run/docker.sock"},
			MemorySwappiness: -1,
		},
		Volumes: map[string]struct{}{
			"/var/run/docker.sock": struct{}{},
		},
	}
}
