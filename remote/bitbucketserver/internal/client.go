package internal

import (
	"net/http"
	log "github.com/Sirupsen/logrus"
	"github.com/mrjones/oauth"
	"github.com/drone/drone/model"
	"fmt"
	"io/ioutil"
	"encoding/json"
	"strings"
	"encoding/hex"
	"crypto/md5"
	"net/url"
	"bytes"
)

const (
	currentUserId = "%s/plugins/servlet/applinks/whoami"
	pathUser   = "%s/rest/api/1.0/users/%s"
	pathRepo   = "%s/rest/api/1.0/projects/%s/repos/%s"
	pathRepos  = "%s/rest/api/1.0/repos?limit=%s"
	pathHook   = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s"
	pathSource = "%s/projects/%s/repos/%s/browse/%s?raw"
	hookName = "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook"
	pathHookEnabled = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled"
)

type Client struct {
	*http.Client
	base string
	accessToken string
}

func NewClientWithToken(url string, Consumer *oauth.Consumer, AccessToken string) *Client {
	var token oauth.AccessToken
	token.Token = AccessToken
	client, err := Consumer.MakeHttpClient(&token)
	if err != nil {
		log.Error(err)
	}
	return &Client{client, url, AccessToken}
}

func (c *Client) FindCurrentUser() (*model.User, error) {
	CurrentUserIdResponse, err := c.Get(fmt.Sprintf(currentUserId, c.base))
	if err != nil {
		return nil, err
	}
	defer CurrentUserIdResponse.Body.Close()
	bits, err := ioutil.ReadAll(CurrentUserIdResponse.Body)
	if err != nil {
		return nil, err
	}
	login := string(bits)

	// TODO errors should never be ignored like this
	CurrentUserResponse, err := c.Get(fmt.Sprintf(pathUser, c.base, login))
	if err != nil {
		return nil, err
	}
	defer CurrentUserResponse.Body.Close()

	contents, err := ioutil.ReadAll(CurrentUserResponse.Body)
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
		Token:  c.accessToken,
		Avatar: avatarLink(user.EmailAddress),
	}, nil

}

func (c *Client) FindRepo(owner string, name string) (*model.Repo, error){
	urlString := fmt.Sprintf(pathRepo, c.base, owner, name)
	response, err := c.Get(urlString)
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
		IsPrivate: true, // Since we have to use Netrc it has to always be private :/
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

func (c *Client) FindRepos() ([]*model.RepoLite, error) {
	response, err := c.Get(fmt.Sprintf(pathRepos, c.base))
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
	var repos = []*model.RepoLite{}
	for _, repo := range repoResponse.Values {
		repos = append(repos, &model.RepoLite{
			Name:     repo.Slug,
			FullName: repo.Project.Key + "/" + repo.Slug,
			Owner:    repo.Project.Key,
		})
	}

	return repos, nil
}

func (c *Client) FindRepoPerms(owner string, repo string) (*model.Perm, error) {
	perms := new(model.Perm)
	// If you don't have access return none right away
	_, err := c.FindRepo(owner, repo)
	if err != nil {
		return perms, err
	}
	// Must have admin to be able to list hooks. If have access the enable perms
	_, err = c.Get(fmt.Sprintf(pathHook, c.base, owner, repo,hookName))
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	perms.Pull = true
	return perms, nil
}

func (c *Client) FindFileForRepo(owner string, repo string, fileName string) ([]byte, error) {
	response, err := c.Get(fmt.Sprintf(pathSource, c.base, owner, repo, fileName))
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

func(c *Client) CreateHook(owner string, name string, callBackLink string) error {
	// Set hook
	//TODO: Check existing and add up to 5
	hookBytes := []byte(fmt.Sprintf(`{"hook-url-0":"%s"}`, callBackLink))
	return c.doPut(fmt.Sprintf(pathHookEnabled,c.base, owner, name, hookName), hookBytes)
}

func(c *Client) DeleteHook(owner string, name string, link string) error {
	//TODO: eventially should only delete the link callback
	return c.doDelete(fmt.Sprintf(pathHookEnabled,c.base, owner, name, hookName ))
}

func avatarLink(email string) (url string) {
	hasher := md5.New()
	hasher.Write([]byte(strings.ToLower(email)))
	emailHash := fmt.Sprintf("%v", hex.EncodeToString(hasher.Sum(nil)))
	avatarURL := fmt.Sprintf("https://www.gravatar.com/avatar/%s.jpg", emailHash)
	log.Info(avatarURL)
	return avatarURL
}


//Helper function to help create the hook
func(c *Client) doPut(url string, body []byte) error {
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	request.Header.Add("Content-Type", "application/json")
	response, err := c.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil

}

//Helper function to do delete on the hook
func(c *Client) doDelete(url string) error {
	request, err := http.NewRequest("DELETE", url, nil)
	response, err := c.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}