package github

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
	// Returned if the specified resource does not exist.
	ErrNotFound = errors.New("Not Found")

	// Returned if the caller attempts to make a call or modify a resource
	// for which the caller is not authorized.
	//
	// The request was a valid request, the caller's authentication credentials
	// succeeded but those credentials do not grant the caller permission to
	// access the resource.
	ErrForbidden = errors.New("Forbidden")

	// Returned if the call requires authentication and either the credentials
	// provided failed or no credentials were provided.
	ErrNotAuthorized = errors.New("Unauthorized")

	// Returned if the caller submits a badly formed request. For example,
	// the caller can receive this return if you forget a required parameter.
	ErrBadRequest = errors.New("Bad Request")
)

// DefaultClient uses DefaultTransport, and is used internall to execute
// all http.Requests. This may be overriden for unit testing purposes.
// 
// IMPORTANT: this is not thread safe and should not be touched with
// the exception overriding for mock unit testing.
var DefaultClient = http.DefaultClient

func (c *Client) do(method, path string, in, out interface{}) error {

	// create the URI
	uri, err := url.Parse(c.ApiUrl + path)
	if err != nil {
		return err
	}

	// add the access token to the URL query string
	// .. if using the Guest Client, this might be empty
	if len(c.Token)>0 {
		params := uri.Query()
		params.Add("access_token", c.Token)
		uri.RawQuery = params.Encode()
	}

	// create the request
	req := &http.Request{
		URL           : uri,
		Method        : method,
		ProtoMajor    : 1,
		ProtoMinor    : 1,
		Close         : true,
		ContentLength : 0,
	}

	// set the appropariate headers
	req.Header = http.Header{}
	req.Header.Set("Content-Type", "application/json")

	// workaround for the email api (see github documentation)
	if path == "/user/emails" && method == "GET" {
		req.Header.Set("Accept", "application/vnd.github.v3")
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
	}

	// make the request using the default http client
	resp, err := DefaultClient.Do(req)
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
	}

	// Unmarshall the JSON response
	if out != nil {
		return json.Unmarshal(body, out)
	}

	return nil
}


// TODO this is a helper function to get raw output. It can
//      be removed at some point in the future.
func (c *Client) raw(method, path string) ([]byte, error) {
	// create the URI
	uri, err := url.Parse(c.ApiUrl + path)
	if err != nil {
		return nil, err
	}

	// add the access token to the URL query string
	params := uri.Query()
	params.Add("access_token", c.Token)
	uri.RawQuery = params.Encode()

	// create the request
	req := &http.Request{
		URL           : uri,
		Method        : method,
		ProtoMajor    : 1,
		ProtoMinor    : 1,
		Close         : true,
		ContentLength : 0,
	}

	// set the appropariate headers
	req.Header = http.Header{}
	req.Header.Set("Content-Type", "application/json")

	// make the request using the default http client
	resp, err := DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Read the bytes from the body (make sure we defer close the body)
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}
