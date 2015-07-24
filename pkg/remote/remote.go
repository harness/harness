package remote

import (
	"fmt"
	"net/http"

	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/oauth2"
	common "github.com/drone/drone/pkg/types"
)

var drivers = make(map[string]DriverFunc)

// Register makes a remote driver available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, driver DriverFunc) {
	if driver == nil {
		panic("remote: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("remote: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// DriverFunc returns a new connection to the remote.
// Config is a struct, with base remote configuration.
type DriverFunc func(conf *config.Config) (Remote, error)

// New creates a new remote connection.
func New(driver string, conf *config.Config) (Remote, error) {
	fn, ok := drivers[driver]
	if !ok {
		return nil, fmt.Errorf("remote: unknown driver %q", driver)
	}
	return fn(conf)
}

type Remote interface {
	// Login authenticates the session and returns the
	// remote user details.
	Login(token, secret string) (*common.User, error)

	// Orgs fetches the organizations for the given user.
	Orgs(u *common.User) ([]string, error)

	// Repo fetches the named repository from the remote system.
	Repo(u *common.User, owner, repo string) (*common.Repo, error)

	// Perm fetches the named repository permissions from
	// the remote system for the specified user.
	Perm(u *common.User, owner, repo string) (*common.Perm, error)

	// Script fetches the build script (.drone.yml) from the remote
	// repository and returns in string format.
	Script(u *common.User, r *common.Repo, b *common.Build) ([]byte, error)

	// Status sends the commit status to the remote system.
	// An example would be the GitHub pull request status.
	Status(u *common.User, r *common.Repo, b *common.Build) error

	// Netrc returns a .netrc file that can be used to clone
	// private repositories from a remote system.
	Netrc(u *common.User) (*common.Netrc, error)

	// Activate activates a repository by creating the post-commit hook and
	// adding the SSH deploy key, if applicable.
	Activate(u *common.User, r *common.Repo, k *common.Keypair, link string) error

	// Deactivate removes a repository by removing all the post-commit hooks
	// which are equal to link and removing the SSH deploy key.
	Deactivate(u *common.User, r *common.Repo, link string) error

	// Hook parses the post-commit hook from the Request body
	// and returns the required data in a standard format.
	Hook(r *http.Request) (*common.Hook, error)

	// Oauth2Transport
	Oauth2Transport(r *http.Request) *oauth2.Transport

	// GetOrgs returns all allowed organizations for remote.
	GetOrgs() []string

	// GetOpen returns boolean field with enabled or disabled
	// registration.
	GetOpen() bool

	// Default scope for remote
	Scope() string
}
