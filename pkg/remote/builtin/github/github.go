package github

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"github.com/drone/drone/pkg/oauth2"
	"github.com/drone/drone/pkg/remote"
	common "github.com/drone/drone/pkg/types"
	"github.com/drone/drone/pkg/utils/httputil"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/google/go-github/github"
)

const (
	DefaultURL   = "https://github.com"
	DefaultAPI   = "https://api.github.com"
	DefaultScope = "repo,repo:status,user:email"
)

type GitHub struct {
	URL         string
	API         string
	Client      string
	Secret      string
	AllowedOrgs []string
	Open        bool
	PrivateMode bool
	SkipVerify  bool

	cache *lru.Cache
}

func init() {
	remote.Register("github", NewDriver)
}

func NewDriver(config string) (remote.Remote, error) {
	url_, err := url.Parse(config)
	if err != nil {
		return nil, err
	}
	params := url_.Query()
	url_.Path = ""
	url_.RawQuery = ""

	github := GitHub{}
	github.URL = url_.String()
	github.Client = params.Get("client_id")
	github.Secret = params.Get("client_secret")
	github.AllowedOrgs = params["orgs"]
	github.PrivateMode, _ = strconv.ParseBool(params.Get("private_mode"))
	github.SkipVerify, _ = strconv.ParseBool(params.Get("skip_verify"))
	github.Open, _ = strconv.ParseBool(params.Get("open"))

	if github.URL == DefaultURL {
		github.API = DefaultAPI
	} else {
		github.API = github.URL + "/api/v3/"
	}

	// here we cache permissions to avoid too many api
	// calls. this should really be moved outise the
	// remote plugin into the app
	github.cache, err = lru.New(1028)
	return &github, err
}

func (g *GitHub) Login(token, secret string) (*common.User, error) {
	client := NewClient(g.API, token, g.SkipVerify)
	login, err := GetUserEmail(client)
	if err != nil {
		return nil, err
	}
	user := common.User{}
	user.Login = *login.Login
	user.Email = *login.Email
	user.Token = token
	user.Secret = secret
	user.Avatar = *login.AvatarURL
	return &user, nil
}

// Orgs fetches the organizations for the given user.
func (g *GitHub) Orgs(u *common.User) ([]string, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	orgs_ := []string{}
	orgs, err := GetOrgs(client)
	if err != nil {
		return orgs_, err
	}
	for _, org := range orgs {
		orgs_ = append(orgs_, *org.Login)
	}
	return orgs_, nil
}

// Accessor method, to allowed remote organizations field.
func (g *GitHub) GetOrgs() []string {
	return g.AllowedOrgs
}

// Accessor method, to open field.
func (g *GitHub) GetOpen() bool {
	return g.Open
}

// Repo fetches the named repository from the remote system.
func (g *GitHub) Repo(u *common.User, owner, name string) (*common.Repo, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	repo_, err := GetRepo(client, owner, name)
	if err != nil {
		return nil, err
	}

	repo := &common.Repo{}
	repo.Owner = owner
	repo.Name = name
	repo.FullName = *repo_.FullName
	repo.Link = *repo_.HTMLURL
	repo.Private = *repo_.Private
	repo.Clone = *repo_.CloneURL
	repo.Branch = "master"
	repo.Avatar = *repo_.Owner.AvatarURL

	if repo_.DefaultBranch != nil {
		repo.Branch = *repo_.DefaultBranch
	}

	if g.PrivateMode {
		repo.Private = true
	}
	return repo, err
}

// Perm fetches the named repository from the remote system.
func (g *GitHub) Perm(u *common.User, owner, name string) (*common.Perm, error) {
	key := fmt.Sprintf("%s/%s/%s", u.Login, owner, name)
	val, ok := g.cache.Get(key)
	if ok {
		return val.(*common.Perm), nil
	}

	client := NewClient(g.API, u.Token, g.SkipVerify)
	repo, err := GetRepo(client, owner, name)
	if err != nil {
		return nil, err
	}
	m := &common.Perm{}
	m.Admin = (*repo.Permissions)["admin"]
	m.Push = (*repo.Permissions)["push"]
	m.Pull = (*repo.Permissions)["pull"]
	g.cache.Add(key, m)
	return m, nil
}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (g *GitHub) Script(u *common.User, r *common.Repo, b *common.Build) ([]byte, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	return GetFile(client, r.Owner, r.Name, ".drone.yml", b.Commit.Sha)
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *GitHub) Netrc(u *common.User, r *common.Repo) (*common.Netrc, error) {
	url_, err := url.Parse(g.URL)
	if err != nil {
		return nil, err
	}
	netrc := &common.Netrc{}
	netrc.Login = u.Token
	netrc.Password = "x-oauth-basic"
	netrc.Machine = url_.Host
	return netrc, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (g *GitHub) Activate(u *common.User, r *common.Repo, k *common.Keypair, link string) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	title, err := GetKeyTitle(link)
	if err != nil {
		return err
	}

	// if the CloneURL is using the SSHURL then we know that
	// we need to add an SSH key to GitHub.
	if r.Private || g.PrivateMode {
		_, err = CreateUpdateKey(client, r.Owner, r.Name, title, k.Public)
		if err != nil {
			return err
		}
	}

	_, err = CreateUpdateHook(client, r.Owner, r.Name, link)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (g *GitHub) Deactivate(u *common.User, r *common.Repo, link string) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	title, err := GetKeyTitle(link)
	if err != nil {
		return err
	}

	// remove the deploy-key if it is installed remote.
	if r.Private || g.PrivateMode {
		if err := DeleteKey(client, r.Owner, r.Name, title); err != nil {
			return err
		}
	}

	return DeleteHook(client, r.Owner, r.Name, link)
}

func (g *GitHub) Status(u *common.User, r *common.Repo, b *common.Build) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)

	link := fmt.Sprintf("%s/%v", r.Self, b.Number)
	status := getStatus(b.Status)
	desc := getDesc(b.Status)
	data := github.RepoStatus{
		Context:     github.String("Drone"),
		State:       github.String(status),
		Description: github.String(desc),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateStatus(r.Owner, r.Name, b.Commit.Sha, &data)
	return err
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *GitHub) Hook(r *http.Request) (*common.Hook, error) {
	switch r.Header.Get("X-Github-Event") {
	case "pull_request":
		return g.pullRequest(r)
	case "push":
		return g.push(r)
	default:
		return nil, nil
	}
}

// return default scope for GitHub
func (g *GitHub) Scope() string {
	return DefaultScope
}

// push parses a hook with event type `push` and returns
// the commit data.
func (g *GitHub) push(r *http.Request) (*common.Hook, error) {
	payload := GetPayload(r)
	hook := &pushHook{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, err
	}

	if hook.Deleted {
		return nil, nil
	}

	repo := &common.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	// Generating rather than using hook.Repo.FullName as it's
	// not always present
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = hook.Repo.HTMLURL
	repo.Private = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Branch = hook.Repo.DefaultBranch

	commit := &common.Commit{}
	commit.Sha = hook.Head.ID
	commit.Ref = hook.Ref
	commit.Link = hook.Head.URL
	commit.Branch = strings.Replace(commit.Ref, "refs/heads/", "", -1)
	commit.Message = hook.Head.Message
	commit.Timestamp = hook.Head.Timestamp
	commit.Author = &common.Author{}
	commit.Author.Email = hook.Head.Author.Email
	commit.Author.Login = hook.Head.Author.Username
	commit.Remote = hook.Repo.CloneURL

	// we should ignore github pages
	if commit.Ref == "refs/heads/gh-pages" {
		return nil, nil
	}

	return &common.Hook{Repo: repo, Commit: commit}, nil
}

// ¯\_(ツ)_/¯
func (g *GitHub) Oauth2Transport(r *http.Request) *oauth2.Transport {
	return &oauth2.Transport{
		Config: &oauth2.Config{
			ClientId:     g.Client,
			ClientSecret: g.Secret,
			Scope:        DefaultScope,
			AuthURL:      fmt.Sprintf("%s/login/oauth/authorize", g.URL),
			TokenURL:     fmt.Sprintf("%s/login/oauth/access_token", g.URL),
			RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(r)),
			//settings.Server.Scheme, settings.Server.Hostname),
		},
		Transport: &http.Transport{
			Proxy:           http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: g.SkipVerify},
		},
	}
}

// pullRequest parses a hook with event type `pullRequest`
// and returns the commit data.
func (g *GitHub) pullRequest(r *http.Request) (*common.Hook, error) {
	payload := GetPayload(r)
	hook := &struct {
		Action      string              `json:"action"`
		PullRequest *github.PullRequest `json:"pull_request"`
		Repo        *github.Repository  `json:"repository"`
	}{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, err
	}

	// ignore these
	if hook.Action != "opened" && hook.Action != "synchronize" {
		return nil, nil
	}
	if *hook.PullRequest.State != "open" {
		return nil, nil
	}

	repo := &common.Repo{}
	repo.Owner = *hook.Repo.Owner.Login
	repo.Name = *hook.Repo.Name
	repo.FullName = *hook.Repo.FullName
	repo.Link = *hook.Repo.HTMLURL
	repo.Private = *hook.Repo.Private
	repo.Clone = *hook.Repo.CloneURL
	repo.Branch = "master"
	if hook.Repo.DefaultBranch != nil {
		repo.Branch = *hook.Repo.DefaultBranch
	}

	c := &common.Commit{}
	c.Sha = *hook.PullRequest.Head.SHA
	c.Ref = *hook.PullRequest.Head.Ref
	c.Ref = fmt.Sprintf("refs/pull/%d/merge", *hook.PullRequest.Number)
	c.Branch = *hook.PullRequest.Head.Ref
	c.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")
	c.Remote = *hook.PullRequest.Head.Repo.CloneURL
	c.Author = &common.Author{}
	c.Author.Login = *hook.PullRequest.Head.User.Login

	// Author.Email
	// Message

	pr := &common.PullRequest{}
	pr.Number = *hook.PullRequest.Number
	pr.Title = *hook.PullRequest.Title
	pr.Base = &common.Commit{}
	pr.Base.Sha = *hook.PullRequest.Base.SHA
	pr.Base.Ref = *hook.PullRequest.Base.Ref
	pr.Base.Remote = *hook.PullRequest.Base.Repo.CloneURL
	pr.Link = *hook.PullRequest.HTMLURL
	// Branch
	// Message
	// Timestamp
	// Author.Login
	// Author.Email

	return &common.Hook{Repo: repo, Commit: c, PullRequest: pr}, nil
}

type pushHook struct {
	Ref     string `json:"ref"`
	Deleted bool   `json:"deleted"`

	Head struct {
		ID        string `json:"id"`
		URL       string `json:"url"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`

		Author struct {
			Name     string `json:"name"`
			Email    string `json:"name"`
			Username string `json:"username"`
		} `json:"author"`

		Committer struct {
			Name     string `json:"name"`
			Email    string `json:"name"`
			Username string `json:"username"`
		} `json:"committer"`
	} `json:"head_commit"`

	Repo struct {
		Owner struct {
			Login string `json:"login"`
			Name  string `json:"name"`
		} `json:"owner"`
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		Language      string `json:"language"`
		Private       bool   `json:"private"`
		HTMLURL       string `json:"html_url"`
		CloneURL      string `json:"clone_url"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
}

const (
	StatusPending = "pending"
	StatusSuccess = "success"
	StatusFailure = "failure"
	StatusError   = "error"
)

const (
	DescPending = "this build is pending"
	DescSuccess = "the build was successful"
	DescFailure = "the build failed"
	DescError   = "oops, something went wrong"
)

// getStatus is a helper functin that converts a Drone
// status to a GitHub status.
func getStatus(status string) string {
	switch status {
	case common.StatePending, common.StateRunning:
		return StatusPending
	case common.StateSuccess:
		return StatusSuccess
	case common.StateFailure:
		return StatusFailure
	case common.StateError, common.StateKilled:
		return StatusError
	default:
		return StatusError
	}
}

// getDesc is a helper function that generates a description
// message for the build based on the status.
func getDesc(status string) string {
	switch status {
	case common.StatePending, common.StateRunning:
		return DescPending
	case common.StateSuccess:
		return DescSuccess
	case common.StateFailure:
		return DescFailure
	case common.StateError, common.StateKilled:
		return DescError
	default:
		return DescError
	}
}
