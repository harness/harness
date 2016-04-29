package gogs

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/gogits/go-gogs-client"
)

// Remote defines a remote implementation that integrates with Gogs, an open
// source Git service written in Go. See https://gogs.io/
type Remote struct {
	URL         string
	Open        bool
	PrivateMode bool
	SkipVerify  bool
}

// New returns a Remote implementation that integrates with Gogs, an open
// source Git service written in Go. See https://gogs.io/
func New(url string, private, skipverify bool) remote.Remote {
	return &Remote{
		URL:         url,
		PrivateMode: private,
		SkipVerify:  skipverify,
	}
}

// Login authenticates the session and returns the authenticated user.
func (g *Remote) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	var (
		username = req.FormValue("username")
		password = req.FormValue("password")
	)

	// if the username or password doesn't exist we re-direct
	// the user to the login screen.
	if len(username) == 0 || len(password) == 0 {
		http.Redirect(res, req, "/login/form", http.StatusSeeOther)
		return nil, false, nil
	}

	client := NewGogsClient(g.URL, "", g.SkipVerify)

	// try to fetch drone token if it exists
	var accessToken string
	tokens, err := client.ListAccessTokens(username, password)
	if err != nil {
		return nil, false, err
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
			return nil, false, err
		}
		accessToken = token.Sha1
	}

	client = NewGogsClient(g.URL, accessToken, g.SkipVerify)
	userInfo, err := client.GetUserInfo(username)
	if err != nil {
		return nil, false, err
	}

	user := model.User{}
	user.Token = accessToken
	user.Login = userInfo.UserName
	user.Email = userInfo.Email
	user.Avatar = expandAvatar(g.URL, userInfo.AvatarUrl)
	return &user, false, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (g *Remote) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Method not supported")
}

// Repo fetches the named repository from the remote system.
func (g *Remote) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := NewGogsClient(g.URL, u.Token, g.SkipVerify)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return nil, err
	}

	fullName := owner + "/" + name
	for _, repo := range repos_ {
		if repo.FullName == fullName {
			return toRepo(repo), nil
		}
	}

	return nil, fmt.Errorf("Not Found")
}

// Repos fetches a list of repos from the remote system.
func (g *Remote) Repos(u *model.User) ([]*model.RepoLite, error) {
	repos := []*model.RepoLite{}

	client := NewGogsClient(g.URL, u.Token, g.SkipVerify)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return repos, err
	}

	for _, repo := range repos_ {
		repos = append(repos, toRepoLite(repo))
	}

	return repos, err
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (g *Remote) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := NewGogsClient(g.URL, u.Token, g.SkipVerify)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return nil, err
	}

	fullName := owner + "/" + name
	for _, repo := range repos_ {
		if repo.FullName == fullName {
			return toPerm(repo.Permissions), nil
		}
	}

	return nil, fmt.Errorf("Not Found")

}

// File fetches a file from the remote repository and returns in string format.
func (g *Remote) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	client := NewGogsClient(g.URL, u.Token, g.SkipVerify)
	cfg, err := client.GetFile(r.Owner, r.Name, b.Commit, f)
	return cfg, err
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (g *Remote) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return fmt.Errorf("Not Implemented")
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *Remote) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	url_, err := url.Parse(g.URL)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(url_.Host)
	if err == nil {
		url_.Host = host
	}
	return &model.Netrc{
		Login:    u.Token,
		Password: "x-oauth-basic",
		Machine:  url_.Host,
	}, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (g *Remote) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	config := map[string]string{
		"url":          link,
		"secret":       r.Hash,
		"content_type": "json",
	}
	hook := gogs.CreateHookOption{
		Type:   "gogs",
		Config: config,
		Active: true,
	}

	client := NewGogsClient(g.URL, u.Token, g.SkipVerify)
	_, err := client.CreateRepoHook(r.Owner, r.Name, hook)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (g *Remote) Deactivate(u *model.User, r *model.Repo, link string) error {
	return fmt.Errorf("Not Implemented")
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Remote) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	var (
		err   error
		repo  *model.Repo
		build *model.Build
	)

	switch r.Header.Get("X-Gogs-Event") {
	case "push":
		var push *PushHook
		push, err = parsePush(r.Body)
		if err == nil {
			repo = repoFromPush(push)
			build = buildFromPush(push)
		}
	}
	return repo, build, err
}

// NewClient initializes and returns a API client.
func NewGogsClient(url, token string, skipVerify bool) *gogs.Client {
	sslClient := &http.Client{}
	c := gogs.NewClient(url, token)

	if skipVerify {
		sslClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		c.SetHTTPClient(sslClient)
	}
	return c
}

func (g *Remote) String() string {
	return "gogs"
}
