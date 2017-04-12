package client

import (
	"io"

	"github.com/drone/drone/model"
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

	// Repo returns a repository by name.
	Repo(string, string) (*model.Repo, error)

	// RepoList returns a list of all repositories to which the user has explicit
	// access in the host system.
	RepoList() ([]*model.Repo, error)

	// RepoPost activates a repository.
	RepoPost(string, string) (*model.Repo, error)

	// RepoPatch updates a repository.
	RepoPatch(string, string, *model.RepoPatch) (*model.Repo, error)

	// RepoChown updates a repository owner.
	RepoChown(string, string) (*model.Repo, error)

	// RepoRepair repairs the repository hooks.
	RepoRepair(string, string) error

	// RepoDel deletes a repository.
	RepoDel(string, string) error

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

	// BuildApprove approves a blocked build.
	BuildApprove(string, string, int) (*model.Build, error)

	// BuildDecline declines a blocked build.
	BuildDecline(string, string, int) (*model.Build, error)

	// BuildLogs returns the build logs for the specified job.
	BuildLogs(string, string, int, int) (io.ReadCloser, error)

	// Deploy triggers a deployment for an existing build using the specified
	// target environment.
	Deploy(string, string, int, string, map[string]string) (*model.Build, error)

	// Registry returns a registry by hostname.
	Registry(owner, name, hostname string) (*model.Registry, error)

	// RegistryList returns a list of all repository registries.
	RegistryList(owner, name string) ([]*model.Registry, error)

	// RegistryCreate creates a registry.
	RegistryCreate(owner, name string, registry *model.Registry) (*model.Registry, error)

	// RegistryUpdate updates a registry.
	RegistryUpdate(owner, name string, registry *model.Registry) (*model.Registry, error)

	// RegistryDelete deletes a registry.
	RegistryDelete(owner, name, hostname string) error

	// Secret returns a secret by name.
	Secret(owner, name, secret string) (*model.Secret, error)

	// SecretList returns a list of all repository secrets.
	SecretList(owner, name string) ([]*model.Secret, error)

	// SecretCreate creates a registry.
	SecretCreate(owner, name string, secret *model.Secret) (*model.Secret, error)

	// SecretUpdate updates a registry.
	SecretUpdate(owner, name string, secret *model.Secret) (*model.Secret, error)

	// SecretDelete deletes a secret.
	SecretDelete(owner, name, secret string) error
}
