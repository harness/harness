package main

import (
	"encoding/json"
	"strconv"
	"strings"

	common "github.com/drone/drone/pkg/types"
	"github.com/samalba/dockerclient"
)

// helper function that converts the build step to
// a containerConfig for use with the dockerclient
func toContainerConfig(step *common.Step) *dockerclient.ContainerConfig {
	config := &dockerclient.ContainerConfig{
		Image:      step.Image,
		Env:        step.Environment,
		Cmd:        step.Command,
		Entrypoint: step.Entrypoint,
		WorkingDir: step.WorkingDir,
		HostConfig: dockerclient.HostConfig{
			Privileged:  step.Privileged,
			NetworkMode: step.NetworkMode,
		},
	}

	if len(config.Entrypoint) == 0 {
		config.Entrypoint = nil
	}

	config.Volumes = map[string]struct{}{}
	for _, path := range step.Volumes {
		if strings.Index(path, ":") == -1 {
			continue
		}
		parts := strings.Split(path, ":")
		config.Volumes[parts[1]] = struct{}{}
		config.HostConfig.Binds = append(config.HostConfig.Binds, path)
	}

	return config
}

// helper function to inject drone-specific environment
// variables into the container.
func toEnv(c *Context) map[string]string {
	return map[string]string{
		"CI":           "true",
		"BUILD_DIR":    c.Clone.Dir,
		"BUILD_ID":     strconv.Itoa(c.Commit.Sequence),
		"BUILD_NUMBER": strconv.Itoa(c.Commit.Sequence),
		"JOB_NAME":     c.Repo.FullName,
		"WORKSPACE":    c.Clone.Dir,
		"GIT_BRANCH":   c.Clone.Branch,
		"GIT_COMMIT":   c.Clone.Sha,

		"DRONE":        "true",
		"DRONE_REPO":   c.Repo.FullName,
		"DRONE_BUILD":  strconv.Itoa(c.Commit.Sequence),
		"DRONE_BRANCH": c.Clone.Branch,
		"DRONE_COMMIT": c.Clone.Sha,
		"DRONE_DIR":    c.Clone.Dir,
	}
}

// helper function to encode the build step to
// a json string. Primarily used for plugins, which
// expect a json encoded string in stdin or arg[1].
func toCommand(c *Context, step *common.Step) []string {
	p := payload{
		c.Repo,
		c.Commit,
		c.Build,
		c.Clone,
		step.Config,
	}
	return []string{p.Encode()}
}

// payload represents the payload of a plugin
// that is serialized and sent to the plugin in JSON
// format via stdin or arg[1].
type payload struct {
	Repo   *common.Repo   `json:"repo"`
	Commit *common.Commit `json:"commit"`
	Build  *common.Build  `json:"build"`
	Clone  *common.Clone  `json:"clone"`

	Config map[string]interface{} `json:"vargs"`
}

// Encode encodes the payload in JSON format.
func (p *payload) Encode() string {
	out, _ := json.Marshal(p)
	return string(out)
}
