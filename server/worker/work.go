package worker

import "github.com/drone/drone/shared/model"

type Work struct {
	Host   string        `json:"host"`
	User   *model.User   `json:"user"`
	Repo   *model.Repo   `json:"repo"`
	Commit *model.Commit `json:"commit"`
}

type Assignment struct {
	Work   *Work
	Worker Worker
}
