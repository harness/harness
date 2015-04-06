package datastore

import (
	"code.google.com/p/go.net/context"
	"github.com/drone/drone/shared/model"
)

type Repostore interface {
	// GetRepo retrieves a specific repo from the
	// datastore for the given ID.
	GetRepo(id int64) (*model.Repo, error)

	// GetRepoName retrieves a repo from the datastore
	// for the specified remote, owner and name.
	GetRepoName(remote, owner, name string) (*model.Repo, error)

	// GetRepoList retrieves a list of all repos from
	// the datastore accessible by the given user ID.
	GetRepoList(user *model.User) ([]*model.Repo, error)

	// PostRepo saves a repo in the datastore.
	PostRepo(repo *model.Repo) error

	// PutRepo saves a repo in the datastore.
	PutRepo(repo *model.Repo) error

	// DelRepo removes the repo from the datastore.
	DelRepo(repo *model.Repo) error
}

// GetRepo retrieves a specific repo from the
// datastore for the given ID.
func GetRepo(c context.Context, id int64) (*model.Repo, error) {
	return FromContext(c).GetRepo(id)
}

// GetRepoName retrieves a repo from the datastore
// for the specified remote, owner and name.
func GetRepoName(c context.Context, remote, owner, name string) (*model.Repo, error) {
	return FromContext(c).GetRepoName(remote, owner, name)
}

// GetRepoList retrieves a list of all repos from
// the datastore accessible by the given user ID.
func GetRepoList(c context.Context, user *model.User) ([]*model.Repo, error) {
	return FromContext(c).GetRepoList(user)
}

// PostRepo saves a repo in the datastore.
func PostRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).PostRepo(repo)
}

// PutRepo saves a repo in the datastore.
func PutRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).PutRepo(repo)
}

// DelRepo removes the repo from the datastore.
func DelRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).DelRepo(repo)
}
