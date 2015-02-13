package bitbucket

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/drone/go-bitbucket/bitbucket"
	"github.com/drone/go-bitbucket/oauth1"
)

const (
	DefaultAPI = "https://api.bitbucket.org/1.0"
	DefaultURL = "https://bitbucket.org"
)

// parses an email address from string format
// `John Doe <john.doe@example.com>`
var emailRegexp = regexp.MustCompile("<(.*)>")

type Bitbucket struct {
	URL    string
	API    string
	Client string
	Secret string
	Open   bool
}

func New(url, api, client, secret string, open bool) *Bitbucket {
	return &Bitbucket{
		URL:    url,
		API:    api,
		Client: client,
		Secret: secret,
		Open:   open,
	}
}

func NewDefault(client, secret string, open bool) *Bitbucket {
	return New(DefaultURL, DefaultAPI, client, secret, open)
}

// Authorize handles Bitbucket API Authorization
func (r *Bitbucket) Authorize(res http.ResponseWriter, req *http.Request) (*model.Login, error) {
	consumer := oauth1.Consumer{
		RequestTokenURL:  "https://bitbucket.org/api/1.0/oauth/request_token/",
		AuthorizationURL: "https://bitbucket.org/!api/1.0/oauth/authenticate",
		AccessTokenURL:   "https://bitbucket.org/api/1.0/oauth/access_token/",
		CallbackURL:      httputil.GetScheme(req) + "://" + httputil.GetHost(req) + "/api/auth/bitbucket.org",
		ConsumerKey:      r.Client,
		ConsumerSecret:   r.Secret,
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
		httputil.SetCookie(res, req, "bitbucket_token", requestToken.Encode())

		url, _ := consumer.AuthorizeRedirect(requestToken)
		http.Redirect(res, req, url, http.StatusSeeOther)
		return nil, nil
	}

	// remove bitbucket token data once before redirecting
	// back to the application.
	defer httputil.DelCookie(res, req, "bitbucket_token")

	// get the tokens from the request
	requestTokenStr := httputil.GetCookie(req, "bitbucket_token")
	requestToken, err := oauth1.ParseRequestTokenStr(requestTokenStr)
	if err != nil {
		return nil, err
	}

	// exchange for an access token
	accessToken, err := consumer.AuthorizeToken(requestToken, verifier)
	if err != nil {
		return nil, err
	}

	// create the Bitbucket client
	client := bitbucket.New(
		r.Client,
		r.Secret,
		accessToken.Token(),
		accessToken.Secret(),
	)

	// get the currently authenticated Bitbucket User
	user, err := client.Users.Current()
	if err != nil {
		return nil, err
	}

	// put the user data in the common format
	login := model.Login{
		Login:  user.User.Username,
		Access: accessToken.Token(),
		Secret: accessToken.Secret(),
		Name:   user.User.DisplayName,
	}

	email, _ := client.Emails.FindPrimary(user.User.Username)
	if email != nil {
		login.Email = email.Email
	}

	return &login, nil
}

// GetKind returns the internal identifier of this remote Bitbucket instane.
func (r *Bitbucket) GetKind() string {
	return model.RemoteBitbucket
}

// GetHost returns the hostname of this remote Bitbucket instance.
func (r *Bitbucket) GetHost() string {
	uri, _ := url.Parse(r.URL)
	return uri.Host
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (r *Bitbucket) GetRepos(user *model.User) ([]*model.Repo, error) {
	var repos []*model.Repo
	var client = bitbucket.New(
		r.Client,
		r.Secret,
		user.Access,
		user.Secret,
	)
	var list, err = client.Repos.List()
	if err != nil {
		return nil, err
	}

	var remote = r.GetKind()
	var hostname = r.GetHost()

	for _, item := range list {
		// for now we only support git repos
		if item.Scm != "git" {
			continue
		}

		// these are the urls required to clone the repository
		// TODO use the bitbucketurl.Host and bitbucketurl.Scheme instead of hardcoding
		//      so that we can support Stash.
		var html = fmt.Sprintf("https://bitbucket.org/%s/%s", item.Owner, item.Slug)
		var clone = fmt.Sprintf("https://bitbucket.org/%s/%s.git", item.Owner, item.Slug)
		var ssh = fmt.Sprintf("git@bitbucket.org:%s/%s.git", item.Owner, item.Slug)

		var repo = model.Repo{
			UserID:   user.ID,
			Remote:   remote,
			Host:     hostname,
			Owner:    item.Owner,
			Name:     item.Slug,
			Private:  item.Private,
			URL:      html,
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
func (r *Bitbucket) GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error) {
	var client = bitbucket.New(
		r.Client,
		r.Secret,
		user.Access,
		user.Secret,
	)

	// get the yaml from the database
	var raw, err = client.Sources.Find(repo.Owner, repo.Name, hook.Sha, ".drone.yml")
	if err != nil {
		return nil, err
	}

	return []byte(raw.Data), nil
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (r *Bitbucket) Activate(user *model.User, repo *model.Repo, link string) error {
	var client = bitbucket.New(
		r.Client,
		r.Secret,
		user.Access,
		user.Secret,
	)

	// parse the hostname from the hook, and use this
	// to name the ssh key
	var hookurl, err = url.Parse(link)
	if err != nil {
		return err
	}

	// if the repository is private we'll need
	// to upload a github key to the repository
	if repo.Private {
		// name the key
		var keyname = "drone@" + hookurl.Host
		var _, err = client.RepoKeys.CreateUpdate(repo.Owner, repo.Name, repo.PublicKey, keyname)
		if err != nil {
			return err
		}
	}

	// add the hook
	_, err = client.Brokers.CreateUpdate(repo.Owner, repo.Name, link, bitbucket.BrokerTypePost)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (r *Bitbucket) Deactivate(user *model.User, repo *model.Repo, link string) error {
	var client = bitbucket.New(
		r.Client,
		r.Secret,
		user.Access,
		user.Secret,
	)
	title, err := GetKeyTitle(link)
	if err != nil {
		return err
	}
	if err := client.RepoKeys.DeleteName(repo.Owner, repo.Name, title); err != nil {
		return err
	}
	return client.Brokers.DeleteUrl(repo.Owner, repo.Name, link, bitbucket.BrokerTypePost)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *Bitbucket) ParseHook(req *http.Request) (*model.Hook, error) {
	var payload = req.FormValue("payload")
	var hook, err = bitbucket.ParseHook([]byte(payload))
	if err != nil {
		return nil, err
	}

	// verify the payload has the minimum amount of required data.
	if hook.Repo == nil || hook.Commits == nil || len(hook.Commits) == 0 {
		return nil, fmt.Errorf("Invalid Bitbucket post-commit Hook. Missing Repo or Commit data.")
	}

	var author = hook.Commits[len(hook.Commits)-1].RawAuthor
	var matches = emailRegexp.FindStringSubmatch(author)
	if len(matches) == 2 {
		author = matches[1]
	}

	return &model.Hook{
		Owner:     hook.Repo.Owner,
		Repo:      hook.Repo.Slug,
		Sha:       hook.Commits[len(hook.Commits)-1].Hash,
		Branch:    hook.Commits[len(hook.Commits)-1].Branch,
		Author:    author,
		Timestamp: time.Now().UTC().String(),
		Message:   hook.Commits[len(hook.Commits)-1].Message,
	}, nil
}

func (r *Bitbucket) OpenRegistration() bool {
	return r.Open
}

func (r *Bitbucket) GetToken(user *model.User) (*model.Token, error) {
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
