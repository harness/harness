package bitbucketserver

// Requires the following to be set
// REMOTE_DRIVER=bitbucketserver
// REMOTE_CONFIG=https://{servername}?consumer_key={key added on the stash server for oath1}&git_username={username for clone}&git_password={password for clone}&consumer_rsa=/path/to/pem.file&open={not used yet}
// Configure application links in the bitbucket server --
// application url needs to be the base url to drone
// incoming auth needs to have the consumer key (same as the key in REMOTE_CONFIG)
// set the public key (public key from the private key added to /var/lib/bitbucketserver/private_key.pem name matters)
// consumer call back is the base url to drone plus /authorize/
// Needs a pem private key added to /var/lib/bitbucketserver/private_key.pem
// After that you should be good to go

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type BitbucketServer struct {
	URL         string
	ConsumerKey string
	GitUserName string
	GitPassword string
	ConsumerRSA string
	Open        bool
	Consumer    oauth.Consumer
}

func Load(config string) *BitbucketServer {

	url_, err := url.Parse(config)
	if err != nil {
		log.Fatalln("unable to parse remote dsn. %s", err)
	}
	params := url_.Query()
	url_.Path = ""
	url_.RawQuery = ""

	bitbucketserver := BitbucketServer{}
	bitbucketserver.URL = url_.String()
	bitbucketserver.GitUserName = params.Get("git_username")
	if bitbucketserver.GitUserName == "" {
		log.Fatalln("Must have a git_username")
	}
	bitbucketserver.GitPassword = params.Get("git_password")
	if bitbucketserver.GitPassword == "" {
		log.Fatalln("Must have a git_password")
	}
	bitbucketserver.ConsumerKey = params.Get("consumer_key")
	if bitbucketserver.ConsumerKey == "" {
		log.Fatalln("Must have a consumer_key")
	}
	bitbucketserver.ConsumerRSA = params.Get("consumer_rsa")
	if bitbucketserver.ConsumerRSA == "" {
		log.Fatalln("Must have a consumer_rsa")
	}

	bitbucketserver.Open, _ = strconv.ParseBool(params.Get("open"))

	bitbucketserver.Consumer = *NewClient(bitbucketserver.ConsumerRSA, bitbucketserver.ConsumerKey, bitbucketserver.URL)

	return &bitbucketserver
}

func (bs *BitbucketServer) Login(res http.ResponseWriter, req *http.Request) (*model.User, bool, error) {
	log.Info("Starting to login for bitbucketServer")

	log.Info("getting the requestToken")
	requestToken, url, err := bs.Consumer.GetRequestTokenAndUrl("oob")
	if err != nil {
		log.Error(err)
	}

	var code = req.FormValue("oauth_verifier")
	if len(code) == 0 {
		log.Info("redirecting to %s", url)
		http.Redirect(res, req, url, http.StatusSeeOther)
		return nil, false, nil
	}

	var request_oauth_token = req.FormValue("oauth_token")
	requestToken.Token = request_oauth_token
	accessToken, err := bs.Consumer.AuthorizeToken(requestToken, code)
	if err != nil {
		log.Error(err)
	}

	client, err := bs.Consumer.MakeHttpClient(accessToken)
	if err != nil {
		log.Error(err)
	}

	response, err := client.Get(fmt.Sprintf("%s/plugins/servlet/applinks/whoami", bs.URL))
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	bits, err := ioutil.ReadAll(response.Body)
	userName := string(bits)

	response1, err := client.Get(fmt.Sprintf("%s/rest/api/1.0/users/%s",bs.URL, userName))
	contents, err := ioutil.ReadAll(response1.Body)
	defer response1.Body.Close()
	var mUser User
	json.Unmarshal(contents, &mUser)

	user := model.User{}
	user.Login = userName
	user.Email = mUser.EmailAddress
	user.Token = accessToken.Token

	user.Avatar = avatarLink(mUser.EmailAddress)

	return &user, bs.Open, nil
}

func (bs *BitbucketServer) Auth(token, secret string) (string, error) {
	log.Info("Staring to auth for bitbucketServer. %s", token)
	if len(token) == 0 {
		return "", fmt.Errorf("Hasn't logged in yet")
	}
	return token, nil
}

func (bs *BitbucketServer) Repo(u *model.User, owner, name string) (*model.Repo, error) {
	log.Info("Staring repo for bitbucketServer with user " + u.Login + " " + owner + " " + name)

	client := NewClientWithToken(&bs.Consumer, u.Token)

	url := fmt.Sprintf("%s/rest/api/projects/%s/repos/%s",bs.URL,owner,name)
	log.Info("Trying to get " + url)
	response, err := client.Get(url)
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	bsRepo := BSRepo{}
	json.Unmarshal(contents, &bsRepo)

	cloneLink := ""
	repoLink := ""

	for _, item := range bsRepo.Links.Clone {
		if item.Name == "http" {
			cloneLink = item.Href
		}
	}
	for _, item := range bsRepo.Links.Self {
		if item.Href != "" {
			repoLink = item.Href
		}
	}
	//TODO: get the real allow tag+ infomration
	repo := &model.Repo{}
	repo.Clone = cloneLink
	repo.Link = repoLink
	repo.Name = bsRepo.Slug
	repo.Owner = bsRepo.Project.Key
	repo.AllowPush = true
	repo.FullName = fmt.Sprintf("%s/%s",bsRepo.Project.Key,bsRepo.Slug)
	repo.Branch = "master"
	repo.Kind = model.RepoGit

	return repo, nil
}

func (bs *BitbucketServer) Repos(u *model.User) ([]*model.RepoLite, error) {
	log.Info("Staring repos for bitbucketServer " + u.Login)
	var repos = []*model.RepoLite{}

	client := NewClientWithToken(&bs.Consumer, u.Token)

	response, err := client.Get(fmt.Sprintf("%s/rest/api/1.0/repos?limit=10000",bs.URL))
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	var repoResponse Repos
	json.Unmarshal(contents, &repoResponse)

	for _, repo := range repoResponse.Values {
		repos = append(repos, &model.RepoLite{
			Name:     repo.Slug,
			FullName: repo.Project.Key + "/" + repo.Slug,
			Owner:    repo.Project.Key,
		})
	}

	return repos, nil
}

func (bs *BitbucketServer) Perm(u *model.User, owner, repo string) (*model.Perm, error) {

	//TODO: find the real permissions
	log.Info("Staring perm for bitbucketServer")
	perms := new(model.Perm)
	perms.Pull = true
	perms.Admin = true
	perms.Push = true
	return perms, nil
}

func (bs *BitbucketServer) File(u *model.User, r *model.Repo, b *model.Build, f string) ([]byte, error) {
	log.Info(fmt.Sprintf("Staring file for bitbucketServer login: %s repo: %s buildevent: %s string: %s", u.Login, r.Name, b.Event, f))

	client := NewClientWithToken(&bs.Consumer, u.Token)
	fileURL := fmt.Sprintf("%s/projects/%s/repos/%s/browse/%s?raw", bs.URL, r.Owner, r.Name, f)
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

func (bs *BitbucketServer) Status(u *model.User, r *model.Repo, b *model.Build, link string) error {
	log.Info("Staring status for bitbucketServer")
	return nil
}

func (bs *BitbucketServer) Netrc(user *model.User, r *model.Repo) (*model.Netrc, error) {
	log.Info("Starting the Netrc lookup")
	u, err := url.Parse(bs.URL)
	if err != nil {
		return nil, err
	}
	return &model.Netrc{
		Machine:  u.Host,
		Login:    bs.GitUserName,
		Password: bs.GitPassword,
	}, nil
}

func (bs *BitbucketServer) Activate(u *model.User, r *model.Repo, k *model.Key, link string) error {
	log.Info(fmt.Sprintf("Staring activate for bitbucketServer user: %s repo: %s key: %s link: %s", u.Login, r.Name, k, link))
	client := NewClientWithToken(&bs.Consumer, u.Token)
	hook, err := bs.CreateHook(client, r.Owner, r.Name, "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook", link)
	if err != nil {
		return err
	}
	log.Info(hook)
	return nil
}

func (bs *BitbucketServer) Deactivate(u *model.User, r *model.Repo, link string) error {
	log.Info(fmt.Sprintf("Staring deactivating for bitbucketServer user: %s repo: %s link: %s", u.Login, r.Name, link))
	client := NewClientWithToken(&bs.Consumer, u.Token)
	err := bs.DeleteHook(client, r.Owner, r.Name, "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook", link)
	if err != nil {
		return err
	}
	return nil
}

func (bs *BitbucketServer) Hook(r *http.Request) (*model.Repo, *model.Build, error) {
	log.Info("Staring hook for bitbucketServer")
	defer r.Body.Close()
	contents, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Info(err)
	}

	var hookPost postHook
	json.Unmarshal(contents, &hookPost)

	buildModel := &model.Build{}
	buildModel.Event = model.EventPush
	buildModel.Ref = hookPost.RefChanges[0].RefID
	buildModel.Author = hookPost.Changesets.Values[0].ToCommit.Author.EmailAddress
	buildModel.Commit = hookPost.RefChanges[0].ToHash
	buildModel.Avatar = avatarLink(hookPost.Changesets.Values[0].ToCommit.Author.EmailAddress)

	//All you really need is the name and owner. That's what creates the lookup key, so it needs to match the repo info. Just an FYI
	repo := &model.Repo{}
	repo.Name = hookPost.Repository.Slug
	repo.Owner = hookPost.Repository.Project.Key
	repo.AllowTag = false
	repo.AllowDeploy = false
	repo.AllowPull = false
	repo.AllowPush = true
	repo.FullName = fmt.Sprintf("%s/%s",hookPost.Repository.Project.Key,hookPost.Repository.Slug)
	repo.Branch = "master"
	repo.Kind = model.RepoGit

	return repo, buildModel, nil
}
func (bs *BitbucketServer) String() string {
	return "bitbucketserver"
}

type HookDetail struct {
	Key           string `"json:key"`
	Name          string `"json:name"`
	Type          string `"json:type"`
	Description   string `"json:description"`
	Version       string `"json:version"`
	ConfigFormKey string `"json:configFormKey"`
}

type Hook struct {
	Enabled bool        `"json:enabled"`
	Details *HookDetail `"json:details"`
}

// Enable hook for named repository
func (bs *BitbucketServer) CreateHook(client *http.Client, project, slug, hook_key, link string) (*Hook, error) {

	// Set hook
	hookBytes := []byte(fmt.Sprintf(`{"hook-url-0":"%s"}`, link))

	// Enable hook
	enablePath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled",
		project, slug, hook_key)

	doPut(client, bs.URL+enablePath, hookBytes)

	return nil, nil
}

// Disable hook for named repository
func (bs *BitbucketServer) DeleteHook(client *http.Client, project, slug, hook_key, link string) error {
	enablePath := fmt.Sprintf("/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled",
		project, slug, hook_key)
	doDelete(client, bs.URL+enablePath)

	return nil
}
