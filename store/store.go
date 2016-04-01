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

	// GetKey gets a key by unique repository ID.
	GetKey(*model.Repo) (*model.Key, error)

	// CreateKey creates a new key.
	CreateKey(*model.Key) error

	// UpdateKey updates a user key.
	UpdateKey(*model.Key) error

	// DeleteKey deletes a user key.
	DeleteKey(*model.Key) error

	// GetSecretList gets a list of repository secrets
	GetSecretList(*model.Repo) ([]*model.Secret, error)

	// GetSecret gets the named repository secret.
	GetSecret(*model.Repo, string) (*model.Secret, error)

	// SetSecret sets the named repository secret.
	SetSecret(*model.Secret) error

	// DeleteSecret deletes the named repository secret.
	DeleteSecret(*model.Secret) error

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

	// CreateBuild creates a new build and jobs.
	CreateBuild(*model.Build, ...*model.Job) error

	// UpdateBuild updates a build.
	UpdateBuild(*model.Build) error

	// GetJob gets a job by unique ID.
	GetJob(int64) (*model.Job, error)

	// GetJobNumber gets a job by number.
	GetJobNumber(*model.Build, int) (*model.Job, error)

	// GetJobList gets a list of all users in the system.
	GetJobList(*model.Build) ([]*model.Job, error)

	// CreateJob creates a job.
	CreateJob(*model.Job) error

	// UpdateJob updates a job.
	UpdateJob(*model.Job) error

	// ReadLog reads the Job logs from the datastore.
	ReadLog(*model.Job) (io.ReadCloser, error)

	// WriteLog writes the job logs to the datastore.
	WriteLog(*model.Job, io.Reader) error

	// GetNode gets a build node from the datastore.
	GetNode(id int64) (*model.Node, error)

	// GetNodeList gets a build node list from the datastore.
	GetNodeList() ([]*model.Node, error)

	// CreateNode add a new build node to the datastore.
	CreateNode(*model.Node) error

	// UpdateNode updates a build node in the datastore.
	UpdateNode(*model.Node) error

	// DeleteNode removes a build node from the datastore.
	DeleteNode(*model.Node) error
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
func GetUserFeed(c context.Context, listof []*model.RepoLite) ([]*model.Feed, error) {
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

func GetKey(c context.Context, repo *model.Repo) (*model.Key, error) {
	return FromContext(c).GetKey(repo)
}

func CreateKey(c context.Context, key *model.Key) error {
	return FromContext(c).CreateKey(key)
}

func UpdateKey(c context.Context, key *model.Key) error {
	return FromContext(c).UpdateKey(key)
}

func DeleteKey(c context.Context, key *model.Key) error {
	return FromContext(c).DeleteKey(key)
}

func GetSecretList(c context.Context, r *model.Repo) ([]*model.Secret, error) {
	return FromContext(c).GetSecretList(r)
}

func GetSecret(c context.Context, r *model.Repo, name string) (*model.Secret, error) {
	return FromContext(c).GetSecret(r, name)
}

func SetSecret(c context.Context, s *model.Secret) error {
	return FromContext(c).SetSecret(s)
}

func DeleteSecret(c context.Context, s *model.Secret) error {
	return FromContext(c).DeleteSecret(s)
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

func CreateBuild(c context.Context, build *model.Build, jobs ...*model.Job) error {
	return FromContext(c).CreateBuild(build, jobs...)
}

func UpdateBuild(c context.Context, build *model.Build) error {
	return FromContext(c).UpdateBuild(build)
}

func GetJob(c context.Context, id int64) (*model.Job, error) {
	return FromContext(c).GetJob(id)
}

func GetJobNumber(c context.Context, build *model.Build, num int) (*model.Job, error) {
	return FromContext(c).GetJobNumber(build, num)
}

func GetJobList(c context.Context, build *model.Build) ([]*model.Job, error) {
	return FromContext(c).GetJobList(build)
}

func CreateJob(c context.Context, job *model.Job) error {
	return FromContext(c).CreateJob(job)
}

func UpdateJob(c context.Context, job *model.Job) error {
	return FromContext(c).UpdateJob(job)
}

func ReadLog(c context.Context, job *model.Job) (io.ReadCloser, error) {
	return FromContext(c).ReadLog(job)
}

func WriteLog(c context.Context, job *model.Job, r io.Reader) error {
	return FromContext(c).WriteLog(job, r)
}

func GetNode(c context.Context, id int64) (*model.Node, error) {
	return FromContext(c).GetNode(id)
}

func GetNodeList(c context.Context) ([]*model.Node, error) {
	return FromContext(c).GetNodeList()
}

func CreateNode(c context.Context, node *model.Node) error {
	return FromContext(c).CreateNode(node)
}

func UpdateNode(c context.Context, node *model.Node) error {
	return FromContext(c).UpdateNode(node)
}

func DeleteNode(c context.Context, node *model.Node) error {
	return FromContext(c).DeleteNode(node)
}
