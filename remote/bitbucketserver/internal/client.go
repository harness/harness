package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	log "github.com/Sirupsen/logrus"
	"github.com/drone/drone/model"
	"github.com/mrjones/oauth"
	"strings"
)

const (
	currentUserId    = "%s/plugins/servlet/applinks/whoami"
	pathUser         = "%s/rest/api/1.0/users/%s"
	pathRepo         = "%s/rest/api/1.0/projects/%s/repos/%s"
	pathRepos        = "%s/rest/api/1.0/repos?start=%s&limit=%s"
	pathHook         = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s"
	pathSource       = "%s/projects/%s/repos/%s/browse/%s?at=%s&raw"
	hookName         = "com.atlassian.stash.plugin.stash-web-post-receive-hooks-plugin:postReceiveHook"
	pathHookDetails  = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s"
	pathHookEnabled  = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/enabled"
	pathHookSettings = "%s/rest/api/1.0/projects/%s/repos/%s/settings/hooks/%s/settings"
	pathStatus       = "%s/rest/build-status/1.0/commits/%s"
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
	return c.paginatedRepos(0)
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
	hookDetails, err := c.GetHookDetails(owner, name)
	if err != nil {
		return err
	}
	var hooks []string
	if hookDetails.Enabled {
		hookSettings, err := c.GetHooks(owner, name)
		if err != nil {
			return err
		}
		hooks = hookSettingsToArray(hookSettings)

	}
	if !stringInSlice(callBackLink, hooks) {
		hooks = append(hooks, callBackLink)
	}

	putHookSettings := arrayToHookSettings(hooks)
	hookBytes, err := json.Marshal(putHookSettings)
	return c.doPut(fmt.Sprintf(pathHookEnabled, c.base, owner, name, hookName), hookBytes)
}

func (c *Client) CreateStatus(revision string, status *BuildStatus) error {
	uri := fmt.Sprintf(pathStatus, c.base, revision)
	return c.doPost(uri, status)
}

func (c *Client) DeleteHook(owner string, name string, link string) error {

	hookSettings, err := c.GetHooks(owner, name)
	if err != nil {
		return err
	}
	putHooks := filter(hookSettingsToArray(hookSettings), func(item string) bool {

		return !strings.Contains(item, link)
	})
	putHookSettings := arrayToHookSettings(putHooks)
	hookBytes, err := json.Marshal(putHookSettings)
	return c.doPut(fmt.Sprintf(pathHookEnabled, c.base, owner, name, hookName), hookBytes)
}

func (c *Client) GetHookDetails(owner string, name string) (*HookPluginDetails, error) {
	urlString := fmt.Sprintf(pathHookDetails, c.base, owner, name, hookName)
	response, err := c.client.Get(urlString)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	hookDetails := HookPluginDetails{}
	err = json.NewDecoder(response.Body).Decode(&hookDetails)
	return &hookDetails, err
}

func (c *Client) GetHooks(owner string, name string) (*HookSettings, error) {
	urlString := fmt.Sprintf(pathHookSettings, c.base, owner, name, hookName)
	response, err := c.client.Get(urlString)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	hookSettings := HookSettings{}
	err = json.NewDecoder(response.Body).Decode(&hookSettings)
	return &hookSettings, err
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

//Helper function to get repos paginated
func (c *Client) paginatedRepos(start int) ([]*Repo, error) {
	limit := 1000
	requestUrl := fmt.Sprintf(pathRepos, c.base, strconv.Itoa(start), strconv.Itoa(limit))
	response, err := c.client.Get(requestUrl)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	var repoResponse Repos
	err = json.NewDecoder(response.Body).Decode(&repoResponse)
	if err != nil {
		return nil, err
	}
	if !repoResponse.IsLastPage {
		reposList, err := c.paginatedRepos(start + limit)
		if err != nil {
			return nil, err
		}
		repoResponse.Values = append(repoResponse.Values, reposList...)
	}
	return repoResponse.Values, nil
}

func filter(vs []string, f func(string) bool) []string {
	var vsf []string
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}

//TODO: find a clean way of doing these next two methods- bitbucket server hooks only support 20 cb hooks
func arrayToHookSettings(hooks []string) HookSettings {
	hookSettings := HookSettings{}
	for loc, value := range hooks {
		switch loc {
		case 0:
			hookSettings.HookURL0 = value
		case 1:
			hookSettings.HookURL1 = value
		case 2:
			hookSettings.HookURL2 = value
		case 3:
			hookSettings.HookURL3 = value
		case 4:
			hookSettings.HookURL4 = value
		case 5:
			hookSettings.HookURL5 = value
		case 6:
			hookSettings.HookURL6 = value
		case 7:
			hookSettings.HookURL7 = value
		case 8:
			hookSettings.HookURL8 = value
		case 9:
			hookSettings.HookURL9 = value
		case 10:
			hookSettings.HookURL10 = value
		case 11:
			hookSettings.HookURL11 = value
		case 12:
			hookSettings.HookURL12 = value
		case 13:
			hookSettings.HookURL13 = value
		case 14:
			hookSettings.HookURL14 = value
		case 15:
			hookSettings.HookURL15 = value
		case 16:
			hookSettings.HookURL16 = value
		case 17:
			hookSettings.HookURL17 = value
		case 18:
			hookSettings.HookURL18 = value
		case 19:
			hookSettings.HookURL19 = value

			//Since there's only 19 hooks it will add to the latest if it doesn't exist :/
		default:
			hookSettings.HookURL19 = value
		}
	}
	return hookSettings
}

func hookSettingsToArray(hookSettings *HookSettings) []string {
	var hooks []string

	if hookSettings.HookURL0 != "" {
		hooks = append(hooks, hookSettings.HookURL0)
	}
	if hookSettings.HookURL1 != "" {
		hooks = append(hooks, hookSettings.HookURL1)
	}
	if hookSettings.HookURL2 != "" {
		hooks = append(hooks, hookSettings.HookURL2)
	}
	if hookSettings.HookURL3 != "" {
		hooks = append(hooks, hookSettings.HookURL3)
	}
	if hookSettings.HookURL4 != "" {
		hooks = append(hooks, hookSettings.HookURL4)
	}
	if hookSettings.HookURL5 != "" {
		hooks = append(hooks, hookSettings.HookURL5)
	}
	if hookSettings.HookURL6 != "" {
		hooks = append(hooks, hookSettings.HookURL6)
	}
	if hookSettings.HookURL7 != "" {
		hooks = append(hooks, hookSettings.HookURL7)
	}
	if hookSettings.HookURL8 != "" {
		hooks = append(hooks, hookSettings.HookURL8)
	}
	if hookSettings.HookURL9 != "" {
		hooks = append(hooks, hookSettings.HookURL9)
	}
	if hookSettings.HookURL10 != "" {
		hooks = append(hooks, hookSettings.HookURL10)
	}
	if hookSettings.HookURL11 != "" {
		hooks = append(hooks, hookSettings.HookURL11)
	}
	if hookSettings.HookURL12 != "" {
		hooks = append(hooks, hookSettings.HookURL12)
	}
	if hookSettings.HookURL13 != "" {
		hooks = append(hooks, hookSettings.HookURL13)
	}
	if hookSettings.HookURL14 != "" {
		hooks = append(hooks, hookSettings.HookURL14)
	}
	if hookSettings.HookURL15 != "" {
		hooks = append(hooks, hookSettings.HookURL15)
	}
	if hookSettings.HookURL16 != "" {
		hooks = append(hooks, hookSettings.HookURL16)
	}
	if hookSettings.HookURL17 != "" {
		hooks = append(hooks, hookSettings.HookURL17)
	}
	if hookSettings.HookURL18 != "" {
		hooks = append(hooks, hookSettings.HookURL18)
	}
	if hookSettings.HookURL19 != "" {
		hooks = append(hooks, hookSettings.HookURL19)
	}
	return hooks
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
