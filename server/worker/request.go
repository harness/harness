package worker

import (
	"github.com/drone/drone/shared/model"
)

type Request struct {
	User   *model.User
	Repo   *model.Repo
	Commit *model.Commit
	server *model.Server
}
