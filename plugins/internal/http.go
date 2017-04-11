package internal

import (
	"bytes"
	"encoding/json"
	"errors"
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
		out, _ := ioutil.ReadAll(resp.Body)
		return errors.New(string(out))
	}

	// if a json response is expected, parse and return
	// the json response.
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}

	return nil
}
