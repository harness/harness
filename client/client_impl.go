package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/drone/drone/model"
	"golang.org/x/net/proxy"
	"golang.org/x/oauth2"
)

const (
	pathPull     = "%s/api/queue/pull/%s/%s"
	pathWait     = "%s/api/queue/wait/%d"
	pathStream   = "%s/api/queue/stream/%d"
	pathPush     = "%s/api/queue/status/%d"
	pathPing     = "%s/api/queue/ping"
	pathLogs     = "%s/api/queue/logs/%d"
	pathLogsAuth = "%s/api/queue/logs/%d?access_token=%s"

	pathSelf          = "%s/api/user"
	pathFeed          = "%s/api/user/feed"
	pathRepos         = "%s/api/user/repos"
	pathRepo          = "%s/api/repos/%s/%s"
	pathChown         = "%s/api/repos/%s/%s/chown"
	pathEncrypt       = "%s/api/repos/%s/%s/encrypt"
	pathBuilds        = "%s/api/repos/%s/%s/builds"
	pathBuild         = "%s/api/repos/%s/%s/builds/%v"
	pathJob           = "%s/api/repos/%s/%s/builds/%d/%d"
	pathLog           = "%s/api/repos/%s/%s/logs/%d/%d"
	pathKey           = "%s/api/repos/%s/%s/key"
	pathSign          = "%s/api/repos/%s/%s/sign"
	pathRepoSecrets   = "%s/api/repos/%s/%s/secrets"
	pathRepoSecret    = "%s/api/repos/%s/%s/secrets/%s"
	pathTeamSecrets   = "%s/api/teams/%s/secrets"
	pathTeamSecret    = "%s/api/teams/%s/secrets/%s"
	pathGlobalSecrets = "%s/api/global/secrets"
	pathGlobalSecret  = "%s/api/global/secrets/%s"
	pathNodes         = "%s/api/nodes"
	pathNode          = "%s/api/nodes/%d"
	pathUsers         = "%s/api/users"
	pathUser          = "%s/api/users/%s"
	pathBuildQueue    = "%s/api/builds"
	pathAgent         = "%s/api/agents"
)

type client struct {
	client *http.Client
	token  string // auth token
	base   string // base url
}

// NewClient returns a client at the specified url.
func NewClient(uri string) Client {
	return &client{client: http.DefaultClient, base: uri}
}

// NewClientToken returns a client at the specified url that authenticates all
// outbound requests with the given token.
func NewClientToken(uri, token string) Client {
	config := new(oauth2.Config)
	auther := config.Client(oauth2.NoContext, &oauth2.Token{AccessToken: token})
	return &client{client: auther, base: uri, token: token}
}

// NewClientTokenTLS returns a client at the specified url that authenticates
// all outbound requests with the given token and tls.Config if provided.
func NewClientTokenTLS(uri, token string, c *tls.Config) (Client, error) {
	config := new(oauth2.Config)
	auther := config.Client(oauth2.NoContext, &oauth2.Token{AccessToken: token})
	if c != nil {
		if trans, ok := auther.Transport.(*oauth2.Transport); ok {
			if os.Getenv("SOCKS_PROXY") != "" {
				dialer, err := proxy.SOCKS5("tcp", os.Getenv("SOCKS_PROXY"), nil, proxy.Direct)
				if err != nil {
					return nil, err
				}
				trans.Base = &http.Transport{
					TLSClientConfig: c,
					Proxy:           http.ProxyFromEnvironment,
					Dial:            dialer.Dial,
				}
			} else {
				trans.Base = &http.Transport{
					TLSClientConfig: c,
					Proxy:           http.ProxyFromEnvironment,
				}
			}
		}
	}
	return &client{client: auther, base: uri, token: token}, nil
}

// Self returns the currently authenticated user.
func (c *client) Self() (*model.User, error) {
	out := new(model.User)
	uri := fmt.Sprintf(pathSelf, c.base)
	err := c.get(uri, out)
	return out, err
}

// User returns a user by login.
func (c *client) User(login string) (*model.User, error) {
	out := new(model.User)
	uri := fmt.Sprintf(pathUser, c.base, login)
	err := c.get(uri, out)
	return out, err
}

// UserList returns a list of all registered users.
func (c *client) UserList() ([]*model.User, error) {
	var out []*model.User
	uri := fmt.Sprintf(pathUsers, c.base)
	err := c.get(uri, &out)
	return out, err
}

// UserPost creates a new user account.
func (c *client) UserPost(in *model.User) (*model.User, error) {
	out := new(model.User)
	uri := fmt.Sprintf(pathUsers, c.base)
	err := c.post(uri, in, out)
	return out, err
}

// UserPatch updates a user account.
func (c *client) UserPatch(in *model.User) (*model.User, error) {
	out := new(model.User)
	uri := fmt.Sprintf(pathUser, c.base, in.Login)
	err := c.patch(uri, in, out)
	return out, err
}

// UserDel deletes a user account.
func (c *client) UserDel(login string) error {
	uri := fmt.Sprintf(pathUser, c.base, login)
	err := c.delete(uri)
	return err
}

// Repo returns a repository by name.
func (c *client) Repo(owner string, name string) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathRepo, c.base, owner, name)
	err := c.get(uri, out)
	return out, err
}

// RepoList returns a list of all repositories to which
// the user has explicit access in the host system.
func (c *client) RepoList() ([]*model.Repo, error) {
	var out []*model.Repo
	uri := fmt.Sprintf(pathRepos, c.base)
	err := c.get(uri, &out)
	return out, err
}

// RepoPost activates a repository.
func (c *client) RepoPost(owner string, name string) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathRepo, c.base, owner, name)
	err := c.post(uri, nil, out)
	return out, err
}

// RepoChow updates a repository owner.
func (c *client) RepoChown(owner string, name string) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathChown, c.base, owner, name)
	err := c.post(uri, nil, out)
	return out, err
}

// RepoPatch updates a repository.
func (c *client) RepoPatch(in *model.Repo) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathRepo, c.base, in.Owner, in.Name)
	err := c.patch(uri, in, out)
	return out, err
}

// RepoDel deletes a repository.
func (c *client) RepoDel(owner, name string) error {
	uri := fmt.Sprintf(pathRepo, c.base, owner, name)
	err := c.delete(uri)
	return err
}

// Build returns a repository build by number.
func (c *client) Build(owner, name string, num int) (*model.Build, error) {
	out := new(model.Build)
	uri := fmt.Sprintf(pathBuild, c.base, owner, name, num)
	err := c.get(uri, out)
	return out, err
}

// Build returns the latest repository build by branch.
func (c *client) BuildLast(owner, name, branch string) (*model.Build, error) {
	out := new(model.Build)
	uri := fmt.Sprintf(pathBuild, c.base, owner, name, "latest")
	if len(branch) != 0 {
		uri += "?branch=" + branch
	}
	err := c.get(uri, out)
	return out, err
}

// BuildList returns a list of recent builds for the
// the specified repository.
func (c *client) BuildList(owner, name string) ([]*model.Build, error) {
	var out []*model.Build
	uri := fmt.Sprintf(pathBuilds, c.base, owner, name)
	err := c.get(uri, &out)
	return out, err
}

// BuildQueue returns a list of enqueued builds.
func (c *client) BuildQueue() ([]*model.Feed, error) {
	var out []*model.Feed
	uri := fmt.Sprintf(pathBuildQueue, c.base)
	err := c.get(uri, &out)
	return out, err
}

// BuildStart re-starts a stopped build.
func (c *client) BuildStart(owner, name string, num int, params map[string]string) (*model.Build, error) {
	out := new(model.Build)
	val := parseToQueryParams(params)
	uri := fmt.Sprintf(pathBuild, c.base, owner, name, num)
	err := c.post(uri+"?"+val.Encode(), nil, out)
	return out, err
}

// BuildStop cancels the running job.
func (c *client) BuildStop(owner, name string, num, job int) error {
	uri := fmt.Sprintf(pathJob, c.base, owner, name, num, job)
	err := c.delete(uri)
	return err
}

// BuildFork re-starts a stopped build with a new build number,
// preserving the prior history.
func (c *client) BuildFork(owner, name string, num int, params map[string]string) (*model.Build, error) {
	out := new(model.Build)
	val := parseToQueryParams(params)
	val.Set("fork", "true")
	uri := fmt.Sprintf(pathBuild, c.base, owner, name, num)
	err := c.post(uri+"?"+val.Encode(), nil, out)
	return out, err
}

// BuildLogs returns the build logs for the specified job.
func (c *client) BuildLogs(owner, name string, num, job int) (io.ReadCloser, error) {
	uri := fmt.Sprintf(pathLog, c.base, owner, name, num, job)
	return stream(c.client, uri, "GET", nil, nil)
}

// Deploy triggers a deployment for an existing build using the
// specified target environment.
func (c *client) Deploy(owner, name string, num int, env string, params map[string]string) (*model.Build, error) {
	out := new(model.Build)
	val := parseToQueryParams(params)
	val.Set("fork", "true")
	val.Set("event", "deployment")
	val.Set("deploy_to", env)
	uri := fmt.Sprintf(pathBuild, c.base, owner, name, num)
	err := c.post(uri+"?"+val.Encode(), nil, out)
	return out, err
}

// SecretList returns a list of a repository secrets.
func (c *client) SecretList(owner, name string) ([]*model.Secret, error) {
	var out []*model.Secret
	uri := fmt.Sprintf(pathRepoSecrets, c.base, owner, name)
	err := c.get(uri, &out)
	return out, err
}

// SecretPost create or updates a repository secret.
func (c *client) SecretPost(owner, name string, secret *model.Secret) error {
	uri := fmt.Sprintf(pathRepoSecrets, c.base, owner, name)
	return c.post(uri, secret, nil)
}

// SecretDel deletes a named repository secret.
func (c *client) SecretDel(owner, name, secret string) error {
	uri := fmt.Sprintf(pathRepoSecret, c.base, owner, name, secret)
	return c.delete(uri)
}

// TeamSecretList returns a list of organizational secrets.
func (c *client) TeamSecretList(team string) ([]*model.Secret, error) {
	var out []*model.Secret
	uri := fmt.Sprintf(pathTeamSecrets, c.base, team)
	err := c.get(uri, &out)
	return out, err
}

// TeamSecretPost create or updates a organizational secret.
func (c *client) TeamSecretPost(team string, secret *model.Secret) error {
	uri := fmt.Sprintf(pathTeamSecrets, c.base, team)
	return c.post(uri, secret, nil)
}

// TeamSecretDel deletes a named orgainization secret.
func (c *client) TeamSecretDel(team, secret string) error {
	uri := fmt.Sprintf(pathTeamSecret, c.base, team, secret)
	return c.delete(uri)
}

// GlobalSecretList returns a list of global secrets.
func (c *client) GlobalSecretList() ([]*model.Secret, error) {
	var out []*model.Secret
	uri := fmt.Sprintf(pathGlobalSecrets, c.base)
	err := c.get(uri, &out)
	return out, err
}

// GlobalSecretPost create or updates a global secret.
func (c *client) GlobalSecretPost(secret *model.Secret) error {
	uri := fmt.Sprintf(pathGlobalSecrets, c.base)
	return c.post(uri, secret, nil)
}

// GlobalSecretDel deletes a named global secret.
func (c *client) GlobalSecretDel(secret string) error {
	uri := fmt.Sprintf(pathGlobalSecret, c.base, secret)
	return c.delete(uri)
}

// Sign returns a cryptographic signature for the input string.
func (c *client) Sign(owner, name string, in []byte) ([]byte, error) {
	uri := fmt.Sprintf(pathSign, c.base, owner, name)
	rc, err := stream(c.client, uri, "POST", in, nil)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return ioutil.ReadAll(rc)
}

// AgentList returns a list of build agents.
func (c *client) AgentList() ([]*model.Agent, error) {
	var out []*model.Agent
	uri := fmt.Sprintf(pathAgent, c.base)
	err := c.get(uri, &out)
	return out, err
}

//
// http request helper functions
//

// helper function for making an http GET request.
func (c *client) get(rawurl string, out interface{}) error {
	return c.do(rawurl, "GET", nil, out)
}

// helper function for making an http POST request.
func (c *client) post(rawurl string, in, out interface{}) error {
	return c.do(rawurl, "POST", in, out)
}

// helper function for making an http PUT request.
func (c *client) put(rawurl string, in, out interface{}) error {
	return c.do(rawurl, "PUT", in, out)
}

// helper function for making an http PATCH request.
func (c *client) patch(rawurl string, in, out interface{}) error {
	return c.do(rawurl, "PATCH", in, out)
}

// helper function for making an http DELETE request.
func (c *client) delete(rawurl string) error {
	return c.do(rawurl, "DELETE", nil, nil)
}

// helper function to make an http request
func (c *client) do(rawurl, method string, in, out interface{}) error {
	// executes the http request and returns the body as
	// and io.ReadCloser
	body, err := c.open(rawurl, method, in, out)
	if err != nil {
		return err
	}
	defer body.Close()

	// if a json response is expected, parse and return
	// the json response.
	if out != nil {
		return json.NewDecoder(body).Decode(out)
	}
	return nil
}

// helper function to open an http request
func (c *client) open(rawurl, method string, in, out interface{}) (io.ReadCloser, error) {
	uri, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	// creates a new http request to bitbucket.
	req, err := http.NewRequest(method, uri.String(), nil)
	if err != nil {
		return nil, err
	}

	// if we are posting or putting data, we need to
	// write it to the body of the request.
	if in != nil {
		rc, ok := in.(io.ReadCloser)
		if ok {
			req.Body = rc
			req.Header.Set("Content-Type", "plain/text")
		} else {
			inJson, err := json.Marshal(in)
			if err != nil {
				return nil, err
			}

			buf := bytes.NewBuffer(inJson)
			req.Body = ioutil.NopCloser(buf)

			req.ContentLength = int64(len(inJson))
			req.Header.Set("Content-Length", strconv.Itoa(len(inJson)))
			req.Header.Set("Content-Type", "application/json")
		}
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode > http.StatusPartialContent {
		defer resp.Body.Close()
		out, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("client error %d: %s", resp.StatusCode, string(out))
	}
	return resp.Body, nil
}

// createRequest is a helper function that builds an http.Request.
func (c *client) createRequest(rawurl, method string, in interface{}) (*http.Request, error) {
	uri, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}

	// if we are posting or putting data, we need to
	// write it to the body of the request.
	var buf io.ReadWriter
	if in != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(in)
		if err != nil {
			return nil, err
		}
	}

	// creates a new http request to bitbucket.
	req, err := http.NewRequest(method, uri.String(), buf)
	if err != nil {
		return nil, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// parseToQueryParams parses a map of strings and returns url.Values
func parseToQueryParams(p map[string]string) url.Values {
	values := url.Values{}
	for k, v := range p {
		values.Add(k, v)
	}
	return values
}
