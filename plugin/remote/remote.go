package remote

import (
	"net/http"

	"github.com/drone/drone/shared/model"
)

type Remote interface {
	// Authorize handles authentication with thrid party remote systems,
	// such as github or bitbucket, and returns user data.
	Authorize(w http.ResponseWriter, r *http.Request) (*model.Login, error)

	// GetKind returns the kind of plugin
	GetKind() string

	// GetHost returns the hostname of the remote service.
	GetHost() string

	// GetRepos fetches all repositories that the specified
	// user has access to in the remote system.
	GetRepos(user *model.User) ([]*model.Repo, error)

	// GetScript fetches the build script (.drone.yml) from the remote
	// repository and returns in string format.
	GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error)

	// Activate activates a repository by creating the post-commit hook and
	// adding the SSH deploy key, if applicable.
	Activate(user *model.User, repo *model.Repo, link string) error

	// Deactivate removes a repository by removing all the post-commit hooks
	// which are equal to link and removing the SSH deploy key.
	Deactivate(user *model.User, repo *model.Repo, link string) error

	// ParseHook parses the post-commit hook from the Request body
	// and returns the required data in a standard format.
	ParseHook(r *http.Request) (*model.Hook, error)

	// Registration returns true if open registration is allowed
	OpenRegistration() bool

	// Get token
	GetToken(*model.User) (*model.Token, error)
}

// List of registered plugins.
var remotes []Remote

// Register registers a plugin by name.
//
// All plugins must be registered when the application
// initializes. This should not be invoked while the application
// is running, and is not thread safe.
func Register(remote Remote) {
	remotes = append(remotes, remote)
}

// List Registered remote plugins
func Registered() []Remote {
	return remotes
}

// Lookup gets a plugin by name.
func Lookup(name string) Remote {
	for _, remote := range remotes {
		if remote.GetKind() == name ||
			remote.GetHost() == name {
			return remote
		}
	}
	return nil
}
