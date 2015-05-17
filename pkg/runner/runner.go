package runner

import (
	"io"

	"github.com/drone/drone/pkg/queue"
	common "github.com/drone/drone/pkg/types"
)

type Runner interface {
	Run(work *queue.Work) error
	Cancel(*common.Build) error
	Logs(*common.Build) (io.ReadCloser, error)
}

// Updater defines a set of functions that are required for
// the runner to sent Drone updates during a build.
type Updater interface {
	SetCommit(*common.User, *common.Repo, *common.Commit) error
	SetBuild(*common.Repo, *common.Commit, *common.Build) error
	SetLogs(*common.Repo, *common.Commit, *common.Build, io.ReadCloser) error
}
