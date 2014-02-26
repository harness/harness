package queue

import (
	"io"
	"time"

	"github.com/drone/drone/pkg/build"
	"github.com/drone/drone/pkg/build/docker"
	"github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"
)

type Runner interface {
	Run(buildScript *script.Build, repo *repo.Repo, key []byte, buildOutput io.Writer) (success bool, err error)
}

type runner struct {
	dockerClient *docker.Client
	timeout      time.Duration
}

func NewRunner(dockerClient *docker.Client, timeout time.Duration) Runner {
	return &runner{
		dockerClient: dockerClient,
		timeout:      timeout,
	}
}

func (r *runner) Run(buildScript *script.Build, repo *repo.Repo, key []byte, buildOutput io.Writer) (bool, error) {
	builder := build.New(r.dockerClient)
	builder.Build = buildScript
	builder.Repo = repo
	builder.Key = key
	builder.Stdout = buildOutput
	builder.Timeout = r.timeout

	err := builder.Run()

	return builder.BuildState == nil || builder.BuildState.ExitCode != 0, err
}
