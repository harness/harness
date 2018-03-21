// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package coding

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/coding/internal"
	"github.com/drone/drone/shared/httputil"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const (
	defaultURL = "https://coding.net" // Default Coding URL
)

// Opts defines configuration options.
type Opts struct {
	URL        string   // Coding server url.
	Client     string   // Coding oauth client id.
	Secret     string   // Coding oauth client secret.
	Scopes     []string // Coding oauth scopes.
	Machine    string   // Optional machine name.
	Username   string   // Optional machine account username.
	Password   string   // Optional machine account password.
	SkipVerify bool     // Skip ssl verification.
}

// New returns a Remote implementation that integrates with a Coding Platform or
// Coding Enterprise version control hosting provider.
func New(opts Opts) (remote.Remote, error) {
	remote := &Coding{
		URL:        defaultURL,
		Client:     opts.Client,
		Secret:     opts.Secret,
		Scopes:     opts.Scopes,
		Machine:    opts.Machine,
		Username:   opts.Username,
		Password:   opts.Password,
		SkipVerify: opts.SkipVerify,
	}
	if opts.URL != defaultURL {
		remote.URL = strings.TrimSuffix(opts.URL, "/")
	}

	// Hack to enable oauth2 access in coding's implementation
	oauth2.RegisterBrokenAuthHeaderProvider(remote.URL)
	return remote, nil
}

type Coding struct {
	URL        string
	Client     string
	Secret     string
	Scopes     []string
	Machine    string
	Username   string
	Password   string
	SkipVerify bool
}

// Login authenticates the session and returns the
// remote user details.
func (c *Coding) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	config := c.newConfig(httputil.GetURL(req))

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
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, nil
	}

	token, err := config.Exchange(c.newContext(), code)
	if err != nil {
		return nil, err
	}

	user, err := c.newClientToken(token.AccessToken, token.RefreshToken).GetCurrentUser()
	if err != nil {
		return nil, err
	}

	return &model.User{
		Login:  user.GlobalKey,
		Email:  user.Email,
		Token:  token.AccessToken,
		Secret: token.RefreshToken,
		Expiry: token.Expiry.UTC().Unix(),
		Avatar: c.resourceLink(user.Avatar),
	}, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (c *Coding) Auth(token, secret string) (string, error) {
	user, err := c.newClientToken(token, secret).GetCurrentUser()
	if err != nil {
		return "", err
	}
	return user.GlobalKey, nil
}

// Refresh refreshes an oauth token and expiration for the given
// user. It returns true if the token was refreshed, false if the
// token was not refreshed, and error if it failed to refersh.
func (c *Coding) Refresh(u *model.User) (bool, error) {
	config := c.newConfig("")
	source := config.TokenSource(c.newContext(), &oauth2.Token{RefreshToken: u.Secret})
	token, err := source.Token()
	if err != nil || len(token.AccessToken) == 0 {
		return false, err
	}

	u.Token = token.AccessToken
	u.Secret = token.RefreshToken
	u.Expiry = token.Expiry.UTC().Unix()
	return true, nil
}

// Teams fetches a list of team memberships from the remote system.
func (c *Coding) Teams(u *model.User) ([]*model.Team, error) {
	// EMPTY: not implemented in Coding OAuth API
	return nil, nil
}

// TeamPerm fetches the named organization permissions from
// the remote system for the specified user.
func (c *Coding) TeamPerm(u *model.User, org string) (*model.Perm, error) {
	// EMPTY: not implemented in Coding OAuth API
	return nil, nil
}

// Repo fetches the named repository from the remote system.
func (c *Coding) Repo(u *model.User, owner, repo string) (*model.Repo, error) {
	project, err := c.newClient(u).GetProject(owner, repo)
	if err != nil {
		return nil, err
	}
	depot, err := c.newClient(u).GetDepot(owner, repo)
	if err != nil {
		return nil, err
	}
	return &model.Repo{
		Owner:     project.Owner,
		Name:      project.Name,
		FullName:  projectFullName(project.Owner, project.Name),
		Avatar:    c.resourceLink(project.Icon),
		Link:      c.resourceLink(project.DepotPath),
		Kind:      model.RepoGit,
		Clone:     project.HttpsURL,
		Branch:    depot.DefaultBranch,
		IsPrivate: !project.IsPublic,
	}, nil
}

// Repos fetches a list of repos from the remote system.
func (c *Coding) Repos(u *model.User) ([]*model.Repo, error) {
	projectList, err := c.newClient(u).GetProjectList()
	if err != nil {
		return nil, err
	}

	repos := make([]*model.Repo, 0)
	for _, project := range projectList {
		depot, err := c.newClient(u).GetDepot(project.Owner, project.Name)
		if err != nil {
			return nil, err
		}
		repo := &model.Repo{
			Owner:     project.Owner,
			Name:      project.Name,
			FullName:  projectFullName(project.Owner, project.Name),
			Avatar:    c.resourceLink(project.Icon),
			Link:      c.resourceLink(project.DepotPath),
			Kind:      model.RepoGit,
			Clone:     project.HttpsURL,
			Branch:    depot.DefaultBranch,
			IsPrivate: !project.IsPublic,
		}
		repos = append(repos, repo)
	}
	return repos, nil
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (c *Coding) Perm(u *model.User, owner, repo string) (*model.Perm, error) {
	project, err := c.newClient(u).GetProject(owner, repo)
	if err != nil {
		return nil, err
	}

	if project.Role == "owner" || project.Role == "admin" {
		return &model.Perm{Pull: true, Push: true, Admin: true}, nil
	}
	if project.Role == "member" {
		return &model.Perm{Pull: true, Push: true, Admin: false}, nil
	}
	return &model.Perm{Pull: false, Push: false, Admin: false}, nil
}

// File fetches a file from the remote repository and returns in string
// format.
func (c *Coding) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	data, err := c.newClient(u).GetFile(r.Owner, r.Name, b.Commit, f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// FileRef fetches a file from the remote repository for the given ref
// and returns in string format.
func (c *Coding) FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error) {
	data, err := c.newClient(u).GetFile(r.Owner, r.Name, ref, f)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// Status sends the commit status to the remote system.
func (c *Coding) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	// EMPTY: not implemented in Coding OAuth API
	return nil
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (c *Coding) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
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

// Activate activates a repository by creating the post-commit hook.
func (c *Coding) Activate(u *model.User, r *model.Repo, link string) error {
	return c.newClient(u).AddWebhook(r.Owner, r.Name, link)
}

// Deactivate deactivates a repository by removing all previously created
// post-commit hooks matching the given link.
func (c *Coding) Deactivate(u *model.User, r *model.Repo, link string) error {
	return c.newClient(u).RemoveWebhook(r.Owner, r.Name, link)
}

// Hook parses the post-commit hook from the Request body and returns the
// required data in a standard format.
func (c *Coding) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	repo, build, err := parseHook(r)
	if build != nil {
		build.Avatar = c.resourceLink(build.Avatar)
	}
	return repo, build, err
}

// helper function to return the Coding oauth2 context using an HTTPClient that
// disables TLS verification if disabled in the remote settings.
func (c *Coding) newContext() context.Context {
	if !c.SkipVerify {
		return oauth2.NoContext
	}
	return context.WithValue(nil, oauth2.HTTPClient, &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	})
}

// helper function to return the Coding oauth2 config
func (c *Coding) newConfig(redirect string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Scopes:       []string{strings.Join(c.Scopes, ",")},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("%s/oauth_authorize.html", c.URL),
			TokenURL: fmt.Sprintf("%s/api/oauth/access_token_v2", c.URL),
		},
		RedirectURL: fmt.Sprintf("%s/authorize", redirect),
	}
}

// helper function to return the Coding oauth2 client
func (c *Coding) newClient(u *model.User) *internal.Client {
	return c.newClientToken(u.Token, u.Secret)
}

// helper function to return the Coding oauth2 client
func (c *Coding) newClientToken(token, secret string) *internal.Client {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: c.SkipVerify,
			},
		},
	}
	return internal.NewClient(c.URL, "/api", token, "drone", client)
}

func (c *Coding) resourceLink(resourcePath string) string {
	if strings.HasPrefix(resourcePath, "http") {
		return resourcePath
	}
	return c.URL + resourcePath
}
