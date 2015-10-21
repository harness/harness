package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type JobStore interface {
	// Get gets a job by unique ID.
	Get(int64) (*model.Job, error)

	// GetNumber gets a job by number.
	GetNumber(*model.Build, int) (*model.Job, error)

	// GetList gets a list of all users in the system.
	GetList(*model.Build) ([]*model.Job, error)

	// Create creates a job.
	Create(*model.Job) error

	// Update updates a job.
	Update(*model.Job) error
}

func GetJob(c context.Context, id int64) (*model.Job, error) {
	return FromContext(c).Jobs().Get(id)
}

func GetJobNumber(c context.Context, build *model.Build, num int) (*model.Job, error) {
	return FromContext(c).Jobs().GetNumber(build, num)
}

func GetJobList(c context.Context, build *model.Build) ([]*model.Job, error) {
	return FromContext(c).Jobs().GetList(build)
}

func CreateJob(c context.Context, job *model.Job) error {
	return FromContext(c).Jobs().Create(job)
}

func UpdateJob(c context.Context, job *model.Job) error {
	return FromContext(c).Jobs().Update(job)
}
