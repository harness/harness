package remote

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/bitbucket"
	"github.com/drone/drone/remote/github"
	"github.com/drone/drone/remote/gitlab"
	"github.com/drone/drone/remote/gogs"
	"github.com/drone/drone/shared/envconfig"

	log "github.com/Sirupsen/logrus"
)

func Load(env envconfig.Env) Remote {
	driver := env.Get("REMOTE_DRIVER")

	switch driver {
	case "bitbucket":
		return bitbucket.Load(env)
	case "github":
		return github.Load(env)
	case "gitlab":
		return gitlab.Load(env)
	case "gogs":
		return gogs.Load(env)

	default:
		log.Fatalf("unknown remote driver %s", driver)
	}

	return nil
}

type Remote interface {
	// Login authenticates the session and returns the
	// remote user details.
	Login(w http.ResponseWriter, r *http.Request) (*model.User, bool, error)

	// Auth authenticates the session and returns the remote user
	// login for the given token and secret
	Auth(token, secret string) (string, error)

	// Repo fetches the named repository from the remote system.
	Repo(u *model.User, owner, repo string) (*model.Repo, error)

	// Repos fetches a list of repos from the remote system.
	Repos(u *model.User) ([]*model.RepoLite, error)

	// Perm fetches the named repository permissions from
	// the remote system for the specified user.
	Perm(u *model.User, owner, repo string) (*model.Perm, error)

	// Script fetches the build script (.drone.yml) from the remote
	// repository and returns in string format.
	Script(u *model.User, r *model.Repo, b *model.Build) ([]byte, []byte, error)

	// Status sends the commit status to the remote system.
	// An example would be the GitHub pull request status.
	Status(u *model.User, r *model.Repo, b *model.Build, link string) error

	// Netrc returns a .netrc file that can be used to clone
	// private repositories from a remote system.
	Netrc(u *model.User, r *model.Repo) (*model.Netrc, error)

	// Activate activates a repository by creating the post-commit hook and
	// adding the SSH deploy key, if applicable.
	Activate(u *model.User, r *model.Repo, k *model.Key, link string) error

	// Deactivate removes a repository by removing all the post-commit hooks
	// which are equal to link and removing the SSH deploy key.
	Deactivate(u *model.User, r *model.Repo, link string) error

	// Hook parses the post-commit hook from the Request body
	// and returns the required data in a standard format.
	Hook(r *http.Request) (*model.Repo, *model.Build, error)
}

type Refresher interface {
	// Refresh refreshes an oauth token and expiration for the given
	// user. It returns true if the token was refreshed, false if the
	// token was not refreshed, and error if it failed to refersh.
	Refresh(*model.User) (bool, error)
}
