package worker

import "github.com/drone/drone/shared/model"

type Work struct {
	User   *model.User
	Repo   *model.Repo
	Commit *model.Commit
}

type Assignment struct {
	Work   *Work
	Worker Worker
}
