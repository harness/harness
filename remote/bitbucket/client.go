package bitbucket

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

const api = "https://api.bitbucket.org"

type Client struct {
	*http.Client
}

func NewClient(client *http.Client) *Client {
	return &Client{client}
}

func NewClientToken(client, secret string, token *oauth2.Token) *Client {
	config := &oauth2.Config{
		ClientID:     client,
		ClientSecret: secret,
		Endpoint:     bitbucket.Endpoint,
	}
	return NewClient(config.Client(oauth2.NoContext, token))
}

func (c *Client) FindCurrent() (*Account, error) {
	var out = new(Account)
	var uri = fmt.Sprintf("%s/2.0/user/", api)
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListEmail() (*EmailResp, error) {
	var out = new(EmailResp)
	var uri = fmt.Sprintf("%s/2.0/user/emails", api)
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListTeams(opts *ListOpts) (*AccountResp, error) {
	var out = new(AccountResp)
	var uri = fmt.Sprintf("%s/2.0/teams/?role=member", api)
	if opts != nil && opts.Page > 0 {
		uri = fmt.Sprintf("%s&page=%d", uri, opts.Page)
	}
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) FindRepo(owner, name string) (*Repo, error) {
	var out = new(Repo)
	var uri = fmt.Sprintf("%s/2.0/repositories/%s/%s", api, owner, name)
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListRepos(account string, opts *ListOpts) (*RepoResp, error) {
	var out = new(RepoResp)
	var uri = fmt.Sprintf("%s/2.0/repositories/%s", api)
	if opts != nil && opts.Page > 0 {
		uri = fmt.Sprintf("%s?page=%d", uri, opts.Page)
	}
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) FindHook(owner, name, id string) (*Hook, error) {
	var out = new(Hook)
	var uri = fmt.Sprintf("%s/2.0/repositories/%s/%s/hooks/%s", api, owner, name, id)
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) ListHooks(owner, name string, opts *ListOpts) (*HookResp, error) {
	var out = new(HookResp)
	var uri = fmt.Sprintf("%s/2.0/repositories/%s/%s/hooks", api, owner, name)
	if opts != nil && opts.Page > 0 {
		uri = fmt.Sprintf("%s?page=%d", uri, opts.Page)
	}
	var err = c.do(uri, get, nil, out)
	return out, err
}

func (c *Client) CreateHook(owner, name, hook *Hook) error {
	var uri = fmt.Sprintf("%s/2.0/repositories/%s/%s/hooks", api, owner, name)
	return c.do(uri, post, hook, nil)
}

func (c *Client) DeleteHook(owner, name, id string) error {
	var uri = fmt.Sprintf("%s/2.0/repositories/%s/%s/hooks/%s", api, owner, name, id)
	return c.do(uri, del, nil, nil)
}

func (c *Client) do(rawurl, method string, in, out interface{}) error {

	uri, err := url.Parse(rawurl)
	if err != nil {
		return err
	}
	println(uri.String())
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
