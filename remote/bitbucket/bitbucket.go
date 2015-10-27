package bitbucket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/drone/drone/model"
	"github.com/drone/drone/shared/envconfig"
	"github.com/drone/drone/shared/httputil"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

type Bitbucket struct {
	Client string
	Secret string
	Orgs   []string
	Open   bool
}

func Load(env envconfig.Env) *Bitbucket {
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
	bitbucket := Bitbucket{}
	bitbucket.Client = params.Get("client_id")
	bitbucket.Secret = params.Get("client_secret")
	bitbucket.Orgs = params["orgs"]
	bitbucket.Open, _ = strconv.ParseBool(params.Get("open"))

	return &bitbucket
}

// Login authenticates the session and returns the
// remote user details.
func (bb *Bitbucket) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {

	config := &oauth2.Config{
		ClientID:     bb.Client,
		ClientSecret: bb.Secret,
		Endpoint:     bitbucket.Endpoint,
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	// get the OAuth code
	var code = req.FormValue("code")
	if len(code) == 0 {
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, false, nil
	}

	var token, err = config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, false, fmt.Errorf("Error exchanging token. %s", err)
	}

	client := NewClient(config.Client(oauth2.NoContext, token))
	curr, err := client.FindCurrent()
	if err != nil {
		return nil, false, err
	}

	// convers the current bitbucket user to the
	// common drone user structure.
	user := model.User{}
	user.Login = curr.Login
	user.Token = token.AccessToken
	user.Secret = token.RefreshToken
	user.Expiry = token.Expiry.UTC().Unix()
	user.Avatar = curr.Links.Avatar.Href

	// gets the primary, confirmed email from bitbucket
	emails, err := client.ListEmail()
	if err != nil {
		return nil, false, err
	}
	for _, email := range emails.Values {
		if email.IsPrimary && email.IsConfirmed {
			user.Email = email.Email
			break
		}
	}

	// if the installation is restricted to a subset
	// of organizations, get the orgs and verify the
	// user is a member.
	if len(bb.Orgs) != 0 {
		resp, err := client.ListTeams(&ListTeamOpts{Page: 1, PageLen: 100, Role: "member"})
		if err != nil {
			return nil, false, err
		}

		var member bool
		for _, team := range resp.Values {
			for _, team_ := range bb.Orgs {
				if team.Login == team_ {
					member = true
					break
				}
			}
		}

		if !member {
			return nil, false, fmt.Errorf("User does not belong to correct org. Must belong to %v", bb.Orgs)
		}
	}

	return &user, bb.Open, nil
}

// Auth authenticates the session and returns the remote user
// login for the given token and secret
func (bb *Bitbucket) Auth(token, secret string) (string, error) {
	token_ := oauth2.Token{AccessToken: token, RefreshToken: secret}
	client := NewClientToken(bb.Client, bb.Secret, &token_)

	user, err := client.FindCurrent()
	if err != nil {
		return "", err
	}
	return user.Login, nil
}

// Refresh refreshes an oauth token and expiration for the given
// user. It returns true if the token was refreshed, false if the
// token was not refreshed, and error if it failed to refersh.
func (bb *Bitbucket) Refresh(user *model.User) (bool, error) {
	config := &oauth2.Config{
		ClientID:     bb.Client,
		ClientSecret: bb.Secret,
		Endpoint:     bitbucket.Endpoint,
	}

	// creates a token source with just the refresh token.
	// this will ensure an access token is automatically
	// requested.
	source := config.TokenSource(
		oauth2.NoContext, &oauth2.Token{RefreshToken: user.Secret})

	// requesting the token automatically refreshes and
	// returns a new access token.
	token, err := source.Token()
	if err != nil || len(token.AccessToken) == 0 {
		return false, err
	}

	// update the user to include tne new access token
	user.Token = token.AccessToken
	user.Secret = token.RefreshToken
	user.Expiry = token.Expiry.UTC().Unix()
	return true, nil
}

// Repo fetches the named repository from the remote system.
func (bb *Bitbucket) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	token := oauth2.Token{AccessToken: u.Token, RefreshToken: u.Secret}
	client := NewClientToken(bb.Client, bb.Secret, &token)

	repo, err := client.FindRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return convertRepo(repo), nil
}

// Repos fetches a list of repos from the remote system.
func (bb *Bitbucket) Repos(u *model.User) ([]*model.RepoLite, error) {
	token := oauth2.Token{AccessToken: u.Token, RefreshToken: u.Secret}
	client := NewClientToken(bb.Client, bb.Secret, &token)
	var repos []*model.RepoLite

	// gets a list of all accounts to query, including the
	// user's account and all team accounts.
	logins := []string{u.Login}
	resp, err := client.ListTeams(&ListTeamOpts{PageLen: 100, Role: "member"})
	if err != nil {
		return repos, err
	}
	for _, team := range resp.Values {
		logins = append(logins, team.Login)
	}

	// for each account, get the list of repos
	for _, login := range logins {
		repos_, err := client.ListReposAll(login)
		if err != nil {
			return repos, err
		}
		for _, repo := range repos_ {
			repos = append(repos, convertRepoLite(repo))
		}
	}

	return repos, nil
}

// Perm fetches the named repository permissions from
// the remote system for the specified user.
func (bb *Bitbucket) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	token := oauth2.Token{AccessToken: u.Token, RefreshToken: u.Secret}
	client := NewClientToken(bb.Client, bb.Secret, &token)

	perms := new(model.Perm)
	_, err := client.FindRepo(owner, name)
	if err != nil {
		return perms, err
	}

	// if we've gotten this far we know that the user at
	// least has read access to the repository.
	perms.Pull = true

	// if the user has access to the repository hooks we
	// can deduce that the user has push and admin access.
	_, err = client.ListHooks(owner, name, &ListOpts{})
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}

	return perms, nil
}

// Script fetches the build script (.drone.yml) from the remote
// repository and returns in string format.
func (bb *Bitbucket) Script(u *model.User, r *model.Repo, b *model.Build) ([]byte, []byte, error) {
	client := NewClientToken(
		bb.Client,
		bb.Secret,
		&oauth2.Token{
			AccessToken:  u.Token,
			RefreshToken: u.Secret,
		},
	)

	// fetches the .drone.yml for the specified revision. This file
	// is required, and will error if not found
	config, err := client.FindSource(r.Owner, r.Name, b.Commit, ".drone.yml")
	if err != nil {
		return nil, nil, err
	}

	// fetches the .drone.sec for the specified revision. This file
	// is completely optional, therefore we will not return a not
	// found error
	sec, _ := client.FindSource(r.Owner, r.Name, b.Commit, ".drone.sec")

	return []byte(config.Data), []byte(sec.Data), err
}

// Status sends the commit status to the remote system.
// An example would be the GitHub pull request status.
func (bb *Bitbucket) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	return nil
}

// Netrc returns a .netrc file that can be used to clone
// private repositories from a remote system.
func (bb *Bitbucket) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return &model.Netrc{
		Machine:  "bitbucket.org",
		Login:    "x-token-auth",
		Password: u.Token,
	}, nil
}

// Activate activates a repository by creating the post-commit hook and
// adding the SSH deploy key, if applicable.
func (bb *Bitbucket) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	client := NewClientToken(
		bb.Client,
		bb.Secret,
		&oauth2.Token{
			AccessToken:  u.Token,
			RefreshToken: u.Secret,
		},
	)

	linkurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	// see if the hook already exists. If yes be sure to
	// delete so that multiple messages aren't sent.
	hooks, _ := client.ListHooks(r.Owner, r.Name, &ListOpts{})
	for _, hook := range hooks.Values {
		hookurl, err := url.Parse(hook.Url)
		if err != nil {
			return err
		}
		if hookurl.Host == linkurl.Host {
			err = client.DeleteHook(r.Owner, r.Name, hook.Uuid)
			if err != nil {
				log.Errorf("unable to delete hook %s. %s", hookurl.Host, err)
			}
			break
		}
	}

	return client.CreateHook(r.Owner, r.Name, &Hook{
		Active: true,
		Desc:   linkurl.Host,
		Events: []string{"repo:push"},
		Url:    link,
	})
}

// Deactivate removes a repository by removing all the post-commit hooks
// which are equal to link and removing the SSH deploy key.
func (bb *Bitbucket) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := NewClientToken(
		bb.Client,
		bb.Secret,
		&oauth2.Token{
			AccessToken:  u.Token,
			RefreshToken: u.Secret,
		},
	)

	linkurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	// see if the hook already exists. If yes be sure to
	// delete so that multiple messages aren't sent.
	hooks, _ := client.ListHooks(r.Owner, r.Name, &ListOpts{})
	for _, hook := range hooks.Values {
		hookurl, err := url.Parse(hook.Url)
		if err != nil {
			return err
		}
		if hookurl.Host == linkurl.Host {
			client.DeleteHook(r.Owner, r.Name, hook.Uuid)
			break
		}
	}

	return nil
}

// Hook parses the post-commit hook from the Request body
// and returns the required data in a standard format.
func (bb *Bitbucket) Hook(r *http.Request) (*model.Repo, *model.Build, error) {

	switch r.Header.Get("X-Event-Key") {
	case "repo:push":
		return bb.pushHook(r)
	case "pullrequest:created", "pullrequest:updated":
		return bb.pullHook(r)
	}

	return nil, nil, nil
}

func (bb *Bitbucket) String() string {
	return "bitbucket"
}

func (bb *Bitbucket) pushHook(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := []byte(r.FormValue("payload"))
	if len(payload) == 0 {
		defer r.Body.Close()
		payload, _ = ioutil.ReadAll(r.Body)
	}

	hook := PushHook{}
	err := json.Unmarshal(payload, &hook)
	if err != nil {
		return nil, nil, err
	}

	// the hook can container one or many changes. Since I don't
	// fully understand this yet, we will just pick the first
	// change that has branch information.
	for _, change := range hook.Push.Changes {

		// must have branch and sha information
		if change.New.Type != "branch" || change.New.Target.Hash == "" {
			continue
		}

		// return the updated repsitory information and the
		// build information.
		return convertRepo(&hook.Repo), &model.Build{
			Event:     model.EventPush,
			Commit:    change.New.Target.Hash,
			Ref:       fmt.Sprintf("refs/heads/%s", change.New.Name),
			Link:      change.New.Target.Links.Html.Href,
			Branch:    change.New.Name,
			Message:   change.New.Target.Message,
			Avatar:    hook.Actor.Links.Avatar.Href,
			Author:    hook.Actor.Login,
			Timestamp: change.New.Target.Date.UTC().Unix(),
		}, nil
	}

	return nil, nil, nil
}

func (bb *Bitbucket) pullHook(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := []byte(r.FormValue("payload"))
	if len(payload) == 0 {
		defer r.Body.Close()
		payload, _ = ioutil.ReadAll(r.Body)
	}

	hook := PullRequestHook{}
	err := json.Unmarshal(payload, &hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.PullRequest.State != "OPEN" {
		return nil, nil, nil
	}

	return convertRepo(&hook.Repo), &model.Build{
		Event:     model.EventPull,
		Commit:    hook.PullRequest.Dest.Commit.Hash,
		Ref:       fmt.Sprintf("refs/heads/%s", hook.PullRequest.Dest.Branch.Name),
		Refspec:   fmt.Sprintf("https://bitbucket.org/%s.git", hook.PullRequest.Source.Repo.FullName),
		Remote:    cloneLink(hook.PullRequest.Dest.Repo),
		Link:      hook.PullRequest.Links.Html.Href,
		Branch:    hook.PullRequest.Dest.Branch.Name,
		Message:   hook.PullRequest.Desc,
		Avatar:    hook.Actor.Links.Avatar.Href,
		Author:    hook.Actor.Login,
		Timestamp: hook.PullRequest.Updated.UTC().Unix(),
	}, nil
}
