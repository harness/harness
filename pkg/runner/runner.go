package runner

import (
	"io"

	"github.com/drone/drone/pkg/queue"
	"github.com/drone/drone/pkg/types"
)

type Runner interface {
	Run(work *queue.Work) error
	Cancel(*types.Job) error
	Logs(*types.Job) (io.ReadCloser, error)
}

// Updater defines a set of functions that are required for
// the runner to sent Drone updates during a build.
type Updater interface {
	SetCommit(*types.User, *types.Repo, *types.Commit) error
	SetJob(*types.Repo, *types.Commit, *types.Job) error
	SetLogs(*types.Repo, *types.Commit, *types.Job, io.ReadCloser) error
}
