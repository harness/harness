package gitlab

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/oauth2"
	"github.com/drone/drone/shared/token"

	"github.com/Bugagazavr/go-gitlab-client"
)

const (
	DefaultScope = "api"
)

type Gitlab struct {
	URL         string
	Client      string
	Secret      string
	AllowedOrgs []string
	CloneMode   string
	Open        bool
	PrivateMode bool
	SkipVerify  bool
	Search      bool
}

func Load(env envconfig.Env) *Gitlab {
	config := env.String("REMOTE_CONFIG", "")

	url_, err := url.Parse(config)
	if err != nil {
		panic(err)
	}
	params := url_.Query()
	url_.RawQuery = ""

	gitlab := Gitlab{}
	gitlab.URL = url_.String()
	gitlab.Client = params.Get("client_id")
	gitlab.Secret = params.Get("client_secret")
	gitlab.AllowedOrgs = params["orgs"]
	gitlab.SkipVerify, _ = strconv.ParseBool(params.Get("skip_verify"))
	gitlab.Open, _ = strconv.ParseBool(params.Get("open"))

	switch params.Get("clone_mode") {
	case "oauth":
		gitlab.CloneMode = "oauth"
	default:
		gitlab.CloneMode = "token"
	}

	// this is a temp workaround
	gitlab.Search, _ = strconv.ParseBool(params.Get("search"))

	return &gitlab
}

// Login authenticates the session and returns the
// remote user details.
func (g *Gitlab) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {

	var config = &oauth2.Config{
		ClientId:     g.Client,
		ClientSecret: g.Secret,
		Scope:        DefaultScope,
		AuthURL:      fmt.Sprintf("%s/oauth/authorize", g.URL),
		TokenURL:     fmt.Sprintf("%s/oauth/token", g.URL),
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	trans_ := &http.Transport{
		Proxy:           http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: g.SkipVerify},
	}

	// get the OAuth code
	var code = req.FormValue("code")
	if len(code) == 0 {
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, false, nil
	}

	var trans = &oauth2.Transport{Config: config, Transport: trans_}
	var token_, err = trans.Exchange(code)
	if err != nil {
		return nil, false, fmt.Errorf("Error exchanging token. %s", err)
	}

	client := NewClient(g.URL, token_.AccessToken, g.SkipVerify)
	login, err := client.CurrentUser()
	if err != nil {
		return nil, false, err
	}
	user := &model.User{}
	user.Login = login.Username
	user.Email = login.Email
	user.Token = token_.AccessToken
	user.Secret = token_.RefreshToken

	if strings.HasPrefix(login.AvatarUrl, "http") {
		user.Avatar = login.AvatarUrl
	} else {
		user.Avatar = g.URL + "/" + login.AvatarUrl
	}

	return user, true, nil
}

func (g *Gitlab) Auth(token, secret string) (string, error) {
	client := NewClient(g.URL, token, g.SkipVerify)
	login, err := client.CurrentUser()
	if err != nil {
		return "", err
	}
	return login.Username, nil
}

// Repo fetches the named repository from the remote system.
func (g *Gitlab) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := NewClient(g.URL, u.Token, g.SkipVerify)
	id, err := GetProjectId(g, client, owner, name)
	if err != nil {
		return nil, err
	}
	repo_, err := client.Project(id)
	if err != nil {
		return nil, err
	}

	repo := &model.Repo{}
	repo.Owner = owner
	repo.Name = name
	repo.FullName = repo_.PathWithNamespace
	repo.Link = repo_.Url
	repo.Clone = repo_.HttpRepoUrl
	repo.Branch = "master"

	if repo_.DefaultBranch != "" {
		repo.Branch = repo_.DefaultBranch
	}

	if g.PrivateMode {
		repo.IsPrivate = true
	} else {
		repo.IsPrivate = !repo_.Public
	}

	return repo, err
}

// Repos fetches a list of repos from the remote system.
func (g *Gitlab) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := NewClient(g.URL, u.Token, g.SkipVerify)

	var repos = []*model.RepoLite{}

	all, err := client.AllProjects()
	if err != nil {
		return repos, err
	}

	for _, repo := range all {
		var parts = strings.Split(repo.PathWithNamespace, "/")
		var owner = parts[0]
		var name = parts[1]

		repos = append(repos, &model.RepoLite{
			Owner:    owner,
			Name:     name,
			FullName: repo.PathWithNamespace,
		})

		// TODO: add repo.AvatarUrl
	}
	return repos, err
}

// Perm fetches the named repository from the remote system.
func (g *Gitlab) Perm(u *model.User, owner, name string) (*model.Perm, error) {

	client := NewClient(g.URL, u.Token, g.SkipVerify)
	id, err := GetProjectId(g, client, owner, name)
	if err != nil {
		return nil, err
	}

	repo, err := client.Project(id)
	if err != nil {
		return nil, err
	}
	m := &model.Perm{}
	m.Admin = IsAdmin(repo)
	m.Pull = IsRead(repo)
	m.Push = IsWrite(repo)
	return m, nil
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (g *Gitlab) Script(user *model.User, repo *model.Repo, build *model.Build) ([]byte, []byte, error) {
	var client = NewClient(g.URL, user.Token, g.SkipVerify)
	id, err := GetProjectId(g, client, repo.Owner, repo.Name)
	if err != nil {
		return nil, nil, err
	}

	cfg, err := client.RepoRawFile(id, build.Commit, ".drone.yml")
	enc, _ := client.RepoRawFile(id, build.Commit, ".drone.sec")
	return cfg, enc, err
}

// NOTE Currently gitlab doesn't support status for commits and events,
//      also if we want get MR status in gitlab we need implement a special plugin for gitlab,
//      gitlab uses API to fetch build status on client side. But for now we skip this.
func (g *Gitlab) Status(u *model.User, repo *model.Repo, b *model.Build, link string) error {
	return nil
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *Gitlab) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	url_, err := url.Parse(g.URL)
	if err != nil {
		return nil, err
	}
	netrc := &model.Netrc{}
	netrc.Machine = url_.Host

	switch g.CloneMode {
	case "oauth":
		netrc.Login = "oauth2"
		netrc.Password = u.Token
	case "token":
		t := token.New(token.HookToken, r.FullName)
		netrc.Login = "drone-ci-token"
		netrc.Password, err = t.Sign(r.Hash)
	}
	return netrc, err
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (g *Gitlab) Activate(user *model.User, repo *model.Repo, k *model.Key, link string) error {
	var client = NewClient(g.URL, user.Token, g.SkipVerify)
	id, err := GetProjectId(g, client, repo.Owner, repo.Name)
	if err != nil {
		return err
	}

	uri, err := url.Parse(link)
	if err != nil {
		return err
	}

	droneUrl := fmt.Sprintf("%s://%s", uri.Scheme, uri.Host)
	droneToken := uri.Query().Get("access_token")
	ssl_verify := strconv.FormatBool(!g.SkipVerify)

	return client.AddDroneService(id, map[string]string{
		"token":                   droneToken,
		"drone_url":               droneUrl,
		"enable_ssl_verification": ssl_verify,
	})
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (g *Gitlab) Deactivate(user *model.User, repo *model.Repo, link string) error {
	var client = NewClient(g.URL, user.Token, g.SkipVerify)
	id, err := GetProjectId(g, client, repo.Owner, repo.Name)
	if err != nil {
		return err
	}

	return client.DeleteDroneService(id)
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Gitlab) Hook(req *http.Request) (*model.Repo, *model.Build, error) {
	defer req.Body.Close()
	var payload, _ = ioutil.ReadAll(req.Body)
	var parsed, err = gogitlab.ParseHook(payload)
	if err != nil {
		return nil, nil, err
	}

	switch parsed.ObjectKind {
	case "merge_request":
		return mergeRequest(parsed, req)
	case "tag_push", "push":
		return push(parsed, req)
	default:
		return nil, nil, nil
	}
}

func mergeRequest(parsed *gogitlab.HookPayload, req *http.Request) (*model.Repo, *model.Build, error) {

	repo := &model.Repo{}
	repo.Owner = req.FormValue("owner")
	repo.Name = req.FormValue("name")
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = parsed.ObjectAttributes.Target.WebUrl
	repo.Clone = parsed.ObjectAttributes.Target.HttpUrl
	repo.Branch = "master"

	build := &model.Build{}
	build.Event = "pull_request"
	build.Message = parsed.ObjectAttributes.LastCommit.Message
	build.Commit = parsed.ObjectAttributes.LastCommit.Id
	//build.Remote = parsed.ObjectAttributes.Source.HttpUrl

	if parsed.ObjectAttributes.SourceProjectId == parsed.ObjectAttributes.TargetProjectId {
		build.Ref = fmt.Sprintf("refs/heads/%s", parsed.ObjectAttributes.SourceBranch)
	} else {
		build.Ref = fmt.Sprintf("refs/merge-requests/%d/head", parsed.ObjectAttributes.IId)
	}

	build.Branch = parsed.ObjectAttributes.SourceBranch
	// build.Timestamp = parsed.ObjectAttributes.LastCommit.Timestamp

	build.Author = parsed.ObjectAttributes.LastCommit.Author.Name
	build.Email = parsed.ObjectAttributes.LastCommit.Author.Email
	build.Title = parsed.ObjectAttributes.Title
	build.Link = parsed.ObjectAttributes.Url

	return repo, build, nil
}

func push(parsed *gogitlab.HookPayload, req *http.Request) (*model.Repo, *model.Build, error) {
	var cloneUrl = parsed.Repository.GitHttpUrl

	repo := &model.Repo{}
	repo.Owner = req.FormValue("owner")
	repo.Name = req.FormValue("name")
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = parsed.Repository.URL
	repo.Clone = cloneUrl
	repo.Branch = "master"

	switch parsed.Repository.VisibilityLevel {
	case 0:
		repo.IsPrivate = true
	case 10:
		repo.IsPrivate = true
	case 20:
		repo.IsPrivate = false
	}

	repo.FullName = fmt.Sprintf("%s/%s", req.FormValue("owner"), req.FormValue("name"))

	build := &model.Build{}
	build.Event = model.EventPush
	build.Commit = parsed.After
	build.Branch = parsed.Branch()
	build.Ref = parsed.Ref
	// hook.Commit.Remote = cloneUrl

	var head = parsed.Head()
	build.Message = head.Message
	// build.Timestamp = head.Timestamp

	// extracts the commit author (ideally email)
	// from the post-commit hook
	switch {
	case head.Author != nil:
		build.Email = head.Author.Email
		build.Author = parsed.UserName
	case head.Author == nil:
		build.Author = parsed.UserName
	}

	return repo, build, nil
}

// ¯\_(ツ)_/¯
func (g *Gitlab) Oauth2Transport(r *http.Request) *oauth2.Transport {
	return &oauth2.Transport{
		Config: &oauth2.Config{
			ClientId:     g.Client,
			ClientSecret: g.Secret,
			Scope:        DefaultScope,
			AuthURL:      fmt.Sprintf("%s/oauth/authorize", g.URL),
			TokenURL:     fmt.Sprintf("%s/oauth/token", g.URL),
			RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(r)),
			//settings.Server.Scheme, settings.Server.Hostname),
		},
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: g.SkipVerify},
		},
	}
}

// Accessor method, to allowed remote organizations field.
func (g *Gitlab) GetOrgs() []string {
	return g.AllowedOrgs
}

// Accessor method, to open field.
func (g *Gitlab) GetOpen() bool {
	return g.Open
}

// return default scope for GitHub
func (g *Gitlab) Scope() string {
	return DefaultScope
}

func (g *Gitlab) String() string {
	return "gitlab"
}
