package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/mrjones/oauth"
	"io/ioutil"
	"net/http"
	"io"
)

const (
	currentUserId   = "%s/plugins/servlet/applinks/whoami"
	pathUser        = "%s/rest/api/1.0/users/%s"
	pathRepo        = "%s/rest/api/1.0/projects/%s/repos/%s"
	pathRepos       = "%s/rest/api/1.0/repos?limit=%s"
	pathHook        = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s"
	pathSource      = "%s/projects/%s/repos/%s/browse/%s?at=%s&raw"
	hookName        = "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook"
	pathHookEnabled = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled"
	pathStatus      = "%s/rest/build-status/1.0/commits/%s"
)

type Client struct {
	client      *http.Client
	base        string
	accessToken string
}

func NewClientWithToken(url string, consumer *oauth.Consumer, AccessToken string) *Client {
	var token oauth.AccessToken
	token.Token = AccessToken
	client, err := consumer.MakeHttpClient(&token)
	log.Debug(fmt.Printf("Create client: %+v %s\n", token, url))
	if err != nil {
		log.Error(err)
	}
	return &Client{client, url, AccessToken}
}

func (c *Client) FindCurrentUser() (*User, error) {
	CurrentUserIdResponse, err := c.client.Get(fmt.Sprintf(currentUserId, c.base))
	if err != nil {
		return nil, err
	}
	defer CurrentUserIdResponse.Body.Close()
	bits, err := ioutil.ReadAll(CurrentUserIdResponse.Body)
	if err != nil {
		return nil, err
	}
	login := string(bits)

	CurrentUserResponse, err := c.client.Get(fmt.Sprintf(pathUser, c.base, login))
	if err != nil {
		return nil, err
	}
	defer CurrentUserResponse.Body.Close()

	contents, err := ioutil.ReadAll(CurrentUserResponse.Body)
	if err != nil {
		return nil, err
	}

	var user User
	err = json.Unmarshal(contents, &user)
	if err != nil {
		return nil, err
	}

	return &user, nil

}

func (c *Client) FindRepo(owner string, name string) (*Repo, error) {
	urlString := fmt.Sprintf(pathRepo, c.base, owner, name)
	response, err := c.client.Get(urlString)
	if err != nil {
		log.Error(err)
	}
	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)
	repo := Repo{}
	err = json.Unmarshal(contents, &repo)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (c *Client) FindRepos() ([]*Repo, error) {
	requestUrl := fmt.Sprintf(pathRepos, c.base, "1000")
	log.Debug(fmt.Printf("request :%s", requestUrl))
	response, err := c.client.Get(requestUrl)
	if err != nil {
		return nil, err
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
	log.Debug(fmt.Printf("repoResponse: %+v\n", repoResponse))

	return repoResponse.Values, nil
}

func (c *Client) FindRepoPerms(owner string, repo string) (*model.Perm, error) {
	perms := new(model.Perm)
	// If you don't have access return none right away
	_, err := c.FindRepo(owner, repo)
	if err != nil {
		return perms, err
	}
	// Must have admin to be able to list hooks. If have access the enable perms
	_, err = c.client.Get(fmt.Sprintf(pathHook, c.base, owner, repo, hookName))
	if err == nil {
		perms.Push = true
		perms.Admin = true
	}
	perms.Pull = true
	log.Debug(fmt.Printf("Perms: %+v\n", perms))
	return perms, nil
}

func (c *Client) FindFileForRepo(owner string, repo string, fileName string, ref string) ([]byte, error) {
	response, err := c.client.Get(fmt.Sprintf(pathSource, c.base, owner, repo, fileName, ref))
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

func (c *Client) CreateHook(owner string, name string, callBackLink string) error {
	// Set hook
	//TODO: Check existing and add up to 5
	hookBytes := []byte(fmt.Sprintf(`{"hook-url-0":"%s"}`, callBackLink))
	return c.doPut(fmt.Sprintf(pathHookEnabled, c.base, owner, name, hookName), hookBytes)
}

func (c *Client) CreateStatus(revision string, status *BuildStatus) error {
	uri := fmt.Sprintf(pathStatus, c.base, revision)
	return c.doPost(uri, status)
}

func (c *Client) DeleteHook(owner string, name string, link string) error {
	//TODO: eventially should only delete the link callback
	return c.doDelete(fmt.Sprintf(pathHookEnabled, c.base, owner, name, hookName))
}

//TODO: make these as as general do with the action

//Helper function to help create the hook
func (c *Client) doPut(url string, body []byte) error {
	request, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	request.Header.Add("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}


//Helper function to help create the hook
func (c *Client) doPost(url string, status *BuildStatus) error {
	// write it to the body of the request.
	var buf io.ReadWriter
	if status != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(status)
		if err != nil {
			return err
		}
	}
	request, err := http.NewRequest("POST", url, buf)
	request.Header.Add("Content-Type", "application/json")
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}





//Helper function to do delete on the hook
func (c *Client) doDelete(url string) error {
	request, err := http.NewRequest("DELETE", url, nil)
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return nil
}
