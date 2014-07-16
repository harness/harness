package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

var (
	ErrNotFound       = errors.New("Not Found")
	ErrForbidden      = errors.New("Forbidden")
	ErrBadRequest     = errors.New("Bad Request")
	ErrNotAuthorized  = errors.New("Unauthorized")
	ErrInternalServer = errors.New("Internal Server Error")
)

type Client struct {
	Token string
	URL   string
}

// Do submits an http.Request and parses the JSON-encoded http.Response,
// storing the result in the value pointed to by v.
func (c *Client) Do(method, path string, in, out interface{}) error {

	// create the URI
	uri, err := url.Parse(c.URL + path)
	if err != nil {
		return err
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "http"
	}

	if len(c.Token) > 0 {
		params := uri.Query()
		params.Add("access_token", c.Token)
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

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

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

	// Unmarshall the JSON response
	if out != nil {
		return json.Unmarshal(body, out)
	}

	return nil
}
