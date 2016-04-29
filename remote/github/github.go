package github

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/oauth2"

	"github.com/google/go-github/github"
)

const (
	DefaultURL = "https://github.com"     // Default GitHub URL
	DefaultAPI = "https://api.github.com" // Default GitHub API URL
)

type Github struct {
	URL         string
	API         string
	Client      string
	Secret      string
	Scope       string
	MergeRef    string
	PrivateMode bool
	SkipVerify  bool
}

func New(url, client, secret string, scope []string, private, skipverify, mergeref bool) (remote.Remote, error) {
	remote := &Github{
		URL:         strings.TrimSuffix(url, "/"),
		Client:      client,
		Secret:      secret,
		Scope:       strings.Join(scope, ","),
		PrivateMode: private,
		SkipVerify:  skipverify,
		MergeRef:    "head",
	}

	if remote.URL == DefaultURL {
		remote.API = DefaultAPI
	} else {
		remote.API = remote.URL + "/api/v3/"
	}
	if mergeref {
		remote.MergeRef = "merge"
	}

	return remote, nil
}

// Login authenticates the session and returns the remote user details.
func (g *Github) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {

	var config = &oauth2.Config{
		ClientId:     g.Client,
		ClientSecret: g.Secret,
		Scope:        g.Scope,
		AuthURL:      fmt.Sprintf("%s/login/oauth/authorize", g.URL),
		TokenURL:     fmt.Sprintf("%s/login/oauth/access_token", g.URL),
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	// get the OAuth code
	var code = req.FormValue("code")
	if len(code) == 0 {
		var random = GetRandom()
		http.Redirect(res, req, config.AuthCodeURL(random), http.StatusSeeOther)
		return nil, nil
	}

	var trans = &oauth2.Transport{
		Config: config,
	}
	if g.SkipVerify {
		trans.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}
	var token, err = trans.Exchange(code)
	if err != nil {
		return nil, fmt.Errorf("Error exchanging token. %s", err)
	}

	var client = NewClient(g.API, token.AccessToken, g.SkipVerify)
	var useremail, errr = GetUserEmail(client)
	if errr != nil {
		return nil, fmt.Errorf("Error retrieving user or verified email. %s", errr)
	}

	user := model.User{}
	user.Login = *useremail.Login
	user.Email = *useremail.Email
	user.Token = token.AccessToken
	user.Avatar = *useremail.AvatarURL
	return &user, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (g *Github) Auth(token, secret string) (string, error) {
	client := NewClient(g.API, token, g.SkipVerify)
	user, _, err := client.Users.Get("")
	if err != nil {
		return "", err
	}
	return *user.Login, nil
}

func (g *Github) Teams(u *model.User) ([]*model.Team, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	orgs, err := GetOrgs(client)
	if err != nil {
		return nil, err
	}

	var teams []*model.Team
	for _, org := range orgs {
		teams = append(teams, &model.Team{
			Login:  *org.Login,
			Avatar: *org.AvatarURL,
		})
	}
	return teams, nil
}

// Repo fetches the named repository from the remote system.
func (g *Github) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	r, err := GetRepo(client, owner, name)
	if err != nil {
		return nil, err
	}

	repo := &model.Repo{
		Owner:     owner,
		Name:      name,
		FullName:  *r.FullName,
		Link:      *r.HTMLURL,
		IsPrivate: *r.Private,
		Clone:     *r.CloneURL,
		Avatar:    *r.Owner.AvatarURL,
		Kind:      model.RepoGit,
	}

	if r.DefaultBranch != nil {
		repo.Branch = *r.DefaultBranch
	} else {
		repo.Branch = "master"
	}

	if g.PrivateMode {
		repo.IsPrivate = true
	}

	return repo, err
}

// Repos fetches a list of repos from the remote system.
func (g *Github) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)

	all, err := GetUserRepos(client)
	if err != nil {
		return nil, err
	}

	var repos = []*model.RepoLite{}
	for _, repo := range all {
		repos = append(repos, &model.RepoLite{
			Owner:    *repo.Owner.Login,
			Name:     *repo.Name,
			FullName: *repo.FullName,
			Avatar:   *repo.Owner.AvatarURL,
		})
	}
	return repos, err
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (g *Github) Perm(u *model.User, owner, name string) (*model.Perm, error) {

	client := NewClient(g.API, u.Token, g.SkipVerify)
	repo, err := GetRepo(client, owner, name)
	if err != nil {
		return nil, err
	}
	m := &model.Perm{}
	m.Admin = (*repo.Permissions)["admin"]
	m.Push = (*repo.Permissions)["push"]
	m.Pull = (*repo.Permissions)["pull"]
	return m, nil
}

// File fetches a file from the remote repository and returns in string format.
func (g *Github) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	cfg, err := GetFile(client, r.Owner, r.Name, f, b.Commit)
	return cfg, err
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (g *Github) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	if b.Event == "deployment" {
		return deploymentStatus(client, r, b, link)
	} else {
		return repoStatus(client, r, b, link)
	}
}

func repoStatus(client *github.Client, r *model.Repo, b *model.Build, link string) error {
	status := getStatus(b.Status)
	desc := getDesc(b.Status)
	data := github.RepoStatus{
		Context:     github.String("continuous-integration/drone"),
		State:       github.String(status),
		Description: github.String(desc),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateStatus(r.Owner, r.Name, b.Commit, &data)
	return err
}

var reDeploy = regexp.MustCompile(".+/deployments/(\\d+)")

func deploymentStatus(client *github.Client, r *model.Repo, b *model.Build, link string) error {
	matches := reDeploy.FindStringSubmatch(b.Link)
	// if the deployment was not triggered from github, don't send a deployment status
	if len(matches) != 2 {
		return nil
	}
	// the deployment ID is only available in the the link to the build as the last element in the URL
	id, _ := strconv.Atoi(matches[1])
	status := getStatus(b.Status)
	desc := getDesc(b.Status)
	data := github.DeploymentStatusRequest{
		State:       github.String(status),
		Description: github.String(desc),
		TargetURL:   github.String(link),
	}
	_, _, err := client.Repositories.CreateDeploymentStatus(r.Owner, r.Name, id, &data)
	return err
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (g *Github) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	url_, err := url.Parse(g.URL)
	if err != nil {
		return nil, err
	}
	netrc := &model.Netrc{}
	netrc.Login = u.Token
	netrc.Password = "x-oauth-basic"
	netrc.Machine = url_.Host
	return netrc, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (g *Github) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	_, err := CreateUpdateHook(client, r.Owner, r.Name, link)
	return err
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (g *Github) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := NewClient(g.API, u.Token, g.SkipVerify)
	return DeleteHook(client, r.Owner, r.Name, link)
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Github) Hook(r *http.Request) (*model.Repo, *model.Build, error) {

	switch r.Header.Get("X-Github-Event") {
	case "pull_request":
		return g.pullRequest(r)
	case "push":
		return g.push(r)
	case "deployment":
		return g.deployment(r)
	default:
		return nil, nil, nil
	}
}

// push parses a hook with event type `push` and returns
// the commit data.
func (g *Github) push(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &pushHook{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.Deleted {
		return nil, nil, err
	}

	repo := &model.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	// Generating rather than using hook.Repo.FullName as it's
	// not always present
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = hook.Repo.HTMLURL
	repo.IsPrivate = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Branch = hook.Repo.DefaultBranch
	repo.Kind = model.RepoGit

	build := &model.Build{}
	build.Event = model.EventPush
	build.Commit = hook.Head.ID
	build.Ref = hook.Ref
	build.Link = hook.Head.URL
	build.Branch = strings.Replace(build.Ref, "refs/heads/", "", -1)
	build.Message = hook.Head.Message
	// build.Timestamp = hook.Head.Timestamp
	build.Email = hook.Head.Author.Email
	build.Avatar = hook.Sender.Avatar
	build.Author = hook.Sender.Login
	build.Remote = hook.Repo.CloneURL

	if len(build.Author) == 0 {
		build.Author = hook.Head.Author.Username
		// default gravatar?
	}

	if strings.HasPrefix(build.Ref, "refs/tags/") {
		// just kidding, this is actually a tag event
		build.Event = model.EventTag
	}

	return repo, build, nil
}

// pullRequest parses a hook with event type `pullRequest`
// and returns the commit data.
func (g *Github) pullRequest(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &struct {
		Action      string              `json:"action"`
		PullRequest *github.PullRequest `json:"pull_request"`
		Repo        *github.Repository  `json:"repository"`
	}{}
	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}

	// ignore these
	if hook.Action != "opened" && hook.Action != "synchronize" {
		return nil, nil, nil
	}
	if *hook.PullRequest.State != "open" {
		return nil, nil, nil
	}

	repo := &model.Repo{}
	repo.Owner = *hook.Repo.Owner.Login
	repo.Name = *hook.Repo.Name
	repo.FullName = *hook.Repo.FullName
	repo.Link = *hook.Repo.HTMLURL
	repo.IsPrivate = *hook.Repo.Private
	repo.Clone = *hook.Repo.CloneURL
	repo.Kind = model.RepoGit
	repo.Branch = "master"
	if hook.Repo.DefaultBranch != nil {
		repo.Branch = *hook.Repo.DefaultBranch
	}

	build := &model.Build{}
	build.Event = model.EventPull
	build.Commit = *hook.PullRequest.Head.SHA
	build.Ref = fmt.Sprintf("refs/pull/%d/%s", *hook.PullRequest.Number, g.MergeRef)
	build.Link = *hook.PullRequest.HTMLURL
	build.Branch = *hook.PullRequest.Head.Ref
	build.Message = *hook.PullRequest.Title
	build.Author = *hook.PullRequest.User.Login
	build.Avatar = *hook.PullRequest.User.AvatarURL
	build.Remote = *hook.PullRequest.Base.Repo.CloneURL
	build.Title = *hook.PullRequest.Title
	// build.Timestamp = time.Now().UTC().Format("2006-01-02 15:04:05.000000000 +0000 MST")

	return repo, build, nil
}

func (g *Github) deployment(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := GetPayload(r)
	hook := &deployHook{}

	err := json.Unmarshal(payload, hook)
	if err != nil {
		return nil, nil, err
	}

	// for older versions of GitHub. Remove.
	if hook.Deployment.ID == 0 {
		hook.Deployment.ID = hook.ID
		hook.Deployment.Sha = hook.Sha
		hook.Deployment.Ref = hook.Ref
		hook.Deployment.Task = hook.Name
		hook.Deployment.Env = hook.Env
		hook.Deployment.Desc = hook.Desc
	}

	repo := &model.Repo{}
	repo.Owner = hook.Repo.Owner.Login
	if len(repo.Owner) == 0 {
		repo.Owner = hook.Repo.Owner.Name
	}
	repo.Name = hook.Repo.Name
	repo.FullName = fmt.Sprintf("%s/%s", repo.Owner, repo.Name)
	repo.Link = hook.Repo.HTMLURL
	repo.IsPrivate = hook.Repo.Private
	repo.Clone = hook.Repo.CloneURL
	repo.Branch = hook.Repo.DefaultBranch
	repo.Kind = model.RepoGit

	// ref can be
	// branch, tag, or sha

	build := &model.Build{}
	build.Event = model.EventDeploy
	build.Commit = hook.Deployment.Sha
	build.Link = hook.Deployment.Url
	build.Message = hook.Deployment.Desc
	build.Avatar = hook.Sender.Avatar
	build.Author = hook.Sender.Login
	build.Ref = hook.Deployment.Ref
	build.Branch = hook.Deployment.Ref
	build.Deploy = hook.Deployment.Env

	// if the ref is a sha or short sha we need to manually
	// construct the ref.
	if strings.HasPrefix(build.Commit, build.Ref) || build.Commit == build.Ref {
		build.Branch = repo.Branch
		build.Ref = fmt.Sprintf("refs/heads/%s", repo.Branch)

	}
	// if the ref is a branch we should make sure it has refs/heads prefix
	if !strings.HasPrefix(build.Ref, "refs/") { // branch or tag
		build.Ref = fmt.Sprintf("refs/heads/%s", build.Branch)

	}

	return repo, build, nil
}

func (g *Github) String() string {
	return "github"
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
	case model.StatusPending, model.StatusRunning:
		return StatusPending
	case model.StatusSuccess:
		return StatusSuccess
	case model.StatusFailure:
		return StatusFailure
	case model.StatusError, model.StatusKilled:
		return StatusError
	default:
		return StatusError
	}
}

// getDesc is a helper function that generates a description
// message for the build based on the status.
func getDesc(status string) string {
	switch status {
	case model.StatusPending, model.StatusRunning:
		return DescPending
	case model.StatusSuccess:
		return DescSuccess
	case model.StatusFailure:
		return DescFailure
	case model.StatusError, model.StatusKilled:
		return DescError
	default:
		return DescError
	}
}
