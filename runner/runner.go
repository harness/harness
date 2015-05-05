package runner

import (
	"io"

	"github.com/drone/drone/common"
	"github.com/drone/drone/queue"
)

type Runner interface {
	Run(work *queue.Work) error
	Cancel(repo string, build, task int) error
	Logs(repo string, build, task int) (io.ReadCloser, error)
}

// Updater defines a set of functions that are required for
// the runner to sent Drone updates during a build.
type Updater interface {
	SetBuild(*common.Repo, *common.Build) error
	SetTask(*common.Repo, *common.Build, *common.Task) error
	SetLogs(*common.Repo, *common.Build, *common.Task, io.ReadCloser) error
}
