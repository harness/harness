package github

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/drone/drone/plugin/remote"
	"github.com/drone/drone/shared/util/httputil"
	"github.com/drone/go-github/github"
	"github.com/drone/go-github/oauth2"
)

var (
	scope = "repo,repo:status,user:email"
	state = "FqB4EbagQ2o"
)

type Github struct {
	URL     string `json:"url"` // https://github.com
	API     string `json:"api"` // https://api.github.com
	Client  string `json:"client"`
	Secret  string `json:"secret"`
	Enabled bool   `json:"enabled"`
}

// GetName returns the name of this remote system.
func (g *Github) GetName() string {
	switch g.URL {
	case "https://github.com":
		return "github.com"
	default:
		return "enterprise.github.com"
	}
}

// GetHost returns the url.Host of this remote system.
func (g *Github) GetHost() (host string) {
	u, err := url.Parse(g.URL)
	if err != nil {
		return
	}
	return u.Host
}

// GetHook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (g *Github) GetHook(r *http.Request) (*remote.Hook, error) {
	// handle github ping
	if r.Header.Get("X-Github-Event") == "ping" {
		return nil, nil
	}

	// handle github pull request hook differently
	if r.Header.Get("X-Github-Event") == "pull_request" {
		return g.GetPullRequestHook(r)
	}

	// get the payload of the message
	payload := r.FormValue("payload")

	// parse the github Hook payload
	data, err := github.ParseHook([]byte(payload))
	if err != nil {
		return nil, nil
	}

	// make sure this is being triggered because of a commit
	// and not something like a tag deletion or whatever
	if data.IsTag() || data.IsGithubPages() ||
		data.IsHead() == false || data.IsDeleted() {
		return nil, nil
	}

	hook := remote.Hook{}
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

	return &hook, nil
}

func (g *Github) GetPullRequestHook(r *http.Request) (*remote.Hook, error) {
	payload := r.FormValue("payload")

	// parse the payload to retrieve the pull-request
	// hook meta-data.
	data, err := github.ParsePullRequestHook([]byte(payload))
	if err != nil {
		return nil, err
	}

	// ignore these
	if data.Action != "opened" && data.Action != "synchronize" {
		return nil, nil
	}

	// TODO we should also store the pull request branch (ie from x to y)
	//      we can find it here: data.PullRequest.Head.Ref
	hook := remote.Hook{
		Owner:       data.Repo.Owner.Login,
		Repo:        data.Repo.Name,
		Sha:         data.PullRequest.Head.Sha,
		Branch:      data.PullRequest.Base.Ref,
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

// GetLogin handles authentication to third party, remote services
// and returns the required user data in a standard format.
func (g *Github) GetLogin(w http.ResponseWriter, r *http.Request) (*remote.Login, error) {
	// create the oauth2 client
	oauth := oauth2.Client{
		RedirectURL:      fmt.Sprintf("%s://%s/login/github.com", httputil.GetScheme(r), httputil.GetHost(r)),
		AccessTokenURL:   fmt.Sprintf("%s/login/oauth/access_token", g.URL),
		AuthorizationURL: fmt.Sprintf("%s/login/oauth/authorize", g.URL),
		ClientId:         g.Client,
		ClientSecret:     g.Secret,
	}

	// get the OAuth code
	code := r.FormValue("code")
	if len(code) == 0 {
		redirect := oauth.AuthorizeRedirect(scope, state)
		http.Redirect(w, r, redirect, http.StatusSeeOther)
		return nil, nil
	}

	// exchange code for an auth token
	token, err := oauth.GrantToken(code)
	if err != nil {
		return nil, fmt.Errorf("Error granting GitHub authorization token. %s", err)
	}

	// create the client
	client := github.New(token.AccessToken)

	// get the user information
	user, err := client.Users.Current()
	if err != nil {
		return nil, fmt.Errorf("Error retrieving currently authenticated GitHub user. %s", err)
	}

	// put the user data in the common format
	login := remote.Login{
		ID:     user.ID,
		Login:  user.Login,
		Access: token.AccessToken,
		Name:   user.Name,
	}

	// get the users primary email address
	email, err := client.Emails.FindPrimary()
	if err == nil {
		login.Email = email.Email
	}

	return &login, nil
}

// GetClient returns a new Github remote client.
func (g *Github) GetClient(access, secret string) remote.Client {
	return &Client{g, access}
}

// IsMatch returns true if the hostname matches the
// hostname of this remote client.
func (g *Github) IsMatch(hostname string) bool {
	return strings.HasSuffix(hostname, g.URL)
}
