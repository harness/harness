package gerrit

import (
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
)

// IMPORTANT Gerrit support is not yet implemented. This file is a placeholder
// for future implementaiton.

// Opts defines configuration options.
type Opts struct {
	URL         string // Gerrit server url.
	Username    string // Optional machine account username.
	Password    string // Optional machine account password.
	PrivateMode bool   // Gerrit is running in private mode.
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

// New returns a Remote implementation that integrates with Getter, an open
// source Git hosting service and code review system.
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

// Login authenticates an account with Gerrit using oauth authenticaiton. The
// Gerrit account details are returned when the user is successfully authenticated.
func (c *client) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	return nil, nil
}

// Auth is not supported by the Gerrit driver.
func (c *client) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Not Implemented")
}

// Teams is not supported by the Gerrit driver.
func (c *client) Teams(u *model.User) ([]*model.Team, error) {
	var empty []*model.Team
	return empty, nil
}

// TeamPerm is not supported by the Gerrit driver.
func (c *client) TeamPerm(u *model.User, org string) (*model.Perm, error) {
	return nil, nil
}

// Repo is not supported by the Gerrit driver.
func (c *client) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	return nil, nil
}

// Repos is not supported by the Gerrit driver.
func (c *client) Repos(u *model.User) ([]*model.RepoLite, error) {
	return nil, nil
}

// Perm is not supported by the Gerrit driver.
func (c *client) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	return nil, nil
}

// File is not supported by the Gerrit driver.
func (c *client) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	return nil, nil
}

// File is not supported by the Gerrit driver.
func (c *client) FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error) {
	return nil, nil
}

// Status is not supported by the Gogs driver.
func (c *client) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return nil
}

// Netrc is not supported by the Gerrit driver.
func (c *client) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return nil, nil
}

// Activate is not supported by the Gerrit driver.
func (c *client) Activate(u *model.User, r *model.Repo, link string) error {
	return nil
}

// Deactivate is not supported by the Gogs driver.
func (c *client) Deactivate(u *model.User, r *model.Repo, link string) error {
	return nil
}

// Hook is not supported by the Gerrit driver.
func (c *client) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	return nil, nil, nil
}
