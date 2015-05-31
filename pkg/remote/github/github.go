package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"github.com/drone/drone/pkg/config"
	common "github.com/drone/drone/pkg/types"

	"github.com/drone/drone/Godeps/_workspace/src/github.com/google/go-github/github"
)

const (
	DefaultAPI   = "https://api.github.com/"
	DefaultURL   = "https://github.com"
	DefaultScope = "repo,repo:status,user:email"
)

type GitHub struct {
	URL         string
	API         string
	Client      string
	Secret      string
	PrivateMode bool
	SkipVerify  bool

	cache *lru.Cache
}

func New(conf *config.Config) *GitHub {
	var github = GitHub{
		API:         DefaultAPI,
		URL:         DefaultURL,
		Client:      conf.Auth.Client,
		Secret:      conf.Auth.Secret,
		PrivateMode: conf.Remote.Private,
		SkipVerify:  conf.Remote.SkipVerify,
	}
	var err error
	github.cache, err = lru.New(1028)
	if err != nil {
		panic(err)
	}

	// if GitHub enterprise then ensure we're using the
	// appropriate URLs
	if !strings.HasPrefix(conf.Remote.Base, DefaultURL) && len(conf.Remote.Base) != 0 {
		github.URL = conf.Remote.Base
		github.API = conf.Remote.Base + "/api/v3/"
	}
	// the API must have a trailing slash
	if !strings.HasSuffix(github.API, "/") {
		github.API += "/"
	}
	// the URL must NOT have a trailing slash
	if strings.HasSuffix(github.URL, "/") {
		github.URL = github.URL[:len(github.URL)-1]
	}
	return &github
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
	user.Name = *login.Name
	user.Token = token
	user.Secret = secret
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

	if repo_.Language != nil {
		repo.Language = *repo_.Language
	}
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
func (g *GitHub) Script(u *common.User, r *common.Repo, c *common.Commit) ([]byte, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	var sha string
	if len(c.SourceSha) == 0 {
		sha = c.Sha
	} else {
		sha = c.SourceSha
	}
	return GetFile(client, r.Owner, r.Name, ".drone.yml", sha)
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *GitHub) Netrc(u *common.User) (*common.Netrc, error) {
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

func (g *GitHub) Status(u *common.User, r *common.Repo, c *common.Commit) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	if len(c.PullRequest) == 0 {
		return nil
	}
	link := fmt.Sprintf("%s/%v", r.Self, c.Sequence)
	status := getStatus(c.State)
	desc := getDesc(c.State)
	data := github.RepoStatus{
		Context:     github.String("Drone"),
		State:       github.String(status),
		Description: github.String(desc),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateStatus(r.Owner, r.Name, c.SourceSha, &data)
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

// push parses a hook with event type `push` and returns
// the commit data.
func (g *GitHub) push(r *http.Request) (*common.Hook, error) {
	payload := GetPayload(r)
	hook := &pushHook{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, err
	}

	repo := &common.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	repo.FullName = hook.Repo.FullName
	repo.Link = hook.Repo.HTMLURL
	repo.Private = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Language = hook.Repo.Language
	repo.Branch = hook.Repo.DefaultBranch

	commit := &common.Commit{}
	commit.Sha = hook.Head.ID
	commit.Ref = hook.Ref
	commit.Branch = strings.Replace(commit.Ref, "refs/heads/", "", -1)
	commit.Message = hook.Head.Message
	commit.Timestamp = hook.Head.Timestamp
	commit.Author = hook.Head.Author.Username
	// commit.Author.Name = hook.Head.Author.Name
	// commit.Author.Email = hook.Head.Author.Email
	// commit.Author.Login = hook.Head.Author.Username

	// we should ignore github pages
	if commit.Ref == "refs/heads/gh-pages" {
		return nil, nil
	}

	return &common.Hook{Repo: repo, Commit: commit}, nil
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

	repo := &common.Repo{}
	repo.Owner = *hook.Repo.Owner.Login
	repo.Name = *hook.Repo.Name
	repo.FullName = *hook.Repo.FullName
	repo.Link = *hook.Repo.HTMLURL
	repo.Private = *hook.Repo.Private
	repo.Clone = *hook.Repo.CloneURL
	if hook.Repo.Language != nil {
		repo.Language = *hook.Repo.Language
	}
	if hook.Repo.DefaultBranch != nil {
		repo.Branch = *hook.Repo.DefaultBranch
	}

	c := &common.Commit{}
	c.PullRequest = strconv.Itoa(*hook.PullRequest.Number)
	c.Message = *hook.PullRequest.Title
	c.Sha = *hook.PullRequest.Base.SHA
	c.Ref = *hook.PullRequest.Base.Ref
	c.Ref = fmt.Sprintf("refs/pull/%s/merge", c.PullRequest)
	c.Branch = *hook.PullRequest.Base.Ref
	c.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")
	c.Author = *hook.PullRequest.Base.User.Login
	c.SourceRemote = *hook.PullRequest.Head.Repo.CloneURL
	c.SourceBranch = *hook.PullRequest.Head.Ref
	c.SourceSha = *hook.PullRequest.Head.SHA

	return &common.Hook{Repo: repo, Commit: c}, nil
}

type pushHook struct {
	Ref string `json:"ref"`

	Head struct {
		ID        string `json:"id"`
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
