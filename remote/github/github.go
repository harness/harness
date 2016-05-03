package github

import (
	"crypto/tls"
	"encoding/base32"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/gorilla/securecookie"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	defaultURL = "https://github.com"     // Default GitHub URL
	defaultAPI = "https://api.github.com" // Default GitHub API URL
)

// Opts defines configuration options.
type Opts struct {
	URL         string   // GitHub server url.
	Client      string   // GitHub oauth client id.
	Secret      string   // GitHub oauth client secret.
	Scopes      []string // GitHub oauth scopes
	Username    string   // Optional machine account username.
	Password    string   // Optional machine account password.
	PrivateMode bool     // GitHub is running in private mode.
	SkipVerify  bool     // Skip ssl verification.
	MergeRef    bool     // Clone pull requests using the merge ref.
}

// New returns a Remote implementation that integrates with a GitHub Cloud or
// GitHub Enterprise version control hosting provider.
func New(opts Opts) (remote.Remote, error) {
	url, err := url.Parse(opts.URL)
	if err != nil {
		return nil, err
	}
	host, _, err := net.SplitHostPort(url.Host)
	if err == nil {
		url.Host = host
	}
	remote := &client{
		API:         defaultAPI,
		URL:         defaultURL,
		Client:      opts.Client,
		Secret:      opts.Secret,
		Scope:       strings.Join(opts.Scopes, ","),
		PrivateMode: opts.PrivateMode,
		SkipVerify:  opts.SkipVerify,
		MergeRef:    opts.MergeRef,
		Machine:     url.Host,
		Username:    opts.Username,
		Password:    opts.Password,
	}
	if opts.URL != defaultURL {
		remote.URL = strings.TrimSuffix(opts.URL, "/")
		remote.API = remote.URL + "/api/v3/"
	}
	return remote, nil
}

type client struct {
	URL         string
	API         string
	Client      string
	Secret      string
	Scope       string
	Machine     string
	Username    string
	Password    string
	PrivateMode bool
	SkipVerify  bool
	MergeRef    bool
}

// Login authenticates the session and returns the remote user details.
func (c *client) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	config := c.newConfig(httputil.GetURL(req))

	code := req.FormValue("code")
	if len(code) == 0 {
		rand := base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32))
		http.Redirect(res, req, config.AuthCodeURL(rand), http.StatusSeeOther)
		return nil, nil
	}

	// TODO(bradrydzewski) what is the best way to provide a SkipVerify flag
	// when exchanging the token?

	token, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	client := c.newClientToken(token.AccessToken)
	useremail, err := GetUserEmail(client)
	if err != nil {
		return nil, err
	}

	return &model.User{
		Login:  *useremail.Login,
		Email:  *useremail.Email,
		Token:  token.AccessToken,
		Avatar: *useremail.AvatarURL,
	}, nil
}

// Auth returns the GitHub user login for the given access token.
func (c *client) Auth(token, secret string) (string, error) {
	client := c.newClientToken(token)
	user, _, err := client.Users.Get("")
	if err != nil {
		return "", err
	}
	return *user.Login, nil
}

// Teams returns a list of all team membership for the GitHub account.
func (c *client) Teams(u *model.User) ([]*model.Team, error) {
	client := c.newClientToken(u.Token)

	opts := new(github.ListOptions)
	opts.Page = 1

	var teams []*model.Team
	for opts.Page > 0 {
		list, resp, err := client.Organizations.List("", opts)
		if err != nil {
			return nil, err
		}
		teams = append(teams, convertTeamList(list)...)
		opts.Page = resp.NextPage
	}
	return teams, nil
}

// Repo returns the named GitHub repository.
func (c *client) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := c.newClientToken(u.Token)
	repo, _, err := client.Repositories.Get(owner, name)
	if err != nil {
		return nil, err
	}
	return convertRepo(repo, c.PrivateMode), nil
}

// Repos returns a list of all repositories for GitHub account, including
// organization repositories.
func (c *client) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := c.newClientToken(u.Token)

	opts := new(github.RepositoryListOptions)
	opts.PerPage = 100
	opts.Page = 1

	var repos []*model.RepoLite
	for opts.Page > 0 {
		list, resp, err := client.Repositories.List("", opts)
		if err != nil {
			return nil, err
		}
		repos = append(repos, convertRepoList(list)...)
		opts.Page = resp.NextPage
	}
	return repos, nil
}

// Perm returns the user permissions for the named GitHub repository.
func (c *client) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClientToken(u.Token)
	repo, _, err := client.Repositories.Get(owner, name)
	if err != nil {
		return nil, err
	}
	return convertPerm(repo), nil
}

// File fetches the file from the Bitbucket repository and returns its contents.
func (c *client) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	client := c.newClientToken(u.Token)

	opts := new(github.RepositoryContentGetOptions)
	opts.Ref = b.Commit
	data, _, _, err := client.Repositories.GetContents(r.Owner, r.Name, f, opts)
	if err != nil {
		return nil, err
	}
	return data.Decode()
}

// Netrc returns a netrc file capable of authenticating GitHub requests and
// cloning GitHub repositories. The netrc will use the global machine account
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

// helper function to return the bitbucket oauth2 config
func (c *client) newConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/login/oauth/authorize", c.URL),
			TokenURL: fmt.Sprintf("%s/login/oauth/access_token", c.URL),
		},
		RedirectURL: fmt.Sprintf("%s/authorize", redirect),
	}
}

// helper function to return the bitbucket oauth2 client
func (c *client) newClientToken(token string) *github.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)
	if c.SkipVerify {
		tc.Transport.(*oauth2.Transport).Base = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	github := github.NewClient(tc)
	github.BaseURL, _ = url.Parse(c.API)
	return github
}

//
// TODO(bradrydzewski) refactor below functions
//

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (c *client) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	client := c.newClientToken(u.Token)
	switch b.Event {
	case "deployment":
		return deploymentStatus(client, r, b, link)
	default:
		return repoStatus(client, r, b, link)
	}
}

func repoStatus(client *github.Client, r *model.Repo, b *model.Build, link string) error {
	data := github.RepoStatus{
		Context:     github.String("continuous-integration/drone"),
		State:       github.String(convertStatus(b.Status)),
		Description: github.String(convertDesc(b.Status)),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateStatus(r.Owner, r.Name, b.Commit, &data)
	return err
}

var reDeploy = regexp.MustCompile(".+/deployments/(\\d+)")

func deploymentStatus(client *github.Client, r *model.Repo, b *model.Build, link string) error {
	matches := reDeploy.FindStringSubmatch(b.Link)
	if len(matches) != 2 {
		return nil
	}
	id, _ := strconv.Atoi(matches[1])

	data := github.DeploymentStatusRequest{
		State:       github.String(convertStatus(b.Status)),
		Description: github.String(convertDesc(b.Status)),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateDeploymentStatus(r.Owner, r.Name, id, &data)
	return err
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (c *client) Activate(u *model.User, r *model.Repo, link string) error {
	client := c.newClientToken(u.Token)
	_, err := CreateUpdateHook(client, r.Owner, r.Name, link)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (c *client) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := c.newClientToken(u.Token)
	return DeleteHook(client, r.Owner, r.Name, link)
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (c *client) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	switch r.Header.Get("X-Github-Event") {
	case "pull_request":
		return c.pullRequest(r)
	case "push":
		return c.push(r)
	case "deployment":
		return c.deployment(r)
	default:
		return nil, nil, nil
	}
}
