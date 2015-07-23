package remote

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/drone/drone/pkg/config"
	"github.com/drone/drone/pkg/oauth2"
	"github.com/drone/drone/pkg/remote/builtin/github"
	common "github.com/drone/drone/pkg/types"
)

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

func New(conf *config.Config) (Remote, error) {
	switch strings.ToLower(conf.Remote.Driver) {
	case "github":
		return github.New(conf), nil
	case "":
		return nil, errors.New("Remote not specifed, please set env variable DRONE_REMOTE_DRIVER")
	default:
		return nil, errors.New(fmt.Sprintf("Remote driver not supported: DRONE_REMOTE_DRIVER=%s", conf.Remote.Driver))
	}
}
