package bitbucket

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/bitbucket/internal"
	"github.com/drone/drone/shared/httputil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

// Bitbucket Server endpoint.
const Endpoint = "https://api.bitbucket.org"

type config struct {
	URL    string
	Client string
	Secret string
}

// New returns a new remote Configuration for integrating with the Bitbucket
// repository hosting service at https://bitbucket.org
func New(client, secret string) remote.Remote {
	return &config{
		URL:    Endpoint,
		Client: client,
		Secret: secret,
	}
}

// helper function to return the bitbucket oauth2 client
func (c *config) newClient(u *model.User) *internal.Client {
	return internal.NewClientToken(
		c.URL,
		c.Client,
		c.Secret,
		&oauth2.Token{
			AccessToken:  u.Token,
			RefreshToken: u.Secret,
		},
	)
}

func (c *config) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	config := &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint:     bitbucket.Endpoint,
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	var code = req.FormValue("code")
	if len(code) == 0 {
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, nil
	}

	var token, err = config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	client := internal.NewClient(c.URL, config.Client(oauth2.NoContext, token))
	curr, err := client.FindCurrent()
	if err != nil {
		return nil, err
	}

	return convertUser(curr, token), nil
}

func (c *config) Auth(token, secret string) (string, error) {
	client := internal.NewClientToken(
		c.URL,
		c.Client,
		c.Secret,
		&oauth2.Token{
			AccessToken:  token,
			RefreshToken: secret,
		},
	)
	user, err := client.FindCurrent()
	if err != nil {
		return "", err
	}
	return user.Login, nil
}

func (c *config) Refresh(user *model.User) (bool, error) {
	config := &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint:     bitbucket.Endpoint,
	}

	// creates a token source with just the refresh token.
	// this will ensure an access token is automatically
	// requested.
	source := config.TokenSource(
		oauth2.NoContext, &oauth2.Token{RefreshToken: user.Secret})

	// requesting the token automatically refreshes and
	// returns a new access token.
	token, err := source.Token()
	if err != nil || len(token.AccessToken) == 0 {
		return false, err
	}

	// update the user to include tne new access token
	user.Token = token.AccessToken
	user.Secret = token.RefreshToken
	user.Expiry = token.Expiry.UTC().Unix()
	return true, nil
}

func (c *config) Teams(u *model.User) ([]*model.Team, error) {
	opts := &internal.ListTeamOpts{
		PageLen: 100,
		Role:    "member",
	}
	resp, err := c.newClient(u).ListTeams(opts)
	if err != nil {
		return nil, err
	}
	return convertTeamList(resp.Values), nil
}

func (c *config) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	repo, err := c.newClient(u).FindRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return convertRepo(repo), nil
}

func (c *config) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := c.newClient(u)

	var repos []*model.RepoLite

	// gets a list of all accounts to query, including the
	// user's account and all team accounts.
	logins := []string{u.Login}
	resp, err := client.ListTeams(&internal.ListTeamOpts{PageLen: 100, Role: "member"})
	if err != nil {
		return repos, err
	}
	for _, team := range resp.Values {
		logins = append(logins, team.Login)
	}

	// for each account, get the list of repos
	for _, login := range logins {
		repos_, err := client.ListReposAll(login)
		if err != nil {
			return repos, err
		}
		for _, repo := range repos_ {
			repos = append(repos, convertRepoLite(repo))
		}
	}

	return repos, nil
}

func (c *config) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClient(u)

	perms := new(model.Perm)
	_, err := client.FindRepo(owner, name)
	if err != nil {
		return perms, err
	}

	// if we've gotten this far we know that the user at least has read access
	// to the repository.
	perms.Pull = true

	// if the user has access to the repository hooks we can deduce that the user
	// has push and admin access.
	_, err = client.ListHooks(owner, name, &internal.ListOpts{})
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	return perms, nil
}

func (c *config) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	config, err := c.newClient(u).FindSource(r.Owner, r.Name, b.Commit, f)
	if err != nil {
		return nil, err
	}
	return []byte(config.Data), err
}

func (c *config) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	status := internal.BuildStatus{
		State: convertStatus(b.Status),
		Desc:  convertDesc(b.Status),
		Key:   "Drone",
		Url:   link,
	}
	return c.newClient(u).CreateStatus(r.Owner, r.Name, b.Commit, &status)
}

func (c *config) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return &model.Netrc{
		Machine:  "bitbucket.org",
		Login:    "x-token-auth",
		Password: u.Token,
	}, nil
}

func (c *config) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	rawurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	// deletes any previously created hooks
	c.Deactivate(u, r, link)

	return c.newClient(u).CreateHook(r.Owner, r.Name, &internal.Hook{
		Active: true,
		Desc:   rawurl.Host,
		Events: []string{"repo:push"},
		Url:    link,
	})
}

func (c *config) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := c.newClient(u)

	linkurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	hooks, err := client.ListHooks(r.Owner, r.Name, &internal.ListOpts{})
	if err != nil {
		return err
	}

	for _, hook := range hooks.Values {
		hookurl, err := url.Parse(hook.Url)
		if err == nil && hookurl.Host == linkurl.Host {
			return client.DeleteHook(r.Owner, r.Name, hook.Uuid)
		}
	}

	return nil
}

// Hook parses the incoming Bitbucket hook and returns the Repository and
// Build details. If the hook is unsupported nil values are returned and the
// hook should be skipped.
func (c *config) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	return parseHook(r)
}
