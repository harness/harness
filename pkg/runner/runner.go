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
	SetBuild(*types.User, *types.Repo, *types.Build) error
	SetJob(*types.Repo, *types.Build, *types.Job) error
	SetLogs(*types.Repo, *types.Build, *types.Job, io.ReadCloser) error
}
