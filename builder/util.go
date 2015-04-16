package builder

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/drone/drone/common"
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
func injectEnv(b *B, conf *dockerclient.ContainerConfig) {
	var branch string
	var commit string
	if b.Build.Commit != nil {
		branch = b.Build.Commit.Ref
		commit = b.Build.Commit.Sha
	} else {
		branch = b.Build.PullRequest.Target.Ref
		commit = b.Build.PullRequest.Target.Sha
	}

	conf.Env = append(conf.Env, "DRONE=true")
	conf.Env = append(conf.Env, fmt.Sprintf("DRONE_BRANCH=%s", branch))
	conf.Env = append(conf.Env, fmt.Sprintf("DRONE_COMMIT=%s", commit))

	// for jenkins campatibility
	conf.Env = append(conf.Env, "CI=true")
	conf.Env = append(conf.Env, fmt.Sprintf("WORKSPACE=%s", b.Clone.Dir))
	conf.Env = append(conf.Env, fmt.Sprintf("JOB_NAME=%s/%s", b.Repo.Owner, b.Repo.Name))
	conf.Env = append(conf.Env, fmt.Sprintf("BUILD_ID=%d", b.Build.Number))
	conf.Env = append(conf.Env, fmt.Sprintf("BUILD_DIR=%s", b.Clone.Dir))
	conf.Env = append(conf.Env, fmt.Sprintf("GIT_BRANCH=%s", branch))
	conf.Env = append(conf.Env, fmt.Sprintf("GIT_COMMIT=%s", commit))

}

// helper function to encode the build step to
// a json string. Primarily used for plugins, which
// expect a json encoded string in stdin or arg[1].
func toCommand(b *B, step *common.Step) []string {
	p := payload{
		b.Repo,
		b.Build,
		b.Task,
		b.Clone,
		step.Config,
	}
	return []string{p.Encode()}
}

// payload represents the payload of a plugin
// that is serialized and sent to the plugin in JSON
// format via stdin or arg[1].
type payload struct {
	Repo  *common.Repo  `json:"repo"`
	Build *common.Build `json:"build"`
	Task  *common.Task  `json:"task"`
	Clone *common.Clone `json:"clone"`

	Config map[string]interface{} `json:"vargs"`
}

// Encode encodes the payload in JSON format.
func (p *payload) Encode() string {
	out, _ := json.Marshal(p)
	return string(out)
}
