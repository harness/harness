package worker

import (
	"github.com/drone/drone/shared/model"
)

type Request struct {
	User   *model.User   `json:"-"`
	Repo   *model.Repo   `json:"repo"`
	Commit *model.Commit `json:"commit"`
	server *model.Server
}
