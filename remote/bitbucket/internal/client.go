package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/bitbucket"
)

const (
	get  = "GET"
	put  = "PUT"
	post = "POST"
	del  = "DELETE"
)

const (
	pathUser   = "%s/2.0/user/"
	pathEmails = "%s/2.0/user/emails"
	pathTeams  = "%s/2.0/teams/?%s"
	pathRepo   = "%s/2.0/repositories/%s/%s"
	pathRepos  = "%s/2.0/repositories/%s?%s"
	pathHook   = "%s/2.0/repositories/%s/%s/hooks/%s"
	pathHooks  = "%s/2.0/repositories/%s/%s/hooks?%s"
	pathSource = "%s/1.0/repositories/%s/%s/src/%s/%s"
	pathStatus = "%s/2.0/repositories/%s/%s/commit/%s/statuses/build"
)

type Client struct {
	*http.Client
	base string
}

func NewClient(url string, client *http.Client) *Client {
	return &Client{client, url}
}

func NewClientToken(url, client, secret string, token *oauth2.Token) *Client {
	config := &oauth2.Config{
		ClientID:     client,
		ClientSecret: secret,
		Endpoint:     bitbucket.Endpoint,
	}
	return NewClient(url, config.Client(oauth2.NoContext, token))
}

func (c *Client) FindCurrent() (*Account, error) {
	out := new(Account)
	uri := fmt.Sprintf(pathUser, c.base)
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListEmail() (*EmailResp, error) {
	out := new(EmailResp)
	uri := fmt.Sprintf(pathEmails, c.base)
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListTeams(opts *ListTeamOpts) (*AccountResp, error) {
	out := new(AccountResp)
	uri := fmt.Sprintf(pathTeams, c.base, opts.Encode())
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) FindRepo(owner, name string) (*Repo, error) {
	out := new(Repo)
	uri := fmt.Sprintf(pathRepo, c.base, owner, name)
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListRepos(account string, opts *ListOpts) (*RepoResp, error) {
	out := new(RepoResp)
	uri := fmt.Sprintf(pathRepos, c.base, account, opts.Encode())
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListReposAll(account string) ([]*Repo, error) {
	var page = 1
	var repos []*Repo

	for {
		resp, err := c.ListRepos(account, &ListOpts{Page: page, PageLen: 100})
		if err != nil {
			return repos, err
		}
		repos = append(repos, resp.Values...)
		if len(resp.Next) == 0 {
			break
		}
		page = resp.Page + 1
	}
	return repos, nil
}

func (c *Client) FindHook(owner, name, id string) (*Hook, error) {
	out := new(Hook)
	uri := fmt.Sprintf(pathHook, c.base, owner, name, id)
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListHooks(owner, name string, opts *ListOpts) (*HookResp, error) {
	out := new(HookResp)
	uri := fmt.Sprintf(pathHooks, c.base, owner, name, opts.Encode())
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) CreateHook(owner, name string, hook *Hook) error {
	uri := fmt.Sprintf(pathHooks, c.base, owner, name, "")
	return c.do(uri, post, hook, nil)
}

func (c *Client) DeleteHook(owner, name, id string) error {
	uri := fmt.Sprintf(pathHook, c.base, owner, name, id)
	return c.do(uri, del, nil, nil)
}

func (c *Client) FindSource(owner, name, revision, path string) (*Source, error) {
	out := new(Source)
	uri := fmt.Sprintf(pathSource, c.base, owner, name, revision, path)
	err := c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) CreateStatus(owner, name, revision string, status *BuildStatus) error {
	uri := fmt.Sprintf(pathStatus, c.base, owner, name, revision)
	return c.do(uri, post, status, nil)
}

func (c *Client) do(rawurl, method string, in, out interface{}) error {

	uri, err := url.Parse(rawurl)
	if err != nil {
		return err
	}

	// if we are posting or putting data, we need to
	// write it to the body of the request.
	var buf io.ReadWriter
	if in != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(in)
		if err != nil {
			return err
		}
	}

	// creates a new http request to bitbucket.
	req, err := http.NewRequest(method, uri.String(), buf)
	if err != nil {
		return err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// if an error is encountered, parse and return the
	// error response.
	if resp.StatusCode > http.StatusPartialContent {
		err := Error{}
		json.NewDecoder(resp.Body).Decode(&err)
		err.Status = resp.StatusCode
		return err
	}

	// if a json response is expected, parse and return
	// the json response.
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}
