package gitea

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"code.gitea.io/sdk/gitea"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
)

// Opts defines configuration options.
type Opts struct {
	URL         string // Gitea server url.
	Username    string // Optional machine account username.
	Password    string // Optional machine account password.
	PrivateMode bool   // Gitea is running in private mode.
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

const (
	DescPending  = "the build is pending"
	DescRunning  = "the build is running"
	DescSuccess  = "the build was successful"
	DescFailure  = "the build failed"
	DescCanceled = "the build canceled"
	DescBlocked  = "the build is pending approval"
	DescDeclined = "the build was rejected"
)

// getStatus is a helper function that converts a Drone
// status to a Gitea status.
func getStatus(status string) gitea.StatusState {
	switch status {
	case model.StatusPending, model.StatusBlocked:
		return gitea.StatusPending
	case model.StatusRunning:
		return gitea.StatusPending
	case model.StatusSuccess:
		return gitea.StatusSuccess
	case model.StatusFailure, model.StatusError:
		return gitea.StatusFailure
	case model.StatusKilled:
		return gitea.StatusFailure
	case model.StatusDeclined:
		return gitea.StatusWarning
	default:
		return gitea.StatusFailure
	}
}

// getDesc is a helper function that generates a description
// message for the build based on the status.
func getDesc(status string) string {
	switch status {
	case model.StatusPending:
		return DescPending
	case model.StatusRunning:
		return DescRunning
	case model.StatusSuccess:
		return DescSuccess
	case model.StatusFailure, model.StatusError:
		return DescFailure
	case model.StatusKilled:
		return DescCanceled
	case model.StatusBlocked:
		return DescBlocked
	case model.StatusDeclined:
		return DescDeclined
	default:
		return DescFailure
	}
}

// New returns a Remote implementation that integrates with Gitea, an open
// source Git service written in Go. See https://gitea.io/
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

// Login authenticates an account with Gitea using basic authenticaiton. The
// Gitea account details are returned when the user is successfully authenticated.
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
			gitea.CreateAccessTokenOption{Name: "drone"},
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
		Avatar: expandAvatar(c.URL, account.AvatarURL),
	}, nil
}

// Auth is not supported by the Gitea driver.
func (c *client) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Not Implemented")
}

// Teams is supported by the Gitea driver.
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

// TeamPerm is not supported by the Gitea driver.
func (c *client) TeamPerm(u *model.User, org string) (*model.Perm, error) {
	return nil, nil
}

// Repo returns the named Gitea repository.
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

// Repos returns a list of all repositories for the Gitea account, including
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

// Perm returns the user permissions for the named Gitea repository.
func (c *client) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClientToken(u.Token)
	repo, err := client.GetRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return toPerm(repo.Permissions), nil
}

// File fetches the file from the Gitea repository and returns its contents.
func (c *client) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	client := c.newClientToken(u.Token)
	cfg, err := client.GetFile(r.Owner, r.Name, b.Commit, f)
	return cfg, err
}

// FileRef fetches the file from the Gitea repository and returns its contents.
func (c *client) FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error) {
	return c.newClientToken(u.Token).GetFile(r.Owner, r.Name, ref, f)
}

// Status is supported by the Gitea driver.
func (c *client) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	client := c.newClientToken(u.Token)

	status := getStatus(b.Status)
	desc := getDesc(b.Status)

	_, err := client.CreateStatus(
		r.Owner,
		r.Name,
		b.Commit,
		gitea.CreateStatusOption{
			State:       status,
			TargetURL:   link,
			Description: desc,
			Context:     "",
		},
	)

	return err
}

// Netrc returns a netrc file capable of authenticating Gitea requests and
// cloning Gitea repositories. The netrc will use the global machine account
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
// the Gitea repository.
func (c *client) Activate(u *model.User, r *model.Repo, link string) error {
	config := map[string]string{
		"url":          link,
		"secret":       r.Hash,
		"content_type": "json",
	}
	hook := gitea.CreateHookOption{
		Type:   "gitea",
		Config: config,
		Events: []string{"push", "create", "pull_request"},
		Active: true,
	}

	client := c.newClientToken(u.Token)
	_, err := client.CreateRepoHook(r.Owner, r.Name, hook)
	return err
}

// Deactivate deactives the repository be removing repository push hooks from
// the Gitea repository.
func (c *client) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := c.newClientToken(u.Token)

	hooks, err := client.ListRepoHooks(r.Owner, r.Name)
	if err != nil {
		return err
	}

	hook := matchingHooks(hooks, link)
	if hook != nil {
		return client.DeleteRepoHook(r.Owner, r.Name, hook.ID)
	}

	return nil
}

// Hook parses the incoming Gitea hook and returns the Repository and Build
// details. If the hook is unsupported nil values are returned.
func (c *client) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	return parseHook(r)
}

// helper function to return the Gitea client
func (c *client) newClient() *gitea.Client {
	return c.newClientToken("")
}

// helper function to return the Gitea client
func (c *client) newClientToken(token string) *gitea.Client {
	client := gitea.NewClient(c.URL, token)
	if c.SkipVerify {
		httpClient := &http.Client{}
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.SetHTTPClient(httpClient)
	}
	return client
}

// helper function to return matching hooks.
func matchingHooks(hooks []*gitea.Hook, rawurl string) *gitea.Hook {
	link, err := url.Parse(rawurl)
	if err != nil {
		return nil
	}
	for _, hook := range hooks {
		if val, ok := hook.Config["url"]; ok {
			hookurl, err := url.Parse(val)
			if err == nil && hookurl.Host == link.Host {
				return hook
			}
		}
	}
	return nil
}
