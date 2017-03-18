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
)

// Bitbucket cloud endpoints.
const (
	DefaultAPI = "https://api.bitbucket.org"
	DefaultURL = "https://bitbucket.org"
)

type config struct {
	API    string
	URL    string
	Client string
	Secret string
}

// New returns a new remote Configuration for integrating with the Bitbucket
// repository hosting service at https://bitbucket.org
func New(client, secret string) remote.Remote {
	return &config{
		API:    DefaultAPI,
		URL:    DefaultURL,
		Client: client,
		Secret: secret,
	}
}

// Login authenticates an account with Bitbucket using the oauth2 protocol. The
// Bitbucket account details are returned when the user is successfully authenticated.
func (c *config) Login(w http.ResponseWriter, req *http.Request) (*model.User, error) {
	redirect := httputil.GetURL(req)
	config := c.newConfig(redirect)

	// get the OAuth errors
	if err := req.FormValue("error"); err != "" {
		return nil, &remote.AuthError{
			Err:         err,
			Description: req.FormValue("error_description"),
			URI:         req.FormValue("error_uri"),
		}
	}

	// get the OAuth code
	code := req.FormValue("code")
	if len(code) == 0 {
		http.Redirect(w, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, nil
	}

	token, err := config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	client := internal.NewClient(c.API, config.Client(oauth2.NoContext, token))
	curr, err := client.FindCurrent()
	if err != nil {
		return nil, err
	}
	return convertUser(curr, token), nil
}

// Auth uses the Bitbucket oauth2 access token and refresh token to authenticate
// a session and return the Bitbucket account login.
func (c *config) Auth(token, secret string) (string, error) {
	client := c.newClientToken(token, secret)
	user, err := client.FindCurrent()
	if err != nil {
		return "", err
	}
	return user.Login, nil
}

// Refresh refreshes the Bitbucket oauth2 access token. If the token is
// refreshed the user is updated and a true value is returned.
func (c *config) Refresh(user *model.User) (bool, error) {
	config := c.newConfig("")
	source := config.TokenSource(
		oauth2.NoContext, &oauth2.Token{RefreshToken: user.Secret})

	token, err := source.Token()
	if err != nil || len(token.AccessToken) == 0 {
		return false, err
	}

	user.Token = token.AccessToken
	user.Secret = token.RefreshToken
	user.Expiry = token.Expiry.UTC().Unix()
	return true, nil
}

// Teams returns a list of all team membership for the Bitbucket account.
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

// TeamPerm is not supported by the Bitbucket driver.
func (c *config) TeamPerm(u *model.User, org string) (*model.Perm, error) {
	return nil, nil
}

// Repo returns the named Bitbucket repository.
func (c *config) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	repo, err := c.newClient(u).FindRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return convertRepo(repo), nil
}

// Repos returns a list of all repositories for Bitbucket account, including
// organization repositories.
func (c *config) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := c.newClient(u)

	var all []*model.RepoLite

	accounts := []string{u.Login}
	resp, err := client.ListTeams(&internal.ListTeamOpts{
		PageLen: 100,
		Role:    "member",
	})
	if err != nil {
		return all, err
	}
	for _, team := range resp.Values {
		accounts = append(accounts, team.Login)
	}

	for _, account := range accounts {
		repos, err := client.ListReposAll(account)
		if err != nil {
			return all, err
		}
		for _, repo := range repos {
			all = append(all, convertRepoLite(repo))
		}
	}
	return all, nil
}

// Perm returns the user permissions for the named repository. Because Bitbucket
// does not have an endpoint to access user permissions, we attempt to fetch
// the repository hook list, which is restricted to administrators to calculate
// administrative access to a repository.
func (c *config) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClient(u)

	perms := new(model.Perm)
	_, err := client.FindRepo(owner, name)
	if err != nil {
		return perms, err
	}

	_, err = client.ListHooks(owner, name, &internal.ListOpts{})
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	perms.Pull = true
	return perms, nil
}

// File fetches the file from the Bitbucket repository and returns its contents.
func (c *config) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	return c.FileRef(u, r, b.Commit, f)
}

// FileRef fetches the file from the Bitbucket repository and returns its contents.
func (c *config) FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error) {
	config, err := c.newClient(u).FindSource(r.Owner, r.Name, ref, f)
	if err != nil {
		return nil, err
	}
	return []byte(config.Data), err
}

// Status creates a build status for the Bitbucket commit.
func (c *config) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	status := internal.BuildStatus{
		State: convertStatus(b.Status),
		Desc:  convertDesc(b.Status),
		Key:   "Drone",
		Url:   link,
	}
	return c.newClient(u).CreateStatus(r.Owner, r.Name, b.Commit, &status)
}

// Activate activates the repository by registering repository push hooks with
// the Bitbucket repository. Prior to registering hook, previously created hooks
// are deleted.
func (c *config) Activate(u *model.User, r *model.Repo, link string) error {
	rawurl, err := url.Parse(link)
	if err != nil {
		return err
	}
	c.Deactivate(u, r, link)

	return c.newClient(u).CreateHook(r.Owner, r.Name, &internal.Hook{
		Active: true,
		Desc:   rawurl.Host,
		Events: []string{"repo:push"},
		Url:    link,
	})
}

// Deactivate deactives the repository be removing repository push hooks from
// the Bitbucket repository.
func (c *config) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := c.newClient(u)

	hooks, err := client.ListHooks(r.Owner, r.Name, &internal.ListOpts{})
	if err != nil {
		return err
	}
	hook := matchingHooks(hooks.Values, link)
	if hook != nil {
		return client.DeleteHook(r.Owner, r.Name, hook.Uuid)
	}
	return nil
}

// Netrc returns a netrc file capable of authenticating Bitbucket requests and
// cloning Bitbucket repositories.
func (c *config) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return &model.Netrc{
		Machine:  "bitbucket.org",
		Login:    "x-token-auth",
		Password: u.Token,
	}, nil
}

// Hook parses the incoming Bitbucket hook and returns the Repository and
// Build details. If the hook is unsupported nil values are returned.
func (c *config) Hook(req *http.Request) (*model.Repo, *model.Build, error) {
	return parseHook(req)
}

// helper function to return the bitbucket oauth2 client
func (c *config) newClient(u *model.User) *internal.Client {
	return c.newClientToken(u.Token, u.Secret)
}

// helper function to return the bitbucket oauth2 client
func (c *config) newClientToken(token, secret string) *internal.Client {
	return internal.NewClientToken(
		c.API,
		c.Client,
		c.Secret,
		&oauth2.Token{
			AccessToken:  token,
			RefreshToken: secret,
		},
	)
}

// helper function to return the bitbucket oauth2 config
func (c *config) newConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/site/oauth2/authorize", c.URL),
			TokenURL: fmt.Sprintf("%s/site/oauth2/access_token", c.URL),
		},
		RedirectURL: fmt.Sprintf("%s/authorize", redirect),
	}
}

// helper function to return matching hooks.
func matchingHooks(hooks []*internal.Hook, rawurl string) *internal.Hook {
	link, err := url.Parse(rawurl)
	if err != nil {
		return nil
	}
	for _, hook := range hooks {
		hookurl, err := url.Parse(hook.Url)
		if err == nil && hookurl.Host == link.Host {
			return hook
		}
	}
	return nil
}
