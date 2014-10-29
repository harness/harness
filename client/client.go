package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

type Client struct {
	token string
	url   string

	Commits *CommitService
	Repos   *RepoService
	Users   *UserService
}

func New(token, url string) *Client {
	c := Client{
		token: token,
		url:   url,
	}

	c.Commits = &CommitService{&c}
	c.Repos = &RepoService{&c}
	c.Users = &UserService{&c}
	return &c
}

var (
	ErrNotFound       = errors.New("Not Found")
	ErrForbidden      = errors.New("Forbidden")
	ErrBadRequest     = errors.New("Bad Request")
	ErrNotAuthorized  = errors.New("Unauthorized")
	ErrInternalServer = errors.New("Internal Server Error")
)

// runs an http.Request and parses the JSON-encoded http.Response,
// storing the result in the value pointed to by v.
func (c *Client) run(method, path string, in, out interface{}) error {

	// create the URI
	uri, err := url.Parse(c.url + path)
	if err != nil {
		return err
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "http"
	}

	if len(c.token) > 0 {
		params := uri.Query()
		params.Add("access_token", c.token)
		uri.RawQuery = params.Encode()
	}

	// create the request
	req := &http.Request{
		URL:           uri,
		Method:        method,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Close:         true,
		ContentLength: 0,
	}

	// if data input is provided, serialize to JSON
	if in != nil {
		inJson, err := json.Marshal(in)
		if err != nil {
			return err
		}

		buf := bytes.NewBuffer(inJson)
		req.Body = ioutil.NopCloser(buf)

		req.ContentLength = int64(len(inJson))
		req.Header.Set("Content-Length", strconv.Itoa(len(inJson)))
		req.Header.Set("Content-Type", "application/json")
	}

	// make the request using the default http client
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	// make sure we defer close the body
	defer resp.Body.Close()

	// Check for an http error status (ie not 200 StatusOK)
	switch resp.StatusCode {
	case 404:
		return ErrNotFound
	case 403:
		return ErrForbidden
	case 401:
		return ErrNotAuthorized
	case 400:
		return ErrBadRequest
	case 500:
		return ErrInternalServer
	}

	// Decode the JSON response
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}

// do makes an http.Request and returns the response
func (c *Client) do(method, path string) (*http.Response, error) {

	// create the URI
	uri, err := url.Parse(c.url + path)
	if err != nil {
		return nil, err
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "http"
	}

	if len(c.token) > 0 {
		params := uri.Query()
		params.Add("access_token", c.token)
		uri.RawQuery = params.Encode()
	}

	// create the request
	req := &http.Request{
		URL:           uri,
		Method:        method,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Close:         true,
		ContentLength: 0,
	}

	// make the request using the default http client
	return http.DefaultClient.Do(req)
}
