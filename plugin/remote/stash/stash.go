package stash

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/reinbach/go-stash/oauth1"
	"github.com/reinbach/go-stash/stash"
)

type Stash struct {
	URL        string
	API        string
	Secret     string
	PrivateKey string
	Hook       string
	Open       bool
}

func New(url, api, secret, private_key, hook string, open bool) *Stash {
	return &Stash{
		URL:        url,
		API:        api,
		Secret:     secret,
		PrivateKey: private_key,
		Hook:       hook,
		Open:       open,
	}
}

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (r *Stash) Authorize(w http.ResponseWriter, req *http.Request) (*model.Login, error) {
	var consumer = oauth1.Consumer{
		RequestTokenURL:       r.URL + "/plugins/servlet/oauth/request-token",
		AuthorizationURL:      r.URL + "/plugins/servlet/oauth/authorize",
		AccessTokenURL:        r.URL + "/plugins/servlet/oauth/access-token",
		CallbackURL:           httputil.GetScheme(req) + "://" + httputil.GetHost(req) + "/api/auth/stash.atlassian.com",
		ConsumerKey:           r.Secret,
		ConsumerPrivateKeyPem: r.PrivateKey,
	}

	// get the oauth verifier
	verifier := req.FormValue("oauth_verifier")
	if len(verifier) == 0 {
		// Generate a Request Token
		requestToken, err := consumer.RequestToken()
		if err != nil {
			return nil, err
		}

		// add the request token as a signed cookie
		httputil.SetCookie(w, req, "stash_token", requestToken.Encode())

		url, _ := consumer.AuthorizeRedirect(requestToken)
		http.Redirect(w, req, url, http.StatusSeeOther)
		return nil, nil
	}

	// remove stash token data once before redirecting
	// back to the application.
	defer httputil.DelCookie(w, req, "stash_token")

	// get the tokens from the request
	requestTokenStr := httputil.GetCookie(req, "stash_token")
	requestToken, err := oauth1.ParseRequestTokenStr(requestTokenStr)
	if err != nil {
		return nil, err
	}

	// exchange for an access token
	accessToken, err := consumer.AuthorizeToken(requestToken, verifier)
	if err != nil {
		return nil, err
	}

	// create the Stash client
	var client = stash.New(
		r.URL,
		r.Secret,
		accessToken.Token(),
		accessToken.Secret(),
		r.PrivateKey,
	)

	// get the currently authenticated Stash User
	user, err := client.Users.Current()
	if err != nil {
		return nil, err
	}

	// put the user data in the common format
	login := model.Login{
		Login:  user.Username,
		Access: accessToken.Token(),
		Secret: accessToken.Secret(),
		//Name:   user.DisplayName,
	}

	return &login, nil
}

// GetKind returns the internal identifier of this remote Stash instance.
func (r *Stash) GetKind() string {
	return model.RemoteStash
}

// GetHost returns the url.Host of this remote system.
func (r *Stash) GetHost() string {
	uri, _ := url.Parse(r.URL)
	return uri.Host
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (r *Stash) GetRepos(user *model.User) ([]*model.Repo, error) {
	var repos []*model.Repo
	var client = stash.New(
		r.URL,
		r.Secret,
		user.Access,
		user.Secret,
		r.PrivateKey,
	)

	// parse the hostname from the stash url
	var stashurl, err = url.Parse(r.URL)
	if err != nil {
		return nil, err
	}

	// parse the hostname from the stash api
	stashapi, err := url.Parse(r.API)
	if err != nil {
		return nil, err
	}

	list, err := client.Repos.List()
	if err != nil {
		return nil, err
	}

	var remote = r.GetKind()
	var hostname = r.GetHost()

	for _, item := range list {
		// for now we only support git repos
		if item.ScmId != "git" {
			continue
		}

		// these are the urls required to clone the repository
		var clone = fmt.Sprintf("https://%s%s/%s/%s.git", stashurl.Host, stashurl.Path, item.Project.Key, item.Name)
		var ssh = fmt.Sprintf("ssh://git@%s/%s/%s.git", stashapi.Host, item.Project.Key, item.Name)

		var repo = model.Repo{
			UserID:   user.ID,
			Remote:   remote,
			Host:     hostname,
			Owner:    item.Project.Key,
			Name:     item.Name,
			Private:  !item.Public,
			CloneURL: clone,
			GitURL:   clone,
			SSHURL:   ssh,
			Role: &model.Perm{
				Admin: true,
				Write: true,
				Read:  true,
			},
		}

		if repo.Private {
			repo.CloneURL = repo.SSHURL
		}

		repos = append(repos, &repo)
	}

	return repos, err
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (r *Stash) GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error) {
	// create the Atlassian Stash client
	var client = stash.New(
		r.URL,
		r.Secret,
		user.Access,
		user.Secret,
		r.PrivateKey,
	)

	// get the yaml from the database
	var raw, err = client.Contents.Find(hook.Owner, hook.Repo, ".drone.yml")
	if err != nil {
		return nil, err
	}

	return []byte(raw), nil
}

// Activate activates a repository by adding a Post-Commit hook and
// a Public Deploy key, if applicable.
func (r *Stash) Activate(user *model.User, repo *model.Repo, link string) error {
	var client = stash.New(
		r.URL,
		r.Secret,
		user.Access,
		user.Secret,
		r.PrivateKey,
	)

	// if the repository is private we'll need
	// to upload a stash key to the repository
	if repo.Private {
		var _, err = client.Keys.CreateUpdate(repo.PublicKey)
		if err != nil {
			return err
		}
	}

	// add the hook
	var hook = GetHook(user, repo, link)
	var _, err = client.Hooks.CreateHook(repo.Owner, repo.Name, r.Hook, hook)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link. SSH key is not removed as this is on the user,
// not the repo
func (r *Stash) Deactivate(user *model.User, repo *model.Repo, link string) error {
	var client = stash.New(
		r.URL,
		r.Secret,
		user.Access,
		user.Secret,
		r.PrivateKey,
	)
	var hook = GetHook(user, repo, link)
	return client.Hooks.DeleteHook(repo.Owner, repo.Name, r.Hook, hook)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *Stash) ParseHook(req *http.Request) (*model.Hook, error) {
	// get the project and repo from the request
	var owner = req.FormValue("owner")
	var name = req.FormValue("name")
	var branch = req.FormValue("branch")
	var hash = req.FormValue("hash")
	var message = req.FormValue("type")
	var author = req.FormValue("displayName")

	// verify the payload has the minimum amount of required data.
	if owner == "" || branch == "" {
		return nil, fmt.Errorf("Invalid Atlassian Stash post-commit Hook. Missing Repo or Branch data.")
	}

	return &model.Hook{
		Owner:     owner,
		Repo:      name,
		Sha:       hash,
		Branch:    branch,
		Author:    author,
		Timestamp: time.Now().UTC().String(),
		Message:   message,
	}, nil
}

func (s *Stash) OpenRegistration() bool {
	return s.Open
}

func (s *Stash) GetToken(user *model.User) (*model.Token, error) {
	return nil, nil
}

// GetKeyTitle is a helper function that generates a title for the
// RSA public key based on the username and domain name.
func GetKeyTitle(rawurl string) (string, error) {
	var uri, err = url.Parse(rawurl)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("drone@%s", uri.Host), nil
}

func GetHook(user *model.User, repo *model.Repo, link string) string {
	return fmt.Sprintf("%s?owner=%s&name=%s&branch=${refChange.name}&hash=${refChange.toHash}&message=${refChange.type}&author=${user.name}", link, repo.Owner, repo.Name)
}
