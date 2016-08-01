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

	// GetSecretList gets a list of repository secrets
	GetSecretList(*model.Repo) ([]*model.RepoSecret, error)

	// GetSecret gets the named repository secret.
	GetSecret(*model.Repo, string) (*model.RepoSecret, error)

	// SetSecret sets the named repository secret.
	SetSecret(*model.RepoSecret) error

	// DeleteSecret deletes the named repository secret.
	DeleteSecret(*model.RepoSecret) error

	// GetTeamSecretList gets a list of team secrets
	GetTeamSecretList(string) ([]*model.TeamSecret, error)

	// GetTeamSecret gets the named team secret.
	GetTeamSecret(string, string) (*model.TeamSecret, error)

	// SetTeamSecret sets the named team secret.
	SetTeamSecret(*model.TeamSecret) error

	// DeleteTeamSecret deletes the named team secret.
	DeleteTeamSecret(*model.TeamSecret) error

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

	GetAgent(int64) (*model.Agent, error)

	GetAgentAddr(string) (*model.Agent, error)

	GetAgentList() ([]*model.Agent, error)

	CreateAgent(*model.Agent) error

	UpdateAgent(*model.Agent) error

	DeleteAgent(*model.Agent) error
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

func GetSecretList(c context.Context, r *model.Repo) ([]*model.RepoSecret, error) {
	return FromContext(c).GetSecretList(r)
}

func GetSecret(c context.Context, r *model.Repo, name string) (*model.RepoSecret, error) {
	return FromContext(c).GetSecret(r, name)
}

func SetSecret(c context.Context, s *model.RepoSecret) error {
	return FromContext(c).SetSecret(s)
}

func DeleteSecret(c context.Context, s *model.RepoSecret) error {
	return FromContext(c).DeleteSecret(s)
}

func GetTeamSecretList(c context.Context, team string) ([]*model.TeamSecret, error) {
	return FromContext(c).GetTeamSecretList(team)
}

func GetTeamSecret(c context.Context, team, name string) (*model.TeamSecret, error) {
	return FromContext(c).GetTeamSecret(team, name)
}

func SetTeamSecret(c context.Context, s *model.TeamSecret) error {
	return FromContext(c).SetTeamSecret(s)
}

func DeleteTeamSecret(c context.Context, s *model.TeamSecret) error {
	return FromContext(c).DeleteTeamSecret(s)
}

func GetMergedSecretList(c context.Context, r *model.Repo) ([]*model.Secret, error) {
	var (
		secrets []*model.Secret
	)

	repoSecs, err := FromContext(c).GetSecretList(r)

	if err != nil {
		return nil, err
	}

	for _, secret := range repoSecs {
		secrets = append(secrets, secret.Secret())
	}

	teamSecs, err := FromContext(c).GetTeamSecretList(r.Owner)

	if err != nil {
		return nil, err
	}

	for _, secret := range teamSecs {
		secrets = append(secrets, secret.Secret())
	}

	return secrets, nil
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

func CreateBuild(c context.Context, build *model.Build, jobs ...*model.Job) error {
	return FromContext(c).CreateBuild(build, jobs...)
}

func UpdateBuild(c context.Context, build *model.Build) error {
	return FromContext(c).UpdateBuild(build)
}

func UpdateBuildJob(c context.Context, build *model.Build, job *model.Job) (bool, error) {
	if err := UpdateJob(c, job); err != nil {
		return false, err
	}

	// if the job is running or started we don't need to update the build
	// status since.
	if job.Status == model.StatusRunning || job.Status == model.StatusPending {
		return false, nil
	}

	jobs, err := GetJobList(c, build)
	if err != nil {
		return false, err
	}
	// check to see if all jobs are finished for this build. If yes, we need to
	// calcualte the overall build status and finish time.
	status := model.StatusSuccess
	finish := job.Finished
	for _, job := range jobs {
		if job.Finished > finish {
			finish = job.Finished
		}
		switch job.Status {
		case model.StatusSuccess:
			// no-op
		case model.StatusRunning, model.StatusPending:
			return false, nil
		default:
			status = job.Status
		}
	}

	build.Status = status
	build.Finished = finish
	if err := FromContext(c).UpdateBuild(build); err != nil {
		return false, err
	}
	return true, nil
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

func GetAgent(c context.Context, id int64) (*model.Agent, error) {
	return FromContext(c).GetAgent(id)
}

func GetAgentAddr(c context.Context, addr string) (*model.Agent, error) {
	return FromContext(c).GetAgentAddr(addr)
}

func GetAgentList(c context.Context) ([]*model.Agent, error) {
	return FromContext(c).GetAgentList()
}

func CreateAgent(c context.Context, agent *model.Agent) error {
	return FromContext(c).CreateAgent(agent)
}

func UpdateAgent(c context.Context, agent *model.Agent) error {
	return FromContext(c).UpdateAgent(agent)
}

func DeleteAgent(c context.Context, agent *model.Agent) error {
	return FromContext(c).DeleteAgent(agent)
}
