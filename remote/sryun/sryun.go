package sryun

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote/sryun/git"
	"github.com/drone/drone/remote/sryun/yaml"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/shared/poller"
	"github.com/drone/drone/store"

	log "github.com/Sirupsen/logrus"
)

const (
	fullName = "leonlee"
	name     = "docker-2048"
	repoLink = "https://omdev.riderzen.com:10080/leonlee/docker-2048.git"
	clone    = "https://omdev.riderzen.com:10080/leonlee/docker-2048.git"
	branch   = "master"
)

var (
	//ErrNoBuildNeed don't need to build
	ErrNoBuildNeed = errors.New("no build need")
	//ErrBadCommit bad commit
	ErrBadCommit = errors.New("bad commit")
	//ErrBadScript bad script
	ErrBadScript = errors.New("bad script")
)

//Sryun model
type Sryun struct {
	User       *model.User
	Password   string
	Workspace  string
	ScriptName string
	SecName    string
	Registry   string
	Insecure   bool
	Storage    string
}

// Load create Sryun by env, impl of Remote interface
func Load(env envconfig.Env) *Sryun {
	log.Infoln("Loading sryun driver...")

	login := env.String("RC_SRY_USER", "sryadmin")
	password := env.String("RC_SRY_PWD", "sryun-pwd")
	token := env.String("RC_SRY_TOKEN", "EFDDF4D3-2EB9-400F-BA83-4A9D292A1170")
	email := env.String("RC_SRY_EMAIL", "sryadmin@dataman-inc.net")
	avatar := env.String("RC_SRY_AVATAR", "https://avatars3.githubusercontent.com/u/76609?v=3&s=460")
	workspace := env.String("RC_SRY_WORKSPACE", "/var/lib/drone/ws/")
	scriptName := env.String("RC_SRY_SCRIPT", ".sryci.yaml")
	secName := env.String("RC_SRY_SEC", ".sryci.sec")
	registry := env.String("RC_SRY_REG_HOST", "")
	storage := env.String("RC_SRY_DOCKER_STORAGE", "aufs")
	insecure := env.Bool("RC_SRY_REG_INSECURE", false)

	user := model.User{}
	user.Token = token
	user.Login = login
	user.Email = email
	user.Avatar = avatar

	sryun := Sryun{
		User:       &user,
		Password:   password,
		Workspace:  workspace,
		ScriptName: scriptName,
		SecName:    secName,
		Registry:   registry,
		Storage:    storage,
		Insecure:   insecure,
	}

	sryunJSON, _ := json.Marshal(sryun)
	log.Infoln(string(sryunJSON))

	log.Infoln("loaded sryun remote driver")

	return &sryun
}

// Login authenticates the session and returns the
// remote user details.
func (sry *Sryun) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	username := req.FormValue("username")
	password := req.FormValue("password")

	log.Infoln("got", username, "/", password)

	if username == sry.User.Login && password == sry.Password {
		return sry.User, true, nil
	}
	return nil, false, errors.New("bad auth")
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
	repo.Name = name
	repo.FullName = fmt.Sprintf("%s/%s", owner, name)
	repo.Link = repoLink
	repo.IsPrivate = true
	repo.Clone = clone
	repo.Branch = branch
	repo.Avatar = sry.User.Avatar
	repo.Kind = model.RepoGit

	return repo, nil
}

// RepoSryun fetches the named repository from the remote system.
func (sry *Sryun) RepoSryun(u *model.User, owner, name string, repo *model.Repo) (*model.Repo, error) {
	repo.FullName = fmt.Sprintf("%s/%s", owner, name)
	repo.IsPrivate = true
	repo.Avatar = sry.User.Avatar
	repo.Kind = model.RepoGit
	repo.AllowPull = true
	repo.AllowDeploy = true
	if !repo.AllowTag && !repo.AllowPush {
		repo.AllowPush = true
	}

	return repo, nil
}

// Repos fetches a list of repos from the remote system.
func (sry *Sryun) Repos(user *model.User) ([]*model.RepoLite, error) {
	repo := &model.RepoLite{
		Owner:    sry.User.Login,
		Name:     name,
		FullName: fullName,
		Avatar:   sry.User.Avatar,
	}
	return []*model.RepoLite{repo}, nil
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (sry *Sryun) Perm(user *model.User, owner, name string) (*model.Perm, error) {
	m := &model.Perm{
		Admin: true,
		Pull:  true,
		Push:  false,
	}

	return m, nil
}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (sry *Sryun) Script(user *model.User, repo *model.Repo, build *model.Build) ([]byte, []byte, error) {
	client, err := git.NewClient(repo.Clone, repo.Branch)
	if err != nil {
		return nil, nil, err
	}
	err = client.InitRepo(sry.Workspace, fmt.Sprintf("%d_%s_%s", repo.ID, repo.Owner, repo.Name))
	if err != nil {
		return nil, nil, err
	}
	err = client.FetchRef(build.Ref)
	if err != nil {
		return nil, nil, err
	}
	script, err := client.ShowFile(build.Commit, sry.ScriptName)
	if err != nil {
		return nil, nil, err
	}
	sec, err := client.ShowFile(build.Commit, sry.SecName)
	if err != nil {
		sec = nil
	}

	log.Infoln("old script\n", string(script))
	script, err = yaml.GenScript(repo, build, script, sry.Insecure, sry.Registry, sry.Storage)
	if err != nil {
		return nil, nil, err
	}

	log.Infoln("script\n", string(script))

	return script, sec, nil
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

//ActivateRepo activate repo, schedule polling
func (sry *Sryun) ActivateRepo(c *gin.Context, user *model.User, repo *model.Repo, key *model.Key, link string, period uint64) error {
	if period < 5 {
		period = 5
	}
	poller.Ref().AddPoll(repo, period*60)
	return nil
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (sry *Sryun) Deactivate(user *model.User, repo *model.Repo, link string) error {
	return poller.Ref().DeletePoll(repo)
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (sry *Sryun) Hook(req *http.Request) (*model.Repo, *model.Build, error) {
	return nil, nil, nil
}

//SryunHook manual hook for sryun cloud
func (sry *Sryun) SryunHook(c *gin.Context) (*model.Repo, *model.Build, error) {
	params := poller.Params{}
	err := c.Bind(&params)
	if err != nil {
		log.Errorln("bad params")
		return nil, nil, err
	}
	log.Infoln("hook params %q", params)

	repo, err := store.GetRepoOwnerName(c, params.Owner, params.Name)
	if err != nil {
		return nil, nil, err
	}

	push, tag, err := retrieveUpdate(repo)
	if err != nil {
		return nil, nil, err
	}
	lastBuild, err := store.GetBuildLast(c, repo, branch)
	if err != nil {
		log.Printf("no build found")
	}
	if lastBuild != nil {
		log.Infof("lastBuild %q", *lastBuild)
	}
	build, err := formBuild(lastBuild, repo, push, tag, params.Force)
	if err != nil {
		return nil, nil, err
	}

	return repo, build, nil
}

func retrieveUpdate(repo *model.Repo) (*git.Reference, *git.Reference, error) {
	client, err := git.NewClient(repo.Clone, repo.Branch)
	if err != nil {
		return nil, nil, err
	}
	var filter uint8
	if repo.AllowTag {
		filter = filter + git.FilterTags
	}
	if repo.AllowPush {
		filter = filter + git.FilterHeads
	}

	push, tag, err := client.LsRemote(filter, "")
	if err != nil {
		return nil, nil, err
	}
	log.Println("push", push, "tag", tag)

	return push, tag, nil
}

func formBuild(lastBuild *model.Build, repo *model.Repo, push *git.Reference, tag *git.Reference, force bool) (*model.Build, error) {
	tagUpdated := isUpdated(lastBuild, tag)
	pushUpdated := isUpdated(lastBuild, push)
	log.Infoln("tagUpdated", tagUpdated, "pushUpdated", pushUpdated)

	if force || tagUpdated || pushUpdated {
		ref, commit, err := determineRef(repo, lastBuild, tag, push, tagUpdated, pushUpdated)
		log.Infoln("determined ref", ref, "commit", commit)
		if err != nil {
			return nil, err
		}
		build := &model.Build{
			Event:     model.EventPush, // for getting correct latest build// determineEvent(tagUpdated, pushUpdated),
			Commit:    commit,
			Ref:       ref,
			Link:      "",
			Branch:    repo.Branch,
			Message:   "",
			Avatar:    "",
			Author:    "",
			Timestamp: time.Now().UTC().Unix(),
		}
		return build, nil
	}
	return nil, ErrNoBuildNeed
}

func isUpdated(build *model.Build, reference *git.Reference) bool {
	if build == nil {
		return true
	}
	if reference == nil {
		return false
	}
	updated := build.Commit != reference.Commit

	if !updated && isTag(reference.Ref) {
		updated = build.Ref != reference.Ref
	}
	return updated
}

func isTag(ref string) bool {
	return strings.HasPrefix(ref, "refs/tags")
}

func determineEvent(tagUpdated bool, pushUpdated bool) string {
	event := model.EventDeploy
	if tagUpdated {
		event = model.EventTag
	}
	if pushUpdated {
		event = model.EventPush
	}
	return event
}

func determineRef(repo *model.Repo, build *model.Build, tag, push *git.Reference, tagUpdated, pushUpdated bool) (string, string, error) {
	/*if tagUpdated && pushUpdated {
		client, err := git.NewClient(repo.Clone, repo.Branch)
		if err != nil {
			return "", "", err
		}
		err = client.InitRepo(sry.Workspace, fmt.Sprintf("%d_%s_%s", repo.ID, repo.Owner, repo.Name))
		if err != nil {
			return nil, nil, err
		}
		err = client.FetchRef(build.Ref)
		if err != nil {
			return nil, nil, err
		}
		timestamps, err := client.ShowTimestamps(tag.Commit, push.Commit)
		if err != nil {
			return "", "", err
		}
		log.Infof("got timestamps %q", timestamps)
		if timestamps[0] > timestamps[1] {
			return tag.Ref, tag.Commit, nil
		} else {
			return push.Ref, push.Commit, nil
		}
	}*/

	if tagUpdated && tag != nil {
		return tag.Ref, tag.Commit, nil
	}
	if pushUpdated && push != nil {
		return push.Ref, push.Commit, nil
	}
	if build != nil {
		return build.Ref, build.Commit, nil
	}

	return "", "", ErrBadCommit
}
