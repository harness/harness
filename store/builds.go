package store

import (
	"github.com/drone/drone/model"
	"golang.org/x/net/context"
)

type BuildStore interface {
	// Get gets a build by unique ID.
	Get(int64) (*model.Build, error)

	// GetNumber gets a build by number.
	GetNumber(*model.Repo, int) (*model.Build, error)

	// GetRef gets a build by its ref.
	GetRef(*model.Repo, string) (*model.Build, error)

	// GetCommit gets a build by its commit sha.
	GetCommit(*model.Repo, string, string) (*model.Build, error)

	// GetLast gets the last build for the branch.
	GetLast(*model.Repo, string) (*model.Build, error)

	// GetLastBefore gets the last build before build number N.
	GetLastBefore(*model.Repo, string, int64) (*model.Build, error)

	// GetList gets a list of builds for the repository
	GetList(*model.Repo) ([]*model.Build, error)

	// Create creates a new build and jobs.
	Create(*model.Build, ...*model.Job) error

	// Update updates a build.
	Update(*model.Build) error
}

func GetBuild(c context.Context, id int64) (*model.Build, error) {
	return FromContext(c).Builds().Get(id)
}

func GetBuildNumber(c context.Context, repo *model.Repo, num int) (*model.Build, error) {
	return FromContext(c).Builds().GetNumber(repo, num)
}

func GetBuildRef(c context.Context, repo *model.Repo, ref string) (*model.Build, error) {
	return FromContext(c).Builds().GetRef(repo, ref)
}

func GetBuildCommit(c context.Context, repo *model.Repo, sha, branch string) (*model.Build, error) {
	return FromContext(c).Builds().GetCommit(repo, sha, branch)
}

func GetBuildLast(c context.Context, repo *model.Repo, branch string) (*model.Build, error) {
	return FromContext(c).Builds().GetLast(repo, branch)
}

func GetBuildLastBefore(c context.Context, repo *model.Repo, branch string, number int64) (*model.Build, error) {
	return FromContext(c).Builds().GetLastBefore(repo, branch, number)
}

func GetBuildList(c context.Context, repo *model.Repo) ([]*model.Build, error) {
	return FromContext(c).Builds().GetList(repo)
}

func CreateBuild(c context.Context, build *model.Build, jobs ...*model.Job) error {
	return FromContext(c).Builds().Create(build, jobs...)
}

func UpdateBuild(c context.Context, build *model.Build) error {
	return FromContext(c).Builds().Update(build)
}
