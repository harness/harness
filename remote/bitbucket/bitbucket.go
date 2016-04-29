package bitbucket

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/drone/drone/remote/bitbucket/internal"
	"github.com/drone/drone/shared/httputil"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

type config struct {
	Client string
	Secret string
}

// New returns a new remote Configuration for integrating with the Bitbucket
// repository hosting service at https://bitbucket.org
func New(client, secret string) remote.Remote {
	return &config{
		Client: client,
		Secret: secret,
	}
}

// helper function to return the bitbucket oauth2 client
func (c *config) newClient(u *model.User) *internal.Client {
	return internal.NewClientToken(
		c.Client,
		c.Secret,
		&oauth2.Token{
			AccessToken:  u.Token,
			RefreshToken: u.Secret,
		},
	)
}

func (c *config) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	config := &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
		Endpoint:     bitbucket.Endpoint,
		RedirectURL:  fmt.Sprintf("%s/authorize", httputil.GetURL(req)),
	}

	var code = req.FormValue("code")
	if len(code) == 0 {
		http.Redirect(res, req, config.AuthCodeURL("drone"), http.StatusSeeOther)
		return nil, nil
	}

	var token, err = config.Exchange(oauth2.NoContext, code)
	if err != nil {
		return nil, err
	}

	client := internal.NewClient(config.Client(oauth2.NoContext, token))
	curr, err := client.FindCurrent()
	if err != nil {
		return nil, err
	}

	return convertUser(curr, token), nil
}

func (c *config) Auth(token, secret string) (string, error) {
	client := internal.NewClientToken(
		c.Client,
		c.Secret,
		&oauth2.Token{
			AccessToken:  token,
			RefreshToken: secret,
		},
	)
	user, err := client.FindCurrent()
	if err != nil {
		return "", err
	}
	return user.Login, nil
}

func (c *config) Refresh(user *model.User) (bool, error) {
	config := &oauth2.Config{
		ClientID:     c.Client,
		ClientSecret: c.Secret,
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

func (c *config) Teams(u *model.User) ([]*model.Team, error) {
	opts := &internal.ListTeamOpts{
		PageLen: 100,
		Role:    "member",
	}
	resp, err := c.newClient(u).ListTeams(opts)
	if err != nil {
		return nil, err
	}
	return convertTeamList(resp.Values), nil
}

func (c *config) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	repo, err := c.newClient(u).FindRepo(owner, name)
	if err != nil {
		return nil, err
	}
	return convertRepo(repo), nil
}

func (c *config) Repos(u *model.User) ([]*model.RepoLite, error) {
	client := c.newClient(u)

	var repos []*model.RepoLite

	// gets a list of all accounts to query, including the
	// user's account and all team accounts.
	logins := []string{u.Login}
	resp, err := client.ListTeams(&internal.ListTeamOpts{PageLen: 100, Role: "member"})
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

func (c *config) Perm(u *model.User, owner, name string) (*model.Perm, error) {
	client := c.newClient(u)

	perms := new(model.Perm)
	_, err := client.FindRepo(owner, name)
	if err != nil {
		return perms, err
	}

	// if we've gotten this far we know that the user at least has read access
	// to the repository.
	perms.Pull = true

	// if the user has access to the repository hooks we can deduce that the user
	// has push and admin access.
	_, err = client.ListHooks(owner, name, &internal.ListOpts{})
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	return perms, nil
}

func (c *config) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	config, err := c.newClient(u).FindSource(r.Owner, r.Name, b.Commit, f)
	if err != nil {
		return nil, err
	}
	return []byte(config.Data), err
}

func (c *config) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	status := internal.BuildStatus{
		State: getStatus(b.Status),
		Desc:  getDesc(b.Status),
		Key:   "Drone",
		Url:   link,
	}
	return c.newClient(u).CreateStatus(r.Owner, r.Name, b.Commit, &status)
}

func (c *config) Netrc(u *model.User, r *model.Repo) (*model.Netrc, error) {
	return &model.Netrc{
		Machine:  "bitbucket.org",
		Login:    "x-token-auth",
		Password: u.Token,
	}, nil
}

func (c *config) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	rawurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	// deletes any previously created hooks
	if err := c.Deactivate(u, r, link); err != nil {
		// we can live with failure here. Things happen and manually scrubbing
		// hooks is certinaly not the end of the world.
	}

	return c.newClient(u).CreateHook(r.Owner, r.Name, &internal.Hook{
		Active: true,
		Desc:   rawurl.Host,
		Events: []string{"repo:push"},
		Url:    link,
	})
}

func (c *config) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := c.newClient(u)

	linkurl, err := url.Parse(link)
	if err != nil {
		return err
	}

	hooks, err := client.ListHooks(r.Owner, r.Name, &internal.ListOpts{})
	if err != nil {
		return nil // we can live with undeleted hooks
	}

	for _, hook := range hooks.Values {
		hookurl, err := url.Parse(hook.Url)
		if err != nil {
			continue
		}
		if hookurl.Host == linkurl.Host {
			client.DeleteHook(r.Owner, r.Name, hook.Uuid)
			break // we can live with undeleted hooks
		}
	}

	return nil
}

func (c *config) Hook(r *http.Request) (*model.Repo, *model.Build, error) {

	switch r.Header.Get("X-Event-Key") {
	case "repo:push":
		return c.pushHook(r)
	case "pullrequest:created", "pullrequest:updated":
		return c.pullHook(r)
	}

	return nil, nil, nil
}

func (c *config) pushHook(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := []byte(r.FormValue("payload"))
	if len(payload) == 0 {
		defer r.Body.Close()
		payload, _ = ioutil.ReadAll(r.Body)
	}

	hook := internal.PushHook{}
	err := json.Unmarshal(payload, &hook)
	if err != nil {
		return nil, nil, err
	}

	// the hook can container one or many changes. Since I don't
	// fully understand this yet, we will just pick the first
	// change that has branch information.
	for _, change := range hook.Push.Changes {

		// must have sha information
		if change.New.Target.Hash == "" {
			continue
		}
		// we only support tag and branch pushes for now
		buildEventType := model.EventPush
		buildRef := fmt.Sprintf("refs/heads/%s", change.New.Name)
		if change.New.Type == "tag" || change.New.Type == "annotated_tag" || change.New.Type == "bookmark" {
			buildEventType = model.EventTag
			buildRef = fmt.Sprintf("refs/tags/%s", change.New.Name)
		} else if change.New.Type != "branch" && change.New.Type != "named_branch" {
			continue
		}

		// return the updated repository information and the
		// build information.
		// TODO(bradrydzewski) uses unit tested conversion function
		return convertRepo(&hook.Repo), &model.Build{
			Event:     buildEventType,
			Commit:    change.New.Target.Hash,
			Ref:       buildRef,
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

func (c *config) pullHook(r *http.Request) (*model.Repo, *model.Build, error) {
	payload := []byte(r.FormValue("payload"))
	if len(payload) == 0 {
		defer r.Body.Close()
		payload, _ = ioutil.ReadAll(r.Body)
	}

	hook := internal.PullRequestHook{}
	err := json.Unmarshal(payload, &hook)
	if err != nil {
		return nil, nil, err
	}
	if hook.PullRequest.State != "OPEN" {
		return nil, nil, nil
	}

	// TODO(bradrydzewski) uses unit tested conversion function
	return convertRepo(&hook.Repo), &model.Build{
		Event:     model.EventPull,
		Commit:    hook.PullRequest.Dest.Commit.Hash,
		Ref:       fmt.Sprintf("refs/heads/%s", hook.PullRequest.Dest.Branch.Name),
		Refspec:   fmt.Sprintf("https://bitbucket.org/%s.git", hook.PullRequest.Source.Repo.FullName),
		Remote:    cloneLink(&hook.PullRequest.Dest.Repo),
		Link:      hook.PullRequest.Links.Html.Href,
		Branch:    hook.PullRequest.Dest.Branch.Name,
		Message:   hook.PullRequest.Desc,
		Avatar:    hook.Actor.Links.Avatar.Href,
		Author:    hook.Actor.Login,
		Timestamp: hook.PullRequest.Updated.UTC().Unix(),
	}, nil
}
