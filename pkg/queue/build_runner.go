package queue

import (
	"io"
	"time"

	"github.com/drone/drone/pkg/build"
	"github.com/drone/drone/pkg/build/docker"
	"github.com/drone/drone/pkg/build/repo"
	"github.com/drone/drone/pkg/build/script"
)

type BuildRunner interface {
	Run(buildScript *script.Build, repo *repo.Repo, key []byte, buildOutput io.Writer) (success bool, err error)
}

type buildRunner struct {
	dockerClient *docker.Client
	timeout      time.Duration
}

func NewBuildRunner(dockerClient *docker.Client, timeout time.Duration) BuildRunner {
	return &buildRunner{
		dockerClient: dockerClient,
		timeout:      timeout,
	}
}

func (runner *buildRunner) Run(buildScript *script.Build, repo *repo.Repo, key []byte, buildOutput io.Writer) (bool, error) {
	builder := build.New(runner.dockerClient)
	builder.Build = buildScript
	builder.Repo = repo
	builder.Key = key
	builder.Stdout = buildOutput
	builder.Timeout = runner.timeout

	err := builder.Run()

	return builder.BuildState == nil || builder.BuildState.ExitCode != 0, err
}
