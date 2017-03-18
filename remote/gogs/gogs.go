package gogs

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/gogits/go-gogs-client"
)

// Opts defines configuration options.
type Opts struct {
	URL         string // Gogs server url.
	Username    string // Optional machine account username.
	Password    string // Optional machine account password.
	PrivateMode bool   // Gogs is running in private mode.
	SkipVerify  bool   // Skip ssl verification.
}

type client struct {
	URL         string
	Machine     string
	Username    string
	Password    string
	PrivateMode bool
	SkipVerify  bool
}

// New returns a Remote implementation that integrates with Gogs, an open
// source Git service written in Go. See https://gogs.io/
func New(opts Opts) (remote.Remote, error) {
	url, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(url.Host)
	if err == nil {
		url.Host = host
	}
	return &client{
		URL:         opts.URL,
		Machine:     url.Host,
		Username:    opts.Username,
		Password:    opts.Password,
		PrivateMode: opts.PrivateMode,
		SkipVerify:  opts.SkipVerify,
	}, nil
}

// Login authenticates an account with Gogs using basic authenticaiton. The
// Gogs account details are returned when the user is successfully authenticated.
func (c *client) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	var (
		username = req.FormValue("username")
		password = req.FormValue("password")
	)

	// if the username or password is empty we re-direct to the login screen.
	if len(username) == 0 || len(password) == 0 {
		http.Redirect(res, req, "/login/form", http.StatusSeeOther)
		return nil, nil
	}

	client := c.newClient()

	// try to fetch drone token if it exists
	var accessToken string
	tokens, err := client.ListAccessTokens(username, password)
	if err == nil {
		for _, token := range tokens {
			if token.Name == "drone" {
				accessToken = token.Sha1
				break
			}
		}
	}

	// if drone token not found, create it
	if accessToken == "" {
		token, terr := client.CreateAccessToken(
			username,
			password,
			gogs.CreateAccessTokenOption{Name: "drone"},
		)
		if terr != nil {
			return nil, terr
		}
		accessToken = token.Sha1
	}

	client = c.newClientToken(accessToken)
	account, err := client.GetUserInfo(username)
	if err != nil {
		return nil, err
	}

	return &model.User{
		Token:  accessToken,
		Login:  account.UserName,
		Email:  account.Email,
		Avatar: expandAvatar(c.URL, account.AvatarUrl),
	}, nil
}

// Auth is not supported by the Gogs driver.
func (c *client) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Not Implemented")
}

// Teams is not supported by the Gogs driver.
func (c *client) Teams(u *model.User) ([]*model.Team, error) {
	client := c.newClientToken(u.Token)
	orgs, err := client.ListMyOrgs()
	if err != nil {
		return nil, err
	}

	var teams []*model.Team
	for _, org := range orgs {
		teams = append(teams, toTeam(org, c.URL))
	}
	return teams, nil
}

// TeamPerm is not supported by the Gogs driver.
func (c *client) TeamPerm(u *model.User, org string) (*model.Perm, error) {
	return nil, nil
}

// Repo returns the named Gogs repository.
func (c *client) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := c.newClientToken(u.Token)
	repo, err := client.GetRepo(owner, name)
	if err != nil {
		return nil, err
	}
	if c.PrivateMode {
		repo.Private = true
	}
	return toRepo(repo), nil
}

// Repos returns a list of all repositories for the Gogs account, including
// organization repositories.
func (c *client) Repos(u *model.User) ([]*model.RepoLite, error) {
	repos := []*model.RepoLite{}

	client := c.newClientToken(u.Token)
	all, err := client.ListMyRepos()
	if err != nil {
		return repos, err
	}

	for _, repo := range all {
		repos = append(repos, toRepoLite(repo))
	}
	return repos, err
}

// Perm returns the user permissions for the named Gogs repository.
func (c *client) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClientToken(u.Token)
	repo, err := client.GetRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return toPerm(repo.Permissions), nil
}

// File fetches the file from the Gogs repository and returns its contents.
func (c *client) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	client := c.newClientToken(u.Token)
	ref := b.Commit

	// TODO gogs does not yet return a sha with the pull request
	// so unfortunately we need to use the pull request branch.
	if b.Event == model.EventPull {
		ref = b.Branch
	}
	if ref == "" {
		// Remove refs/tags or refs/heads, Gogs needs a short ref
		ref = strings.TrimPrefix(
			strings.TrimPrefix(
				b.Ref,
				"refs/heads/",
			),
			"refs/tags/",
		)
	}
	cfg, err := client.GetFile(r.Owner, r.Name, ref, f)
	return cfg, err
}

// FileRef fetches the file from the Gogs repository and returns its contents.
func (c *client) FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error) {
	return c.newClientToken(u.Token).GetFile(r.Owner, r.Name, ref, f)
}

// Status is not supported by the Gogs driver.
func (c *client) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return nil
}

// Netrc returns a netrc file capable of authenticating Gogs requests and
// cloning Gogs repositories. The netrc will use the global machine account
// when configured.
func (c *client) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	if c.Password != "" {
		return &model.Netrc{
			Login:    c.Username,
			Password: c.Password,
			Machine:  c.Machine,
		}, nil
	}
	return &model.Netrc{
		Login:    u.Token,
		Password: "x-oauth-basic",
		Machine:  c.Machine,
	}, nil
}

// Activate activates the repository by registering post-commit hooks with
// the Gogs repository.
func (c *client) Activate(u *model.User, r *model.Repo, link string) error {
	config := map[string]string{
		"url":          link,
		"secret":       r.Hash,
		"content_type": "json",
	}
	hook := gogs.CreateHookOption{
		Type:   "gogs",
		Config: config,
		Events: []string{"push", "create", "pull_request"},
		Active: true,
	}

	client := c.newClientToken(u.Token)
	_, err := client.CreateRepoHook(r.Owner, r.Name, hook)
	return err
}

// Deactivate is not supported by the Gogs driver.
func (c *client) Deactivate(u *model.User, r *model.Repo, link string) error {
	return nil
}

// Hook parses the incoming Gogs hook and returns the Repository and Build
// details. If the hook is unsupported nil values are returned.
func (c *client) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	return parseHook(r)
}

// helper function to return the Gogs client
func (c *client) newClient() *gogs.Client {
	return c.newClientToken("")
}

// helper function to return the Gogs client
func (c *client) newClientToken(token string) *gogs.Client {
	client := gogs.NewClient(c.URL, token)
	if c.SkipVerify {
		httpClient := &http.Client{}
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.SetHTTPClient(httpClient)
	}
	return client
}
