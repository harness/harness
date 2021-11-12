// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package rpc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/drone/drone/operator/manager"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/oxtoacart/bpool"
)

var _ manager.BuildManager = (*Client)(nil)

var bufpool = bpool.NewBufferPool(64)

// Client defines an RPC client.
type Client struct {
	token  string
	server string
	client *retryablehttp.Client
}

// NewClient returns a new rpc client that is able to
// interact with a remote build controller using the
// http transport.
func NewClient(server, token string) *Client {
	client := retryablehttp.NewClient()
	client.RetryMax = 30
	client.RetryWaitMax = time.Second * 10
	client.RetryWaitMin = time.Second * 1
	client.Logger = nil
	return &Client{
		client: client,
		server: strings.TrimSuffix(server, "/"),
		token:  token,
	}
}

// SetDebug enabled debug-level logging within the retryable
// http.Client. This can be useful if you are debugging network
// connectivity issues and want to monitor disconnects,
// reconnects, and retries.
func (s *Client) SetDebug(debug bool) {
	if debug == true {
		s.client.Logger = log.New(os.Stderr, "", log.LstdFlags)
	} else {
		s.client.Logger = nil
	}
}

// Request requests the next available build stage for execution.
func (s *Client) Request(ctx context.Context, args *manager.Request) (*core.Stage, error) {
	timeout, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	in := &requestRequest{Request: args}
	out := &core.Stage{}
	err := s.send(timeout, "/rpc/v1/request", in, out)

	// The request is performing long polling and is subject
	// to a client-side and server-side timeout. The timeout
	// error is therefore expected behavior, and is not
	// considered an error by the system.
	if err == context.DeadlineExceeded {
		return nil, nil // no error
	}
	return out, err
}

// Accept accepts the build stage for execution.
func (s *Client) Accept(ctx context.Context, stage int64, machine string) (*core.Stage, error) {
	in := &acceptRequest{Stage: stage, Machine: machine}
	return nil, s.send(noContext, "/rpc/v1/accept", in, nil)
}

// Netrc returns a valid netrc for execution.
func (s *Client) Netrc(ctx context.Context, repo int64) (*core.Netrc, error) {
	in := &netrcRequest{repo}
	out := &core.Netrc{}
	err := s.send(noContext, "/rpc/v1/netrc", in, out)
	return out, err
}

// Details fetches build details
func (s *Client) Details(ctx context.Context, stage int64) (*manager.Context, error) {
	in := &detailsRequest{Stage: stage}
	out := &buildContextToken{}
	err := s.send(noContext, "/rpc/v1/details", in, out)
	if err != nil {
		return nil, err
	}
	// the repository token is excluded from the json encoding
	// by default. this workaround ensures it is available to
	// the remote build agent.
	out.Context.Repo.Secret = out.Secret
	return out.Context, nil
}

// Before signals the build step is about to start.
func (s *Client) Before(ctx context.Context, step *core.Step) error {
	in := &stepRequest{Step: step}
	out := &core.Step{}
	err := s.send(noContext, "/rpc/v1/before", in, out)
	if err != nil {
		return err
	}
	// the step ID and version (optimistic locking) are
	// updated when the step is created. Copy the updated
	// values back to the original step object.
	step.ID = out.ID
	step.Version = out.Version
	return err
}

// After signals the build step is complete.
func (s *Client) After(ctx context.Context, step *core.Step) error {
	in := &stepRequest{Step: step}
	out := &core.Step{}
	err := s.send(noContext, "/rpc/v1/after", in, out)
	if err != nil {
		return err
	}
	// the step version (optimistic locking) is updated
	// when the step is created. Copy the updated values
	// back to the original step object.
	step.Version = out.Version
	return err
}

// BeforeAll signals the build stage is about to start.
func (s *Client) BeforeAll(ctx context.Context, stage *core.Stage) error {
	in := &stageRequest{Stage: stage}
	out := &core.Stage{}
	err := s.send(noContext, "/rpc/v1/beforeAll", in, out)
	if err != nil {
		return err
	}
	stage.Version = out.Version
	stage.Updated = out.Updated
	stage.Created = out.Created
	// TODO(bradrydzewski) clean this code to prevent possible
	// index-out-of-bounds exceptions.
	for i, step := range stage.Steps {
		step.ID = out.Steps[i].ID
		step.Version = out.Steps[i].Version
	}
	return err
}

// AfterAll signals the build stage is complete.
func (s *Client) AfterAll(ctx context.Context, stage *core.Stage) error {
	in := &stageRequest{Stage: stage}
	out := &core.Stage{}
	err := s.send(noContext, "/rpc/v1/afterAll", in, out)
	if err != nil {
		return err
	}
	// the stage timestamps and version (optimistic locking)
	// are updated when the step is created. Copy the updated
	// values back to the original step object.
	stage.Version = out.Version
	stage.Updated = out.Updated
	stage.Created = out.Created
	return err
}

func (s *Client) Watch(ctx context.Context, build int64) (bool, error) {
	in := &watchRequest{build}
	out := &watchResponse{}
	err := s.send(ctx, "/rpc/v1/watch", in, out)
	return out.Done, err
}

func (s *Client) Write(ctx context.Context, step int64, line *core.Line) error {
	in := writePool.Get().(*writeRequest)
	in.Step = step
	in.Line = line
	err := s.send(noContext, "/rpc/v1/write", in, nil)
	writePool.Put(in)
	return err
}

func (s *Client) Upload(ctx context.Context, step int64, r io.Reader) error {
	endpoint := "/rpc/v1/upload?id=" + fmt.Sprint(step)
	return s.upload(noContext, endpoint, r)
}

func (s *Client) UploadBytes(ctx context.Context, step int64, data []byte) error {
	endpoint := "/rpc/v1/upload?id=" + fmt.Sprint(step)
	return s.upload(noContext, endpoint, data)
}

func (s *Client) UploadCard(ctx context.Context, step int64, input *core.CardInput) error {
	return errors.New("rpc upload card not supported")
}

func (s *Client) send(ctx context.Context, path string, in, out interface{}) error {
	// Source a buffer from a pool. The agent may generate a
	// large number of small requests for log entries. This will
	// help reduce pressure on the garbage collector.
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	err := json.NewEncoder(buf).Encode(in)
	if err != nil {
		return err
	}

	url := s.server + path
	req, err := retryablehttp.NewRequest("POST", url, buf)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Drone-Token", s.token)

	res, err := s.client.Do(req)
	if res != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return err
	}

	// Check the response for a 409 conflict. This indicates an
	// optimistic lock error, in which case multiple clients may
	// be attempting to update the same record. Convert this error
	// code to a proper error.
	if res.StatusCode == 409 {
		return db.ErrOptimisticLock
	}

	// Check the response for a 524 deadline exceeded. This is a
	// custom status code that indicates the server canceled the
	// request due to an internal polling timeout (this is normal).
	if res.StatusCode == 524 {
		return context.DeadlineExceeded
	}

	if res.StatusCode > 299 {
		body, _ := ioutil.ReadAll(res.Body)
		return &serverError{
			Status:  res.StatusCode,
			Message: string(body),
		}
	}

	// Check the response for a 204 no content. This indicates
	// the response body is empty and should be discarded.
	if res.StatusCode == 204 || out == nil {
		return nil
	}

	return json.NewDecoder(res.Body).Decode(out)
}

func (s *Client) upload(ctx context.Context, path string, body interface{}) error {
	url := s.server + path
	req, err := retryablehttp.NewRequest("POST", url, body)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)
	req.Header.Set("X-Drone-Token", s.token)

	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		body, _ := ioutil.ReadAll(res.Body)
		return &serverError{
			Status:  res.StatusCode,
			Message: string(body),
		}
	}
	return nil
}

// helper function returns true if the http.Request should be
// retried based on error and http status code. This function
// is used by the retryablehttp.Client.
func retryFunc(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if resp != nil {
		// Check the path to prevent retries when writing to the log
		// stream. This stream is temporary and ephemeral, and losing
		// log lines will not negatively impact the final persisted
		// log entries.
		if resp.Request.URL.Path == "/rpc/v1/write" {
			return false, err
		}
		// Check the response code. We retry on 500-range responses
		// to allow the server time to recover, as 500's are typically
		// not permanent errors and may relate to outages on the
		// server side.
		if resp.StatusCode >= 500 {
			return true, nil
		}
	}
	if err != nil {
		return true, err
	}
	return false, nil
}
