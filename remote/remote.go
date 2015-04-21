package remote

import (
	"net/http"

	"github.com/drone/drone/common"
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
	Status(u *common.User, r *common.Repo, b *common.Build, link string) error

	// Activate activates a repository by creating the post-commit hook and
	// adding the SSH deploy key, if applicable.
	Activate(u *common.User, r *common.Repo, k *common.Keypair, link string) error

	// Deactivate removes a repository by removing all the post-commit hooks
	// which are equal to link and removing the SSH deploy key.
	Deactivate(u *common.User, r *common.Repo, link string) error

	// Hook parses the post-commit hook from the Request body
	// and returns the required data in a standard format.
	Hook(r *http.Request) (*common.Hook, error)
}
