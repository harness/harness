package datastore

import (
	"io"

	"github.com/drone/drone/common"
)

type Datastore interface {
	// GetUser gets a user by user login.
	GetUser(string) (*common.User, error)

	// GetUserTokens gets a list of all tokens for
	// the given user login.
	GetUserTokens(string) ([]*common.Token, error)

	// GetUserRepos gets a list of repositories for the
	// given user account.
	GetUserRepos(string) ([]*common.Repo, error)

	// GetUserCount gets a count of all registered users
	// in the system.
	GetUserCount() (int, error)

	// GetUserList gets a list of all registered users.
	GetUserList() ([]*common.User, error)

	// UpdateUser updates an existing user. If the user
	// does not exists an error is returned.
	UpdateUser(*common.User) error

	// InsertUser inserts a new user into the datastore. If
	// the user login already exists an error is returned.
	InsertUser(*common.User) error

	// DeleteUser deletes the user.
	DeleteUser(*common.User) error

	// GetToken gets a token by sha value.
	GetToken(string, string) (*common.Token, error)

	// InsertToken inserts a new user token in the datastore.
	// If the token already exists and error is returned.
	InsertToken(*common.Token) error

	// DeleteUser deletes the token.
	DeleteToken(*common.Token) error

	// GetSubscriber gets the subscriber by login for the
	// named repository.
	GetSubscriber(string, string) (*common.Subscriber, error)

	// InsertSubscriber inserts a subscriber for the named
	// repository.
	InsertSubscriber(string, string) error

	// DeleteSubscriber removes the subscriber by login for the
	// named repository.
	DeleteSubscriber(string, string) error

	// GetRepo gets the repository by name.
	GetRepo(string) (*common.Repo, error)

	// GetRepoParams gets the private environment parameters
	// for the given repository.
	GetRepoParams(string) (map[string]string, error)

	// GetRepoParams gets the private and public rsa keys
	// for the given repository.
	GetRepoKeys(string) (*common.Keypair, error)

	// UpdateRepos updates a repository. If the repository
	// does not exist an error is returned.
	UpdateRepo(*common.Repo) error

	// InsertRepo inserts a repository in the datastore and
	// subscribes the user to that repository.
	InsertRepo(*common.User, *common.Repo) error

	// UpsertRepoParams inserts or updates the private
	// environment parameters for the named repository.
	UpsertRepoParams(string, map[string]string) error

	// UpsertRepoKeys inserts or updates the private and
	// public keypair for the named repository.
	UpsertRepoKeys(string, *common.Keypair) error

	// DeleteRepo deletes the repository.
	DeleteRepo(*common.Repo) error

	// GetBuild gets the specified build number for the
	// named repository and build number
	GetBuild(string, int) (*common.Build, error)

	// GetBuildList gets a list of recent builds for the
	// named repository.
	GetBuildList(string) ([]*common.Build, error)

	// GetBuildLast gets the last executed build for the
	// named repository.
	GetBuildLast(string) (*common.Build, error)

	// GetBuildStatus gets the named build status for the
	// named repository and build number.
	GetBuildStatus(string, int, string) (*common.Status, error)

	// GetBuildStatusList gets a list of all build statues for
	// the named repository and build number.
	GetBuildStatusList(string, int) ([]*common.Status, error)

	// InsertBuild inserts a new build for the named repository
	InsertBuild(string, *common.Build) error

	// InsertBuildStatus inserts a new build status for the
	// named repository and build number. If the status already
	// exists an error is returned.
	InsertBuildStatus(string, int, *common.Status) error

	// UpdateBuild updates an existing build for the named
	// repository. If the build already exists and error is
	// returned.
	UpdateBuild(string, *common.Build) error

	// GetTask gets the task at index N for the named
	// repository and build number.
	GetTask(string, int, int) (*common.Task, error)

	// GetTaskLogs gets the task logs at index N for
	// the named repository and build number.
	GetTaskLogs(string, int, int) (io.Reader, error)

	// GetTaskList gets all tasks for the named repository
	// and build number.
	GetTaskList(string, int) ([]*common.Task, error)

	// UpsertTask inserts or updates a task for the named
	// repository and build number.
	UpsertTask(string, int, *common.Task) error

	// UpsertTaskLogs inserts or updates a task logs for the
	// named repository and build number.
	UpsertTaskLogs(string, int, int, []byte) error
}
