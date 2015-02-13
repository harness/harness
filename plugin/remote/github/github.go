package github

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/plugin/remote/github/oauth"
	"github.com/drone/drone/shared/httputil"
	"github.com/drone/drone/shared/model"
	"github.com/drone/go-github/github"
)

const (
	DefaultAPI   = "https://api.github.com/"
	DefaultURL   = "https://github.com"
	DefaultScope = "repo,repo:status,user:email"
)

type GitHub struct {
	URL        string
	API        string
	Client     string
	Secret     string
	Private    bool
	SkipVerify bool
	Orgs       []string
	Open       bool
}

func New(url, api, client, secret string, private, skipVerify bool, orgs []string, open bool) *GitHub {
	var github = GitHub{
		URL:        url,
		API:        api,
		Client:     client,
		Secret:     secret,
		Private:    private,
		SkipVerify: skipVerify,
		Orgs:       orgs,
		Open:       open,
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

func NewDefault(client, secret string, orgs []string, open bool) *GitHub {
	return New(DefaultURL, DefaultAPI, client, secret, false, false, orgs, open)
}

// Authorize handles GitHub API Authorization.
func (r *GitHub) Authorize(res http.ResponseWriter, req *http.Request) (*model.Login, error) {
	var config = &oauth.Config{
		ClientId:     r.Client,
		ClientSecret: r.Secret,
		Scope:        DefaultScope,
		AuthURL:      fmt.Sprintf("%s/login/oauth/authorize", r.URL),
		TokenURL:     fmt.Sprintf("%s/login/oauth/access_token", r.URL),
		RedirectURL:  fmt.Sprintf("%s/api/auth/%s", httputil.GetURL(req), r.GetKind()),
	}

	// get the OAuth code
	var code = req.FormValue("code")
	var state = req.FormValue("state")
	if len(code) == 0 {
		var random = GetRandom()
		httputil.SetCookie(res, req, "github_state", random)
		http.Redirect(res, req, config.AuthCodeURL(random), http.StatusSeeOther)
		return nil, nil
	}

	cookieState := httputil.GetCookie(req, "github_state")
	httputil.DelCookie(res, req, "github_state")
	if cookieState != state {
		return nil, fmt.Errorf("Error matching state in OAuth2 redirect")
	}

	var trans = &oauth.Transport{Config: config}
	var token, err = trans.Exchange(code)
	if err != nil {
		return nil, fmt.Errorf("Error exchanging token. %s", err)
	}

	var client = NewClient(r.API, token.AccessToken, r.SkipVerify)
	var useremail, errr = GetUserEmail(client)
	if errr != nil {
		return nil, fmt.Errorf("Error retrieving user or verified email. %s", errr)
	}

	if len(r.Orgs) > 0 {
		allowedOrg, err := UserBelongsToOrg(client, r.Orgs)
		if err != nil {
			return nil, fmt.Errorf("Could not check org membership. %s", err)
		}
		if !allowedOrg {
			return nil, fmt.Errorf("User does not belong to correct org. Must belong to %v", r.Orgs)
		}
	}

	var login = new(model.Login)
	login.ID = int64(*useremail.ID)
	login.Access = token.AccessToken
	login.Login = *useremail.Login
	login.Email = *useremail.Email
	if useremail.Name != nil {
		login.Name = *useremail.Name
	}

	return login, nil
}

// GetKind returns the internal identifier of this remote GitHub instane.
func (r *GitHub) GetKind() string {
	if r.IsEnterprise() {
		return model.RemoteGithubEnterprise
	} else {
		return model.RemoteGithub
	}
}

// GetHost returns the hostname of this remote GitHub instance.
func (r *GitHub) GetHost() string {
	uri, _ := url.Parse(r.URL)
	return uri.Host
}

// IsEnterprise returns true if the remote system is an
// instance of GitHub Enterprise Edition.
func (r *GitHub) IsEnterprise() bool {
	return r.URL != DefaultURL
}

// GetRepos fetches all repositories that the specified
// user has access to in the remote system.
func (r *GitHub) GetRepos(user *model.User) ([]*model.Repo, error) {
	var repos []*model.Repo
	var client = NewClient(r.API, user.Access, r.SkipVerify)
	var list, err = GetAllRepos(client)
	if err != nil {
		return nil, err
	}

	var remote = r.GetKind()
	var hostname = r.GetHost()

	for _, item := range list {
		var repo = model.Repo{
			UserID:   user.ID,
			Remote:   remote,
			Host:     hostname,
			Owner:    *item.Owner.Login,
			Name:     *item.Name,
			Private:  *item.Private,
			URL:      *item.HTMLURL,
			CloneURL: *item.GitURL,
			GitURL:   *item.GitURL,
			SSHURL:   *item.SSHURL,
			Role:     &model.Perm{},
		}

		if r.Private || repo.Private {
			repo.CloneURL = *item.SSHURL
			repo.Private = true
		}

		// if no permissions we should skip the repository
		// entirely, since this should never happen
		if item.Permissions == nil {
			continue
		}

		repo.Role.Admin = (*item.Permissions)["admin"]
		repo.Role.Write = (*item.Permissions)["push"]
		repo.Role.Read = (*item.Permissions)["pull"]
		repos = append(repos, &repo)
	}

	return repos, err
}

// GetScript fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (r *GitHub) GetScript(user *model.User, repo *model.Repo, hook *model.Hook) ([]byte, error) {
	var client = NewClient(r.API, user.Access, r.SkipVerify)
	return GetFile(client, repo.Owner, repo.Name, ".drone.yml", hook.Sha)
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (r *GitHub) Deactivate(user *model.User, repo *model.Repo, link string) error {
	var client = NewClient(r.API, user.Access, r.SkipVerify)
	var title, err = GetKeyTitle(link)
	if err != nil {
		return err
	}

	// remove the deploy-key if it is installed remote.
	if err := DeleteKey(client, repo.Owner, repo.Name, title, repo.PublicKey); err != nil {
		return err
	}

	return DeleteHook(client, repo.Owner, repo.Name, link)
}

// Activate activates a repository by adding a Post-commit hook and
// a Public Deploy key, if applicable.
func (r *GitHub) Activate(user *model.User, repo *model.Repo, link string) error {
	var client = NewClient(r.API, user.Access, r.SkipVerify)
	var title, err = GetKeyTitle(link)
	if err != nil {
		return err
	}

	// if the CloneURL is using the SSHURL then we know that
	// we need to add an SSH key to GitHub.
	if repo.SSHURL == repo.CloneURL {
		_, err = CreateUpdateKey(client, repo.Owner, repo.Name, title, repo.PublicKey)
		if err != nil {
			return err
		}
	}

	_, err = CreateUpdateHook(client, repo.Owner, repo.Name, link)
	return err
}

// ParseHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (r *GitHub) ParseHook(req *http.Request) (*model.Hook, error) {
	// handle github ping
	if req.Header.Get("X-Github-Event") == "ping" {
		return nil, nil
	}

	// handle github pull request hook differently
	if req.Header.Get("X-Github-Event") == "pull_request" {
		return r.ParsePullRequestHook(req)
	}

	// parse the github Hook payload
	var payload = GetPayload(req)
	var data, err = github.ParseHook(payload)
	if err != nil {
		return nil, nil
	}

	// make sure this is being triggered because of a commit
	// and not something like a tag deletion or whatever
	if data.IsTag() ||
		data.IsGithubPages() ||
		data.IsHead() == false ||
		data.IsDeleted() {
		return nil, nil
	}

	var hook = new(model.Hook)
	hook.Repo = data.Repo.Name
	hook.Owner = data.Repo.Owner.Login
	hook.Sha = data.Head.Id
	hook.Branch = data.Branch()

	if len(hook.Owner) == 0 {
		hook.Owner = data.Repo.Owner.Name
	}

	// extract the author and message from the commit
	// this is kind of experimental, since I don't know
	// what I'm doing here.
	if data.Head != nil && data.Head.Author != nil {
		hook.Message = data.Head.Message
		hook.Timestamp = data.Head.Timestamp
		hook.Author = data.Head.Author.Email
	} else if data.Commits != nil && len(data.Commits) > 0 && data.Commits[0].Author != nil {
		hook.Message = data.Commits[0].Message
		hook.Timestamp = data.Commits[0].Timestamp
		hook.Author = data.Commits[0].Author.Email
	}

	return hook, nil
}

// ParsePullRequestHook parses the pull request hook from the Request body
// and returns the required data in a standard format.
func (r *GitHub) ParsePullRequestHook(req *http.Request) (*model.Hook, error) {

	// parse the payload to retrieve the pull-request
	// hook meta-data.
	var payload = GetPayload(req)
	var data, err = github.ParsePullRequestHook(payload)
	if err != nil {
		return nil, err
	}

	// ignore these
	if data.Action != "opened" && data.Action != "synchronize" {
		return nil, nil
	}

	// TODO we should also store the pull request branch (ie from x to y)
	//      we can find it here: data.PullRequest.Head.Ref
	var hook = model.Hook{
		Owner:       data.Repo.Owner.Login,
		Repo:        data.Repo.Name,
		Sha:         data.PullRequest.Head.Sha,
		Branch:      data.PullRequest.Head.Ref,
		Author:      data.PullRequest.User.Login,
		Gravatar:    data.PullRequest.User.GravatarId,
		Timestamp:   time.Now().UTC().String(),
		Message:     data.PullRequest.Title,
		PullRequest: strconv.Itoa(data.Number),
	}

	if len(hook.Owner) == 0 {
		hook.Owner = data.Repo.Owner.Name
	}

	return &hook, nil
}

func (r *GitHub) OpenRegistration() bool {
	return r.Open
}

func (r *GitHub) GetToken(user *model.User) (*model.Token, error) {
	return nil, nil
}
