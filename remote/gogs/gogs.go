package gogs

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/gogits/go-gogs-client"

	log "github.com/Sirupsen/logrus"
)

type Gogs struct {
	URL         string
	Open        bool
	PrivateMode bool
	SkipVerify  bool
}

func Load(env envconfig.Env) *Gogs {
	config := env.String("REMOTE_CONFIG", "")

	// parse the remote DSN configuration string
	url_, err := url.Parse(config)
	if err != nil {
		log.Fatalln("unable to parse remote dsn. %s", err)
	}
	params := url_.Query()
	url_.Path = ""
	url_.RawQuery = ""

	// create the Githbub remote using parameters from
	// the parsed DSN configuration string.
	gogs := Gogs{}
	gogs.URL = url_.String()
	gogs.PrivateMode, _ = strconv.ParseBool(params.Get("private_mode"))
	gogs.SkipVerify, _ = strconv.ParseBool(params.Get("skip_verify"))
	gogs.Open, _ = strconv.ParseBool(params.Get("open"))

	return &gogs
}

// Login authenticates the session and returns the
// remote user details.
func (g *Gogs) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	var (
		username = req.FormValue("username")
		password = req.FormValue("password")
	)

	// if the username or password doesn't exist we re-direct
	// the user to the login screen.
	if len(username) == 0 || len(password) == 0 {
		http.Redirect(res, req, "/login/form", http.StatusSeeOther)
		return nil, false, nil
	}

	client := gogs.NewClient(g.URL, "")

	// try to fetch drone token if it exists
	var accessToken string
	tokens, err := client.ListAccessTokens(username, password)
	if err != nil {
		return nil, false, err
	}
	for _, token := range tokens {
		if token.Name == "drone" {
			accessToken = token.Sha1
			break
		}
	}

	// if drone token not found, create it
	if accessToken == "" {
		token, err := client.CreateAccessToken(username, password, gogs.CreateAccessTokenOption{Name: "drone"})
		if err != nil {
			return nil, false, err
		}
		accessToken = token.Sha1
	}

	client = gogs.NewClient(g.URL, accessToken)
	userInfo, err := client.GetUserInfo(username)
	if err != nil {
		return nil, false, err
	}

	user := model.User{}
	user.Token = accessToken
	user.Login = userInfo.UserName
	user.Email = userInfo.Email
	user.Avatar = expandAvatar(g.URL, userInfo.AvatarUrl)
	return &user, g.Open, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (g *Gogs) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Method not supported")
}

// Repo fetches the named repository from the remote system.
func (g *Gogs) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := gogs.NewClient(g.URL, u.Token)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return nil, err
	}

	fullName := owner + "/" + name
	for _, repo := range repos_ {
		if repo.FullName == fullName {
			return toRepo(repo), nil
		}
	}

	return nil, fmt.Errorf("Not Found")
}

// Repos fetches a list of repos from the remote system.
func (g *Gogs) Repos(u *model.User) ([]*model.RepoLite, error) {
	repos := []*model.RepoLite{}

	client := gogs.NewClient(g.URL, u.Token)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return repos, err
	}

	for _, repo := range repos_ {
		repos = append(repos, toRepoLite(repo))
	}

	return repos, err
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (g *Gogs) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := gogs.NewClient(g.URL, u.Token)
	repos_, err := client.ListMyRepos()
	if err != nil {
		return nil, err
	}

	fullName := owner + "/" + name
	for _, repo := range repos_ {
		if repo.FullName == fullName {
			return toPerm(repo.Permissions), nil
		}
	}

	return nil, fmt.Errorf("Not Found")

}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (g *Gogs) Script(u *model.User, r *model.Repo, b *model.Build) ([]byte, []byte, error) {
	client := gogs.NewClient(g.URL, u.Token)
	cfg, err := client.GetFile(r.Owner, r.Name, b.Commit, ".drone.yml")
	sec, _ := client.GetFile(r.Owner, r.Name, b.Commit, ".drone.sec")
	return cfg, sec, err
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (g *Gogs) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return fmt.Errorf("Not Implemented")
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *Gogs) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	url_, err := url.Parse(g.URL)
	if err != nil {
		return nil, err
	}
	return &model.Netrc{
		Login:    u.Token,
		Password: "x-oauth-basic",
		Machine:  url_.Host,
	}, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (g *Gogs) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	config := map[string]string{
		"url":          link,
		"secret":       r.Hash,
		"content_type": "json",
	}
	hook := gogs.CreateHookOption{
		Type:   "gogs",
		Config: config,
		Active: true,
	}

	client := gogs.NewClient(g.URL, u.Token)
	_, err := client.CreateRepoHook(r.Owner, r.Name, hook)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (g *Gogs) Deactivate(u *model.User, r *model.Repo, link string) error {
	return fmt.Errorf("Not Implemented")
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Gogs) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	var (
		err   error
		repo  *model.Repo
		build *model.Build
	)

	switch r.Header.Get("X-Gogs-Event") {
	case "push":
		var push *PushHook
		push, err = parsePush(r.Body)
		if err == nil {
			repo = repoFromPush(push)
			build = buildFromPush(push)
		}
	}
	return repo, build, err
}

func (g *Gogs) String() string {
	return "gogs"
}
