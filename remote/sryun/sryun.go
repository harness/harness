package sryun

import (
	"net/http"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"

	log "github.com/Sirupsen/logrus"
)

const (
	login    = "sysadmin"
	token    = "SRYUN-ABCD-999"
	email    = "sysadmin@dataman-inc.com"
	avatar   = "https://avatars3.githubusercontent.com/u/76609?v=3&s=460"
	fullName = "leonlee"
	name     = "docker-2048"
	repoLink = "https://omdev.riderzen.com:10080/leonlee/docker-2048.git"
	clone    = "https://omdev.riderzen.com:10080/leonlee/docker-2048.git"
	branch   = "master"
)

//Sryun model
type Sryun struct {
	User *model.User
}

// Load create Sryun by env, impl of Remote interface
func Load(env envconfig.Env) *Sryun {
	user := model.User{}
	user.Token = token
	user.Login = login
	user.Email = email
	user.Avatar = avatar

	sryun := Sryun{
		User: &user,
	}

	log.Infoln("loaded sryun remote driver")

	return &sryun
}

// Login authenticates the session and returns the
// remote user details.
func (sry *Sryun) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	return sry.User, false, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (sry *Sryun) Auth(token, secret string) (string, error) {
	return sry.User.Login, nil
}

// Repo fetches the named repository from the remote system.
func (sry *Sryun) Repo(user *model.User, owner, name string) (*model.Repo, error) {
	repo := &model.Repo{}
	repo.Owner = owner
	repo.FullName = fullName
	repo.Link = repoLink
	repo.IsPrivate = true
	repo.Clone = clone
	repo.Branch = branch
	repo.Avatar = avatar
	repo.Kind = model.RepoGit

	return repo, nil
}

// Repos fetches a list of repos from the remote system.
func (sry *Sryun) Repos(user *model.User) ([]*model.RepoLite, error) {
	repo := &model.RepoLite{
		Owner:    login,
		Name:     name,
		FullName: fullName,
		Avatar:   avatar,
	}
	return []*model.RepoLite{repo}, nil
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (sry *Sryun) Perm(user *model.User, owner, name string) (*model.Perm, error) {
	m := &model.Perm{
		Admin: false,
		Pull:  true,
		Push:  false,
	}

	return m, nil
}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (sry *Sryun) Script(user *model.User, repo *model.Repo, build *model.Build) ([]byte, []byte, error) {
	cfg := `
clone:
  skip_verify: true
build:
  image: alpine:latest
  commands:
    - echo 'done'
publish:
  docker:
    username: blackicebird
    password: youman
    email: blackicebird@126.com
    repo: blackicebird/hello-2048
    tag:
      - latest
    load: docker/hello-2048.tar
    save:
      destination: docker/hello-2048.tar
      tag: latest
cache:
  mount:
    - docker/hello-2048.tar
	`

	sec := ""
	return []byte(cfg), []byte(sec), nil
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (sry *Sryun) Status(user *model.User, repo *model.Repo, build *model.Build, link string) error {
	return nil
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (sry *Sryun) Netrc(user *model.User, repo *model.Repo) (*model.Netrc, error) {
	netrc := &model.Netrc{}
	return netrc, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (sry *Sryun) Activate(user *model.User, repo *model.Repo, key *model.Key, link string) error {
	return nil
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (sry *Sryun) Deactivate(user *model.User, repo *model.Repo, link string) error {
	return nil
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (sry *Sryun) Hook(req *http.Request) (*model.Repo, *model.Build, error) {
	return nil, nil, nil
}
