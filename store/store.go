package store

import (
	"io"

	"github.com/drone/drone/model"

	"golang.org/x/net/context"
)

type Store interface {
	// GetUser gets a user by unique ID.
	GetUser(int64) (*model.User, error)

	// GetUserLogin gets a user by unique Login name.
	GetUserLogin(string) (*model.User, error)

	// GetUserList gets a list of all users in the system.
	GetUserList() ([]*model.User, error)

	// GetUserFeed gets a user activity feed.
	GetUserFeed([]*model.RepoLite) ([]*model.Feed, error)

	// GetUserFeedLatest gets a user activity feed for all repositories including
	// only the latest build for each repository.
	GetUserFeedLatest(listof []*model.RepoLite) ([]*model.Feed, error)

	// GetUserCount gets a count of all users in the system.
	GetUserCount() (int, error)

	// CreateUser creates a new user account.
	CreateUser(*model.User) error

	// UpdateUser updates a user account.
	UpdateUser(*model.User) error

	// DeleteUser deletes a user account.
	DeleteUser(*model.User) error

	// GetRepo gets a repo by unique ID.
	GetRepo(int64) (*model.Repo, error)

	// GetRepoName gets a repo by its full name.
	GetRepoName(string) (*model.Repo, error)

	// GetRepoListOf gets the list of enumerated repos in the system.
	GetRepoListOf([]*model.RepoLite) ([]*model.Repo, error)

	// GetRepoCount gets a count of all repositories in the system.
	GetRepoCount() (int, error)

	// CreateRepo creates a new repository.
	CreateRepo(*model.Repo) error

	// UpdateRepo updates a user repository.
	UpdateRepo(*model.Repo) error

	// DeleteRepo deletes a user repository.
	DeleteRepo(*model.Repo) error

	// GetBuild gets a build by unique ID.
	GetBuild(int64) (*model.Build, error)

	// GetBuildNumber gets a build by number.
	GetBuildNumber(*model.Repo, int) (*model.Build, error)

	// GetBuildRef gets a build by its ref.
	GetBuildRef(*model.Repo, string) (*model.Build, error)

	// GetBuildCommit gets a build by its commit sha.
	GetBuildCommit(*model.Repo, string, string) (*model.Build, error)

	// GetBuildLast gets the last build for the branch.
	GetBuildLast(*model.Repo, string) (*model.Build, error)

	// GetBuildLastBefore gets the last build before build number N.
	GetBuildLastBefore(*model.Repo, string, int64) (*model.Build, error)

	// GetBuildList gets a list of builds for the repository
	GetBuildList(*model.Repo) ([]*model.Build, error)

	// GetBuildQueue gets a list of build in queue.
	GetBuildQueue() ([]*model.Feed, error)

	// CreateBuild creates a new build and jobs.
	CreateBuild(*model.Build, ...*model.Proc) error

	// UpdateBuild updates a build.
	UpdateBuild(*model.Build) error

	//
	// new functions
	//

	ConfigLoad(int64) (*model.Config, error)
	ConfigFind(*model.Repo, string) (*model.Config, error)
	ConfigFindApproved(*model.Config) (bool, error)
	ConfigCreate(*model.Config) error

	SenderFind(*model.Repo, string) (*model.Sender, error)
	SenderList(*model.Repo) ([]*model.Sender, error)
	SenderCreate(*model.Sender) error
	SenderUpdate(*model.Sender) error
	SenderDelete(*model.Sender) error

	SecretFind(*model.Repo, string) (*model.Secret, error)
	SecretList(*model.Repo) ([]*model.Secret, error)
	SecretCreate(*model.Secret) error
	SecretUpdate(*model.Secret) error
	SecretDelete(*model.Secret) error

	RegistryFind(*model.Repo, string) (*model.Registry, error)
	RegistryList(*model.Repo) ([]*model.Registry, error)
	RegistryCreate(*model.Registry) error
	RegistryUpdate(*model.Registry) error
	RegistryDelete(*model.Registry) error

	ProcLoad(int64) (*model.Proc, error)
	ProcFind(*model.Build, int) (*model.Proc, error)
	ProcChild(*model.Build, int, string) (*model.Proc, error)
	ProcList(*model.Build) ([]*model.Proc, error)
	ProcCreate([]*model.Proc) error
	ProcUpdate(*model.Proc) error
	ProcClear(*model.Build) error

	LogFind(*model.Proc) (io.ReadCloser, error)
	LogSave(*model.Proc, io.Reader) error

	FileList(*model.Build) ([]*model.File, error)
	FileFind(*model.Proc, string) (*model.File, error)
	FileRead(*model.Proc, string) (io.ReadCloser, error)
	FileCreate(*model.File, io.Reader) error

	TaskList() ([]*model.Task, error)
	TaskInsert(*model.Task) error
	TaskDelete(string) error
}

// GetUser gets a user by unique ID.
func GetUser(c context.Context, id int64) (*model.User, error) {
	return FromContext(c).GetUser(id)
}

// GetUserLogin gets a user by unique Login name.
func GetUserLogin(c context.Context, login string) (*model.User, error) {
	return FromContext(c).GetUserLogin(login)
}

// GetUserList gets a list of all users in the system.
func GetUserList(c context.Context) ([]*model.User, error) {
	return FromContext(c).GetUserList()
}

// GetUserFeed gets a user activity feed.
func GetUserFeed(c context.Context, listof []*model.RepoLite, latest bool) ([]*model.Feed, error) {
	if latest {
		return FromContext(c).GetUserFeedLatest(listof)
	}
	return FromContext(c).GetUserFeed(listof)
}

// GetUserCount gets a count of all users in the system.
func GetUserCount(c context.Context) (int, error) {
	return FromContext(c).GetUserCount()
}

func CreateUser(c context.Context, user *model.User) error {
	return FromContext(c).CreateUser(user)
}

func UpdateUser(c context.Context, user *model.User) error {
	return FromContext(c).UpdateUser(user)
}

func DeleteUser(c context.Context, user *model.User) error {
	return FromContext(c).DeleteUser(user)
}

func GetRepo(c context.Context, id int64) (*model.Repo, error) {
	return FromContext(c).GetRepo(id)
}

func GetRepoName(c context.Context, name string) (*model.Repo, error) {
	return FromContext(c).GetRepoName(name)
}

func GetRepoOwnerName(c context.Context, owner, name string) (*model.Repo, error) {
	return FromContext(c).GetRepoName(owner + "/" + name)
}

func GetRepoListOf(c context.Context, listof []*model.RepoLite) ([]*model.Repo, error) {
	return FromContext(c).GetRepoListOf(listof)
}

func CreateRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).CreateRepo(repo)
}

func UpdateRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).UpdateRepo(repo)
}

func DeleteRepo(c context.Context, repo *model.Repo) error {
	return FromContext(c).DeleteRepo(repo)
}

func GetBuild(c context.Context, id int64) (*model.Build, error) {
	return FromContext(c).GetBuild(id)
}

func GetBuildNumber(c context.Context, repo *model.Repo, num int) (*model.Build, error) {
	return FromContext(c).GetBuildNumber(repo, num)
}

func GetBuildRef(c context.Context, repo *model.Repo, ref string) (*model.Build, error) {
	return FromContext(c).GetBuildRef(repo, ref)
}

func GetBuildCommit(c context.Context, repo *model.Repo, sha, branch string) (*model.Build, error) {
	return FromContext(c).GetBuildCommit(repo, sha, branch)
}

func GetBuildLast(c context.Context, repo *model.Repo, branch string) (*model.Build, error) {
	return FromContext(c).GetBuildLast(repo, branch)
}

func GetBuildLastBefore(c context.Context, repo *model.Repo, branch string, number int64) (*model.Build, error) {
	return FromContext(c).GetBuildLastBefore(repo, branch, number)
}

func GetBuildList(c context.Context, repo *model.Repo) ([]*model.Build, error) {
	return FromContext(c).GetBuildList(repo)
}

func GetBuildQueue(c context.Context) ([]*model.Feed, error) {
	return FromContext(c).GetBuildQueue()
}

func CreateBuild(c context.Context, build *model.Build, procs ...*model.Proc) error {
	return FromContext(c).CreateBuild(build, procs...)
}

func UpdateBuild(c context.Context, build *model.Build) error {
	return FromContext(c).UpdateBuild(build)
}
