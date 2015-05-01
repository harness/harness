package datastore

import (
	"errors"
	"io"

	"github.com/drone/drone/common"
)

var (
	ErrConflict    = errors.New("Key not unique")
	ErrKeyNotFound = errors.New("Key not found")
)

type Datastore interface {
	// User returns a user by user login.
	User(string) (*common.User, error)

	// UserCount returns a count of all registered users.
	UserCount() (int, error)

	// UserList returns a list of all registered users.
	UserList() ([]*common.User, error)

	// SetUser inserts or updates a user.
	SetUser(*common.User) error

	// SetUserNotExists inserts a new user into the datastore.
	// If the user login already exists ErrConflict is returned.
	SetUserNotExists(*common.User) error

	// Del deletes the user.
	DelUser(*common.User) error

	// Token returns the token for the given user and label.
	Token(string, string) (*common.Token, error)

	// TokenList returns a list of all tokens for the given
	// user login.
	TokenList(string) ([]*common.Token, error)

	// SetToken inserts a new user token in the datastore.
	SetToken(*common.Token) error

	// DelToken deletes the token.
	DelToken(*common.Token) error

	// Subscribed returns true if the user is subscribed
	// to the named repository.
	Subscribed(string, string) (bool, error)

	// SetSubscriber inserts a subscriber for the named
	// repository.
	SetSubscriber(string, string) error

	// DelSubscriber removes the subscriber by login for the
	// named repository.
	DelSubscriber(string, string) error

	// Repo returns the repository with the given name.
	Repo(string) (*common.Repo, error)

	// RepoList returns a list of repositories for the
	// given user account.
	RepoList(string) ([]*common.Repo, error)

	// RepoParams returns the private environment parameters
	// for the given repository.
	RepoParams(string) (map[string]string, error)

	// RepoKeypair returns the private and public rsa keys
	// for the given repository.
	RepoKeypair(string) (*common.Keypair, error)

	// SetRepo inserts or updates a repository.
	SetRepo(*common.Repo) error

	// SetRepo updates a repository. If the repository
	// already exists ErrConflict is returned.
	SetRepoNotExists(*common.User, *common.Repo) error

	// SetRepoParams inserts or updates the private
	// environment parameters for the named repository.
	SetRepoParams(string, map[string]string) error

	// SetRepoKeypair inserts or updates the private and
	// public keypair for the named repository.
	SetRepoKeypair(string, *common.Keypair) error

	// DelRepo deletes the repository.
	DelRepo(*common.Repo) error

	// Build gets the specified build number for the
	// named repository and build number
	Build(string, int) (*common.Build, error)

	// BuildList gets a list of recent builds for the
	// named repository.
	BuildList(string) ([]*common.Build, error)

	// BuildLast gets the last executed build for the
	// named repository.
	BuildLast(string) (*common.Build, error)

	// BuildAgent returns the agent that is being
	// used to execute the build.
	BuildAgent(string, int) (*common.Agent, error)

	// SetBuild inserts or updates a build for the named
	// repository. The build number is incremented and
	// assigned to the provided build.
	SetBuild(string, *common.Build) error

	// SetBuildState updates an existing build's start time,
	// finish time, duration and state. No other fields are
	// updated.
	SetBuildState(string, *common.Build) error

	// SetBuildStatus appends a new build status to an
	// existing build record.
	SetBuildStatus(string, int, *common.Status) error

	// SetBuildTask updates an existing build task. The build
	// and task must already exist. If the task does not exist
	// an error is returned.
	SetBuildTask(string, int, *common.Task) error

	// SetBuildAgent insert or updates the agent that is
	// running a build.
	SetBuildAgent(string, int, *common.Agent) error

	// DelBuildAgent purges the referce to the agent
	// that ran a build.
	DelBuildAgent(string, int) error

	// LogReader gets the task logs at index N for
	// the named repository and build number.
	LogReader(string, int, int) (io.Reader, error)

	// SetLogs inserts or updates a task logs for the
	// named repository and build number.
	SetLogs(string, int, int, []byte) error
}
