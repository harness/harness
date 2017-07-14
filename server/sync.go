package server

import (
	"time"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/store"
)

// Syncer synces the user repository and permissions.
type Syncer interface {
	Sync(user *model.User) error
}

type syncer struct {
	remote remote.Remote
	store  store.Store
	perms  model.PermStore
}

func (s *syncer) Sync(user *model.User) error {
	unix := time.Now().Unix()
	repos, err := s.remote.Repos(user)
	if err != nil {
		return err
	}

	var perms []*model.Perm
	for _, repo := range repos {
		perm := model.Perm{
			UserID: user.ID,
			Repo:   repo.FullName,
			Pull:   true,
			Synced: unix,
		}
		if repo.Perm != nil {
			perm.Push = repo.Perm.Push
			perm.Admin = repo.Perm.Admin
		}
		perms = append(perms, &perm)
	}

	err = s.store.RepoBatch(repos)
	if err != nil {
		return err
	}

	err = s.store.PermBatch(perms)
	if err != nil {
		return err
	}

	return nil
}
