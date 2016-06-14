package bitbucketserver

// WARNING! This is an work-in-progress patch and does not yet conform to the coding,
// quality or security standards expected of this project. Please use with caution.

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/drone/drone/remote"
	"github.com/mrjones/oauth"
	"strings"
)

// Opts defines configuration options.
type Opts struct {
	URL         string // Stash server url.
	Username    string // Git machine account username.
	Password    string // Git machine account password.
	ConsumerKey string // Oauth1 consumer key.
	ConsumerRSA string // Oauth1 consumer key file.

	SkipVerify bool // Skip ssl verification.
}

// New returns a Remote implementation that integrates with Bitbucket Server,
// the on-premise edition of Bitbucket Cloud, formerly known as Stash.
func New(opts Opts) (remote.Remote, error) {
	bb := &client{
		URL:         opts.URL,
		ConsumerKey: opts.ConsumerKey,
		ConsumerRSA: opts.ConsumerRSA,
		GitUserName: opts.Username,
		GitPassword: opts.Password,
	}

	switch {
	case bb.GitUserName == "":
		return nil, fmt.Errorf("Must have a git machine account username")
	case bb.GitPassword == "":
		return nil, fmt.Errorf("Must have a git machine account password")
	case bb.ConsumerKey == "":
		return nil, fmt.Errorf("Must have a oauth1 consumer key")
	case bb.ConsumerRSA == "":
		return nil, fmt.Errorf("Must have a oauth1 consumer key file")
	}

	keyfile, err := ioutil.ReadFile(bb.ConsumerRSA)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(keyfile)
	bb.PrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// TODO de-referencing is a bit weird and may not behave as expected, and could
	// have race conditions. Instead store the parsed key (I already did this above)
	// and then pass the parsed private key when creating the Bitbucket client.
	bb.Consumer = *NewClient(bb.ConsumerRSA, bb.ConsumerKey, bb.URL)
	return bb, nil
}

type client struct {
	URL         string
	ConsumerKey string
	GitUserName string
	GitPassword string
	ConsumerRSA string
	PrivateKey  *rsa.PrivateKey
	Consumer    oauth.Consumer
}

func (c *client) Login(res http.ResponseWriter, req *http.Request) (*model.User, error) {
	requestToken, url, err := c.Consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		return nil, err
	}

	var code = req.FormValue("oauth_verifier")
	if len(code) == 0 {
		http.Redirect(res, req, url, http.StatusSeeOther)
		return nil, nil
	}

	requestToken.Token = req.FormValue("oauth_token")
	accessToken, err := c.Consumer.AuthorizeToken(requestToken, code)
	if err != nil {
		return nil, err
	}

	client, err := c.Consumer.MakeHttpClient(accessToken)
	if err != nil {
		return nil, err
	}

	response, err := client.Get(fmt.Sprintf("%s/plugins/servlet/applinks/whoami", c.URL))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	bits, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	login := string(bits)

	// TODO errors should never be ignored like this
	response1, err := client.Get(fmt.Sprintf("%s/rest/api/1.0/users/%s", c.URL, login))
	if err != nil {
		return nil, err
	}
	defer response1.Body.Close()

	contents, err := ioutil.ReadAll(response1.Body)
	if err !=nil {
		return nil, err
	}

	var user User
	err = json.Unmarshal(contents, &user)
	if err != nil {
		return nil, err
	}
	return &model.User{
		Login:  login,
		Email:  user.EmailAddress,
		Token:  accessToken.Token,
		Avatar: avatarLink(user.EmailAddress),
	}, nil
}

// Auth is not supported by the Stash driver.
func (*client) Auth(token, secret string) (string, error) {
	return "", fmt.Errorf("Not Implemented")
}

// Teams is not supported by the Stash driver.
func (*client) Teams(u *model.User) ([]*model.Team, error) {
	var teams []*model.Team
	return teams, nil
}

func (c *client) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	client := NewClientWithToken(&c.Consumer, u.Token)
	repo , err := c.FindRepo(client,owner,name)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (c *client) Repos(u *model.User) ([]*model.RepoLite, error) {

	var repos = []*model.RepoLite{}

	client := NewClientWithToken(&c.Consumer, u.Token)

	response, err := client.Get(fmt.Sprintf("%s/rest/api/1.0/repos?limit=10000", c.URL))
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var repoResponse Repos
	err = json.Unmarshal(contents, &repoResponse)
	if err != nil {
		return nil, err
	}

	for _, repo := range repoResponse.Values {
		repos = append(repos, &model.RepoLite{
			Name:     repo.Slug,
			FullName: repo.Project.Key + "/" + repo.Slug,
			Owner:    repo.Project.Key,
		})
	}

	return repos, nil
}

func (c *client) Perm(u *model.User, owner, repo string) (*model.Perm, error) {
	client := NewClientWithToken(&c.Consumer, u.Token)
	perms := new(model.Perm)

	// If you don't have access return none right away
	_, err := c.FindRepo(client, owner, repo)
	if err != nil {
		return perms, err
	}

	// Must have admin to be able to list hooks. If have access the enable perms
	_, err = client.Get(fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s", c.URL, owner, repo,"com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook"))
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	perms.Pull = true
	return perms, nil
}

func (c *client) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	log.Info(fmt.Sprintf("Staring file for bitbucketServer login: %s repo: %s buildevent: %s string: %s", u.Login, r.Name, b.Event, f))

	client := NewClientWithToken(&c.Consumer, u.Token)
	fileURL := fmt.Sprintf("%s/projects/%s/repos/%s/browse/%s?raw", c.URL, r.Owner, r.Name, f)
	log.Info(fileURL)
	response, err := client.Get(fileURL)
	if err != nil {
		log.Error(err)
	}
	if response.StatusCode == 404 {
		return nil, nil
	}
	defer response.Body.Close()
	responseBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error(err)
	}

	return responseBytes, nil
}

// Status is not supported by the Gogs driver.
func (*client) Status(*model.User, *model.Repo, *model.Build, string) error {
	return nil
}

func (c *client) Netrc(user *model.User, r *model.Repo) (*model.Netrc, error) {
	u, err := url.Parse(c.URL)
	if err != nil {
		return nil, err
	}
	//remove the port
	tmp := strings.Split(u.Host, ":")
	var host = tmp[0]

	if err != nil {
		return nil, err
	}
	return &model.Netrc{
		Machine:  host,
		Login:    c.GitUserName,
		Password: c.GitPassword,
	}, nil
}

func (c *client) Activate(u *model.User, r *model.Repo, link string) error {
	client := NewClientWithToken(&c.Consumer, u.Token)
	hook, err := c.CreateHook(client, r.Owner, r.Name, "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook", link)
	if err != nil {
		return err
	}
	log.Info(hook)
	return nil
}

func (c *client) Deactivate(u *model.User, r *model.Repo, link string) error {
	client := NewClientWithToken(&c.Consumer, u.Token)
	err := c.DeleteHook(client, r.Owner, r.Name, "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook", link)
	if err != nil {
		return err
	}
	return nil
}

func (c *client) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	hook := new(postHook)
	if err := json.NewDecoder(r.Body).Decode(hook); err != nil {
		return nil, nil, err
	}

	build := &model.Build{
		Event:  model.EventPush,
		Ref:    hook.RefChanges[0].RefID,                               // TODO check for index Values
		Author: hook.Changesets.Values[0].ToCommit.Author.EmailAddress, // TODO check for index Values
		Commit: hook.RefChanges[0].ToHash,                              // TODO check for index value
		Avatar: avatarLink(hook.Changesets.Values[0].ToCommit.Author.EmailAddress),
	}

	repo := &model.Repo{
		Name:     hook.Repository.Slug,
		Owner:    hook.Repository.Project.Key,
		FullName: fmt.Sprintf("%s/%s", hook.Repository.Project.Key, hook.Repository.Slug),
		Branch:   "master",
		Kind:     model.RepoGit,
	}

	return repo, build, nil
}

type HookDetail struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	ConfigFormKey string `json:"configFormKey"`
}

type Hook struct {
	Enabled bool        `json:"enabled"`
	Details *HookDetail `json:"details"`
}

// Enable hook for named repository
func (bs *client) CreateHook(client *http.Client, project, slug, hook_key, link string) (*Hook, error) {

	// Set hook
	hookBytes := []byte(fmt.Sprintf(`{"hook-url-0":"%s"}`, link))

	// Enable hook
	enablePath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled",
		project, slug, hook_key)

	doPut(client, bs.URL+enablePath, hookBytes)

	return nil, nil
}

// Disable hook for named repository
func (bs *client) DeleteHook(client *http.Client, project, slug, hook_key, link string) error {
	enablePath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled",
		project, slug, hook_key)
	doDelete(client, bs.URL+enablePath)

	return nil
}

func (c *client) FindRepo(client *http.Client, owner string, name string) (*model.Repo, error){

	urlString := fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s", c.URL, owner, name)

	response, err := client.Get(urlString)
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	bsRepo := BSRepo{}
	err = json.Unmarshal(contents, &bsRepo)
	if err !=nil {
		return nil, err
	}
	repo := &model.Repo{
		Name:      bsRepo.Slug,
		Owner:     bsRepo.Project.Key,
		Branch:    "master",
		Kind:      model.RepoGit,
		IsPrivate: true, // TODO(josmo) possibly set this as a setting - must always be private to use netrc
		FullName:  fmt.Sprintf("%s/%s", bsRepo.Project.Key, bsRepo.Slug),
	}

	for _, item := range bsRepo.Links.Clone {
		if item.Name == "http" {
			uri, err := url.Parse(item.Href)
			if err != nil {
				return nil, err
			}
			uri.User = nil
			repo.Clone = uri.String()
		}
	}
	for _, item := range bsRepo.Links.Self {
		if item.Href != "" {
			repo.Link = item.Href
		}
	}

	return repo, nil
}
