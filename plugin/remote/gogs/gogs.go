package gogs

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/drone/drone/shared/model"
	"github.com/gogits/go-gogs-client"
)

type Gogs struct {
	URL    string
	Secret string
	Open   bool
}

func New(url string, secret string, open bool) *Gogs {
	return &Gogs{URL: url, Secret: secret, Open: open}
}

// Authorize handles Gogs authorization
func (r *Gogs) Authorize(res http.ResponseWriter, req *http.Request) (*model.Login, error) {
	var username = req.FormValue("username")
	var password = req.FormValue("password")
	var client = gogs.NewClient(r.URL, "")

	// try to fetch drone token if it exists
	var accessToken = ""
	tokens, err := client.ListAccessTokens(username, password)
	if err != nil {
		return nil, err
	}
	for _, token := range tokens {
		if token.Name == "drone" {
			accessToken = token.Sha1
			break
		}
	}

	// if drone token not found, create it
	if accessToken == "" {
		token, err := client.CreateAccessToken(username, password, gogs.CreateAccessTokenOption{Name: "drone"})
		if err != nil {
			return nil, err
		}
		accessToken = token.Sha1
	}

	// update client
	client = gogs.NewClient(r.URL, accessToken)

	// fetch user information
	user, err := client.GetUserInfo(username)
	if err != nil {
		return nil, err
	}

	var login = new(model.Login)
	login.Name = user.FullName
	login.Email = user.Email
	login.Access = accessToken
	login.Login = username
	return login, nil
}

// GetKind returns the internal identifier of this remote Gogs instance
func (r *Gogs) GetKind() string {
	return model.RemoteGogs
}

// GetHost returns the hostname of this remote Gogs instance
func (r *Gogs) GetHost() string {
	uri, _ := url.Parse(r.URL)
	return uri.Host
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (r *Gogs) GetRepos(user *model.User) ([]*model.Repo, error) {
	var repos []*model.Repo

	var remote = r.GetKind()
	var hostname = r.GetHost()
	var client = gogs.NewClient(r.URL, user.Access)

	gogsRepos, err := client.ListMyRepos()

	if err != nil {
		return nil, err
	}

	for _, repo := range gogsRepos {
		var repoName = strings.Split(repo.FullName, "/")
		if len(repoName) < 2 {
			log.Println("invalid repo full_name", repo.FullName)
			continue
		}
		var owner = repoName[0]
		var name = repoName[1]

		var repo = model.Repo{
			UserID:   user.ID,
			Remote:   remote,
			Host:     hostname,
			Owner:    owner,
			Name:     name,
			Private:  repo.Private,
			CloneURL: repo.CloneUrl,
			GitURL:   repo.CloneUrl,
			SSHURL:   repo.SshUrl,
			URL:      repo.HtmlUrl,
			Role: &model.Perm{
				Admin: repo.Permissions.Admin,
				Write: repo.Permissions.Push,
				Read:  repo.Permissions.Pull,
			},
		}

		repos = append(repos, &repo)
	}

	return repos, err
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns a byte array
func (r *Gogs) GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error) {
	var client = gogs.NewClient(r.URL, user.Access)
	return client.GetFile(repo.Owner, repo.Name, hook.Sha, ".drone.yml")
}

// Activate activates a repository
func (r *Gogs) Activate(user *model.User, repo *model.Repo, link string) error {
	var client = gogs.NewClient(r.URL, user.Access)

	var config = map[string]string{
		"url":          link,
		"secret":       r.Secret,
		"content_type": "json",
	}
	var hook = gogs.CreateHookOption{
		Type:   "gogs",
		Config: config,
		Active: true,
	}

	_, err := client.CreateRepoHook(repo.Owner, repo.Name, hook)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (r *Gogs) Deactivate(user *model.User, repo *model.Repo, link string) error {
	return fmt.Errorf("Remove %#v in gogs not implemented", *repo)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *Gogs) ParseHook(req *http.Request) (*model.Hook, error) {
	defer req.Body.Close()
	var payloadbytes, _ = ioutil.ReadAll(req.Body)
	var payload, err = gogs.ParseHook(payloadbytes)
	if err != nil {
		return nil, err
	}

	// verify the payload has the minimum amount of required data.
	if payload.Repo == nil || payload.Commits == nil || len(payload.Commits) == 0 {
		return nil, fmt.Errorf("Invalid Gogs post-commit Hook. Missing Repo or Commit data.")
	}

	if payload.Secret != r.Secret {
		return nil, fmt.Errorf("Payload secret does not match stored secret")
	}

	return &model.Hook{
		Owner:     payload.Repo.Owner.UserName,
		Repo:      payload.Repo.Name,
		Sha:       payload.Commits[0].Id,
		Branch:    payload.Branch(),
		Author:    payload.Commits[0].Author.UserName,
		Timestamp: time.Now().UTC().String(),
		Message:   payload.Commits[0].Message,
	}, nil
}

func (r *Gogs) OpenRegistration() bool {
	return r.Open
}

func (r *Gogs) GetToken(user *model.User) (*model.Token, error) {
	return nil, nil
}
