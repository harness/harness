package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type RepoStore interface {
	// Get gets a repo by unique ID.
	Get(int64) (*model.Repo, error)

	// GetLogin gets a repo by its full name.
	GetName(string) (*model.Repo, error)

	// GetListOf gets the list of enumerated repos in the system.
	GetListOf([]*model.RepoLite) ([]*model.Repo, error)

	// Count gets a count of all repos in the system.
	Count() (int, error)

	// Create creates a new repository.
	Create(*model.Repo) error

	// Update updates a user repository.
	Update(*model.Repo) error

	// Delete deletes a user repository.
	Delete(*model.Repo) error
}

func GetRepo(c context.Context, id int64) (*model.Repo, error) {
	return FromContext(c).Repos().Get(id)
}

func GetRepoName(c context.Context, name string) (*model.Repo, error) {
	return FromContext(c).Repos().GetName(name)
}

func GetRepoOwnerName(c context.Context, owner, name string) (*model.Repo, error) {
	return FromContext(c).Repos().GetName(owner + "/" + name)
}

func GetRepoListOf(c context.Context, listof []*model.RepoLite) ([]*model.Repo, error) {
	return FromContext(c).Repos().GetListOf(listof)
}

func CountRepos(c context.Context) (int, error) {
	return FromContext(c).Repos().Count()
}

func CreateRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).Repos().Create(repo)
}

func UpdateRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).Repos().Update(repo)
}

func DeleteRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).Repos().Delete(repo)
}
