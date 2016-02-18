package phabricator

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/oauth2"

	log "github.com/Sirupsen/logrus"
)

const (
	DefaultScope = "offline_access"
)

type Phabricator struct {
	URL        string
	Client     string
	Secret     string
	Open       bool
	SkipVerify bool
}

func Load(env envconfig.Env) *Phabricator {
	config := env.String("REMOTE_CONFIG", "")
	parsed, err := url.Parse(config)

	if err != nil {
		log.Fatalln("Unable to parse remote DSN. %s", err)
	}

	params := parsed.Query()
	parsed.Path = ""
	parsed.RawQuery = ""

	remote := Phabricator{}
	remote.URL = parsed.String()
	remote.Client = params.Get("client_id")
	remote.Secret = params.Get("client_secret")
	remote.SkipVerify, _ = strconv.ParseBool(params.Get("skip_verify"))
	remote.Open, _ = strconv.ParseBool(params.Get("open"))

	return &remote
}

// Login authenticates the session and returns the
// remote user details.
func (remote *Phabricator) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	config := &oauth2.Config{
		ClientId:     remote.Client,
		ClientSecret: remote.Secret,
		Scope:        DefaultScope,
		AuthURL:      fmt.Sprintf("%s/oauthserver/auth/", remote.URL),
		TokenURL:     fmt.Sprintf("%s/oauthserver/token/", remote.URL),
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	if req.FormValue("code") == "" {
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, false, nil
	}

	trans := &oauth2.Transport{
		Config: config,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: remote.SkipVerify,
			},
		},
	}

	token, err := trans.Exchange(req.FormValue("code"))

	if err != nil {
		return nil, false, fmt.Errorf("Error exchanging token. %s", err)
	}

	client := NewClient(remote.URL, token.AccessToken, remote.SkipVerify)
	login, err := client.CurrentUser()

	if err != nil {
		return nil, false, err
	}

	user := model.User{
		Login:  login.Username,
		Email:  login.Email,
		Avatar: login.AvatarUrl,
		Token:  token.AccessToken,
		Secret: token.RefreshToken,
	}

	return &user, remote.Open, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (remote *Phabricator) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Not implemented!")
}

// Repo fetches the named repository from the remote system.
func (remote *Phabricator) Repo(u *model.User, owner, repo string) (*model.Repo, error) {
	return nil, fmt.Errorf("Not implemented!")
}

// Repos fetches a list of repos from the remote system.
func (remote *Phabricator) Repos(u *model.User) ([]*model.RepoLite, error) {
	return nil, fmt.Errorf("Not implemented!")
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (remote *Phabricator) Perm(u *model.User, owner, repo string) (*model.Perm, error) {
	return nil, fmt.Errorf("Not implemented!")
}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (remote *Phabricator) Script(u *model.User, r *model.Repo, b *model.Build) ([]byte, []byte, error) {
	return nil, nil, fmt.Errorf("Not implemented!")
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (remote *Phabricator) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return fmt.Errorf("Not implemented!")
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (remote *Phabricator) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return nil, fmt.Errorf("Not implemented!")
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (remote *Phabricator) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	return fmt.Errorf("Not implemented!")
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (remote *Phabricator) Deactivate(u *model.User, r *model.Repo, link string) error {
	return fmt.Errorf("Not implemented!")
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (remote *Phabricator) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	return nil, nil, fmt.Errorf("Not implemented!")
}
