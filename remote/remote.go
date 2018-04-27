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

package remote

//go:generate mockery -name Remote -output mock -case=underscore

import (
	"net/http"
	"time"

	"github.com/drone/drone/model"

	"golang.org/x/net/context"
)

type Remote interface {
	// Login authenticates the session and returns the
	// remote user details.
	Login(w http.ResponseWriter, r *http.Request) (*model.User, error)

	// Auth authenticates the session and returns the remote user
	// login for the given token and secret
	Auth(token, secret string) (string, error)

	// Teams fetches a list of team memberships from the remote system.
	Teams(u *model.User) ([]*model.Team, error)

	// Repo fetches the named repository from the remote system.
	Repo(u *model.User, owner, repo string) (*model.Repo, error)

	// Repos fetches a list of repos from the remote system.
	Repos(u *model.User) ([]*model.Repo, error)

	// Perm fetches the named repository permissions from
	// the remote system for the specified user.
	Perm(u *model.User, owner, repo string) (*model.Perm, error)

	// File fetches a file from the remote repository and returns in string
	// format.
	File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error)

	// FileRef fetches a file from the remote repository for the given ref
	// and returns in string format.
	FileRef(u *model.User, r *model.Repo, ref, f string) ([]byte, error)

	// Status sends the commit status to the remote system.
	// An example would be the GitHub pull request status.
	Status(u *model.User, r *model.Repo, b *model.Build, link string) error

	// Netrc returns a .netrc file that can be used to clone
	// private repositories from a remote system.
	Netrc(u *model.User, r *model.Repo) (*model.Netrc, error)

	// Activate activates a repository by creating the post-commit hook.
	Activate(u *model.User, r *model.Repo, link string) error

	// Deactivate deactivates a repository by removing all previously created
	// post-commit hooks matching the given link.
	Deactivate(u *model.User, r *model.Repo, link string) error

	// Hook parses the post-commit hook from the Request body and returns the
	// required data in a standard format.
	Hook(r *http.Request) (*model.Repo, *model.Build, error)
}

// Refresher refreshes an oauth token and expiration for the given user. It
// returns true if the token was refreshed, false if the token was not refreshed,
// and error if it failed to refersh.
type Refresher interface {
	Refresh(*model.User) (bool, error)
}

// Login authenticates the session and returns the
// remote user details.
func Login(c context.Context, w http.ResponseWriter, r *http.Request) (*model.User, error) {
	return FromContext(c).Login(w, r)
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func Auth(c context.Context, token, secret string) (string, error) {
	return FromContext(c).Auth(token, secret)
}

// Teams fetches a list of team memberships from the remote system.
func Teams(c context.Context, u *model.User) ([]*model.Team, error) {
	return FromContext(c).Teams(u)
}

// Repo fetches the named repository from the remote system.
func Repo(c context.Context, u *model.User, owner, repo string) (*model.Repo, error) {
	return FromContext(c).Repo(u, owner, repo)
}

// Repos fetches a list of repos from the remote system.
func Repos(c context.Context, u *model.User) ([]*model.Repo, error) {
	return FromContext(c).Repos(u)
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func Perm(c context.Context, u *model.User, owner, repo string) (*model.Perm, error) {
	return FromContext(c).Perm(u, owner, repo)
}

// File fetches a file from the remote repository and returns in string format.
func File(c context.Context, u *model.User, r *model.Repo, b *model.Build, f string) (out []byte, err error) {
	for i := 0; i < 12; i++ {
		out, err = FromContext(c).File(u, r, b, f)
		if err == nil {
			return
		}
		time.Sleep(5 * time.Second)
	}
	return
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func Status(c context.Context, u *model.User, r *model.Repo, b *model.Build, link string) error {
	return FromContext(c).Status(u, r, b, link)
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func Netrc(c context.Context, u *model.User, r *model.Repo) (*model.Netrc, error) {
	return FromContext(c).Netrc(u, r)
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func Activate(c context.Context, u *model.User, r *model.Repo, link string) error {
	return FromContext(c).Activate(u, r, link)
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func Deactivate(c context.Context, u *model.User, r *model.Repo, link string) error {
	return FromContext(c).Deactivate(u, r, link)
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func Hook(c context.Context, r *http.Request) (*model.Repo, *model.Build, error) {
	return FromContext(c).Hook(r)
}

// Refresh refreshes an oauth token and expiration for the given
// user. It returns true if the token was refreshed, false if the
// token was not refreshed, and error if it failed to refersh.
func Refresh(c context.Context, u *model.User) (bool, error) {
	remote := FromContext(c)
	refresher, ok := remote.(Refresher)
	if !ok {
		return false, nil
	}
	return refresher.Refresh(u)
}

// FileBackoff fetches the file using an exponential backoff.
// TODO replace this with a proper backoff
func FileBackoff(remote Remote, u *model.User, r *model.Repo, b *model.Build, f string) (out []byte, err error) {
	for i := 0; i < 5; i++ {
		select {
		case <-time.After(time.Second * time.Duration(i)):
			out, err = remote.File(u, r, b, f)
			if err == nil {
				return
			}
		}
	}
	return
}
