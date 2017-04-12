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

	pathSelf           = "%s/api/user"
	pathFeed           = "%s/api/user/feed"
	pathRepos          = "%s/api/user/repos"
	pathRepo           = "%s/api/repos/%s/%s"
	pathChown          = "%s/api/repos/%s/%s/chown"
	pathRepair         = "%s/api/repos/%s/%s/repair"
	pathBuilds         = "%s/api/repos/%s/%s/builds"
	pathBuild          = "%s/api/repos/%s/%s/builds/%v"
	pathApprove        = "%s/api/repos/%s/%s/builds/%d/approve"
	pathDecline        = "%s/api/repos/%s/%s/builds/%d/decline"
	pathJob            = "%s/api/repos/%s/%s/builds/%d/%d"
	pathLog            = "%s/api/repos/%s/%s/logs/%d/%d"
	pathRepoSecrets    = "%s/api/repos/%s/%s/secrets"
	pathRepoSecret     = "%s/api/repos/%s/%s/secrets/%s"
	pathRepoRegistries = "%s/api/repos/%s/%s/registry"
	pathRepoRegistry   = "%s/api/repos/%s/%s/registry/%s"
	pathUsers          = "%s/api/users"
	pathUser           = "%s/api/users/%s"
	pathBuildQueue     = "%s/api/builds"
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

// RepoChown updates a repository owner.
func (c *client) RepoChown(owner string, name string) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathChown, c.base, owner, name)
	err := c.post(uri, nil, out)
	return out, err
}

// RepoRepair repais the repository hooks.
func (c *client) RepoRepair(owner string, name string) error {
	uri := fmt.Sprintf(pathRepair, c.base, owner, name)
	return c.post(uri, nil, nil)
}

// RepoPatch updates a repository.
func (c *client) RepoPatch(owner, name string, in *model.RepoPatch) (*model.Repo, error) {
	out := new(model.Repo)
	uri := fmt.Sprintf(pathRepo, c.base, owner, name)
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

// BuildApprove approves a blocked build.
func (c *client) BuildApprove(owner, name string, num int) (*model.Build, error) {
	out := new(model.Build)
	uri := fmt.Sprintf(pathApprove, c.base, owner, name, num)
	err := c.post(uri, nil, out)
	return out, err
}

// BuildDecline declines a blocked build.
func (c *client) BuildDecline(owner, name string, num int) (*model.Build, error) {
	out := new(model.Build)
	uri := fmt.Sprintf(pathDecline, c.base, owner, name, num)
	err := c.post(uri, nil, out)
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

// Registry returns a registry by hostname.
func (c *client) Registry(owner, name, hostname string) (*model.Registry, error) {
	out := new(model.Registry)
	uri := fmt.Sprintf(pathRepoRegistry, c.base, owner, name, hostname)
	err := c.get(uri, out)
	return out, err
}

// RegistryList returns a list of all repository registries.
func (c *client) RegistryList(owner string, name string) ([]*model.Registry, error) {
	var out []*model.Registry
	uri := fmt.Sprintf(pathRepoRegistries, c.base, owner, name)
	err := c.get(uri, &out)
	return out, err
}

// RegistryCreate creates a registry.
func (c *client) RegistryCreate(owner, name string, in *model.Registry) (*model.Registry, error) {
	out := new(model.Registry)
	uri := fmt.Sprintf(pathRepoRegistries, c.base, owner, name)
	err := c.post(uri, in, out)
	return out, err
}

// RegistryUpdate updates a registry.
func (c *client) RegistryUpdate(owner, name string, in *model.Registry) (*model.Registry, error) {
	out := new(model.Registry)
	uri := fmt.Sprintf(pathRepoRegistry, c.base, owner, name, in.Address)
	err := c.patch(uri, in, out)
	return out, err
}

// RegistryDelete deletes a registry.
func (c *client) RegistryDelete(owner, name, hostname string) error {
	uri := fmt.Sprintf(pathRepoRegistry, c.base, owner, name, hostname)
	return c.delete(uri)
}

// Secret returns a secret by name.
func (c *client) Secret(owner, name, secret string) (*model.Secret, error) {
	out := new(model.Secret)
	uri := fmt.Sprintf(pathRepoSecret, c.base, owner, name, secret)
	err := c.get(uri, out)
	return out, err
}

// SecretList returns a list of all repository secrets.
func (c *client) SecretList(owner string, name string) ([]*model.Secret, error) {
	var out []*model.Secret
	uri := fmt.Sprintf(pathRepoSecrets, c.base, owner, name)
	err := c.get(uri, &out)
	return out, err
}

// SecretCreate creates a secret.
func (c *client) SecretCreate(owner, name string, in *model.Secret) (*model.Secret, error) {
	out := new(model.Secret)
	uri := fmt.Sprintf(pathRepoSecrets, c.base, owner, name)
	err := c.post(uri, in, out)
	return out, err
}

// SecretUpdate updates a secret.
func (c *client) SecretUpdate(owner, name string, in *model.Secret) (*model.Secret, error) {
	out := new(model.Secret)
	uri := fmt.Sprintf(pathRepoSecret, c.base, owner, name, in.Name)
	err := c.patch(uri, in, out)
	return out, err
}

// SecretDelete deletes a secret.
func (c *client) SecretDelete(owner, name, secret string) error {
	uri := fmt.Sprintf(pathRepoSecret, c.base, owner, name, secret)
	return c.delete(uri)
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
