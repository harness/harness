package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	logs "github.com/Sirupsen/logrus"
	common "github.com/drone/drone/pkg/types"
)

type updater struct{}

func (u *updater) SetCommit(user *common.User, r *common.Repo, c *common.Commit) error {
	path := fmt.Sprintf("/api/queue/push/%s", r.FullName)
	return sendBackoff("POST", path, c, nil)
}

func (u *updater) SetBuild(r *common.Repo, c *common.Commit, b *common.Build) error {
	path := fmt.Sprintf("/api/queue/push/%s/%v", r.FullName, c.Sequence)
	return sendBackoff("POST", path, b, nil)
}

func (u *updater) SetLogs(r *common.Repo, c *common.Commit, b *common.Build, rc io.ReadCloser) error {
	path := fmt.Sprintf("/api/queue/push/%s/%v/%v", r.FullName, c.Sequence, b.Sequence)
	return sendBackoff("POST", path, rc, nil)
}

func sendBackoff(method, path string, in, out interface{}) error {
	var err error
	var attempts int
	for {
		err = send(method, path, in, out)
		if err == nil {
			break
		}
		if attempts > 99 {
			break
		}
		attempts++
		time.Sleep(time.Second * 30)
	}
	return err
}

// do makes an http.Request and returns the response
func send(method, path string, in, out interface{}) error {

	// create the URI
	uri, err := url.Parse(addr + path)
	if err != nil {
		return err
	}

	if len(uri.Scheme) == 0 {
		uri.Scheme = "http"
	}

	params := uri.Query()
	params.Add("token", token)
	uri.RawQuery = params.Encode()

	// create the request
	req, err := http.NewRequest(method, uri.String(), nil)
	if err != nil {
		return err
	}
	req.ProtoAtLeast(1, 1)
	req.Close = true
	req.ContentLength = 0

	// If the data is a readCloser we can attach directly
	// to the request body.
	//
	// Else we serialize the data input as JSON.
	if rc, ok := in.(io.ReadCloser); ok {
		req.Body = rc

	} else if in != nil {
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
		logs.Errorf("Error posting request. %s", err)
		return err
	}
	defer resp.Body.Close()

	// Check for an http error status (ie not 200 StatusOK)
	if resp.StatusCode > 300 {
		logs.Errorf("Error status code %d", resp.StatusCode)
		return fmt.Errorf(resp.Status)
	}

	// Decode the JSON response
	if out != nil {
		err = json.NewDecoder(resp.Body).Decode(out)
	}

	return err
}
