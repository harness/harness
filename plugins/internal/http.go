package internal

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
)

// Send makes an http request to the given endpoint, writing the input
// to the request body and unmarshaling the output from the response body.
func Send(method, path string, in, out interface{}) error {
	uri, err := url.Parse(path)
	if err != nil {
		return err
	}

	// if we are posting or putting data, we need to
	// write it to the body of the request.
	var buf io.ReadWriter
	if in != nil {
		buf = new(bytes.Buffer)
		jsonerr := json.NewEncoder(buf).Encode(in)
		if jsonerr != nil {
			return jsonerr
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

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// if an error is encountered, parse and return the
	// error response.
	if resp.StatusCode > http.StatusPartialContent {
		out, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return &Error{
			code: resp.StatusCode,
			text: string(out),
		}
	}

	// if a json response is expected, parse and return
	// the json response.
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}

// Error represents a http error.
type Error struct {
	code int
	text string
}

// Code returns the http error code.
func (e *Error) Code() int {
	return e.code
}

// Error returns the error message in string format.
func (e *Error) Error() string {
	return e.text
}
