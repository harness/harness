package client

import (
	"io"

	"github.com/drone/drone/model"
	"github.com/drone/drone/queue"
)

// Client is used to communicate with a Drone server.
type Client interface {
	// Self returns the currently authenticated user.
	Self() (*model.User, error)

	// User returns a user by login.
	User(string) (*model.User, error)

	// UserList returns a list of all registered users.
	UserList() ([]*model.User, error)

	// UserPost creates a new user account.
	UserPost(*model.User) (*model.User, error)

	// UserPatch updates a user account.
	UserPatch(*model.User) (*model.User, error)

	// UserDel deletes a user account.
	UserDel(string) error

	// // UserFeed returns the user's activity feed.
	// UserFeed() ([]*Activity, error)

	// Repo returns a repository by name.
	Repo(string, string) (*model.Repo, error)

	// RepoList returns a list of all repositories to which the user has explicit
	// access in the host system.
	RepoList() ([]*model.Repo, error)

	// RepoPost activates a repository.
	RepoPost(string, string) (*model.Repo, error)

	// RepoPatch updates a repository.
	RepoPatch(*model.Repo) (*model.Repo, error)

	// RepoChown updates a repository owner.
	RepoChown(string, string) (*model.Repo, error)

	// RepoDel deletes a repository.
	RepoDel(string, string) error

	// Sign returns a cryptographic signature for the input string.
	Sign(string, string, []byte) ([]byte, error)

	// SecretList returns a list of all repository secrets.
	SecretList(string, string) ([]*model.Secret, error)

	// SecretPost create or updates a repository secret.
	SecretPost(string, string, *model.Secret) error

	// SecretDel deletes a named repository secret.
	SecretDel(string, string, string) error

	// TeamSecretList returns a list of all team secrets.
	TeamSecretList(string) ([]*model.Secret, error)

	// TeamSecretPost create or updates a team secret.
	TeamSecretPost(string, *model.Secret) error

	// TeamSecretDel deletes a named team secret.
	TeamSecretDel(string, string) error

	// Build returns a repository build by number.
	Build(string, string, int) (*model.Build, error)

	// BuildLast returns the latest repository build by branch. An empty branch
	// will result in the default branch.
	BuildLast(string, string, string) (*model.Build, error)

	// BuildList returns a list of recent builds for the
	// the specified repository.
	BuildList(string, string) ([]*model.Build, error)

	// BuildQueue returns a list of enqueued builds.
	BuildQueue() ([]*model.Feed, error)

	// BuildStart re-starts a stopped build.
	BuildStart(string, string, int, map[string]string) (*model.Build, error)

	// BuildStop stops the specified running job for given build.
	BuildStop(string, string, int, int) error

	// BuildFork re-starts a stopped build with a new build number, preserving
	// the prior history.
	BuildFork(string, string, int, map[string]string) (*model.Build, error)

	// BuildLogs returns the build logs for the specified job.
	BuildLogs(string, string, int, int) (io.ReadCloser, error)

	// Deploy triggers a deployment for an existing build using the specified
	// target environment.
	Deploy(string, string, int, string, map[string]string) (*model.Build, error)

	// AgentList returns a list of build agents.
	AgentList() ([]*model.Agent, error)

	//
	// below items for Queue (internal use only)
	//

	// Pull pulls work from the server queue.
	Pull(os, arch string) (*queue.Work, error)

	// Push pushes an update to the server.
	Push(*queue.Work) error

	// Stream streams the build logs to the server.
	Stream(int64, io.ReadCloser) error

	LogStream(int64) (StreamWriter, error)

	LogPost(int64, io.ReadCloser) error

	// Wait waits for the job to the complete.
	Wait(int64) *Wait

	// Ping the server
	Ping() error
}
