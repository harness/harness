// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"
)

var (
	ErrNotFound = fmt.Errorf("not found")
)

// client is the client used to call the githooks api of gitness api server.
type client struct {
	httpClient *http.Client

	// baseURL is the base url of the gitness api server.
	baseURL string

	// requestPreparation is used to prepare the request before sending.
	// This can be used to inject required headers.
	requestPreparation func(*http.Request) *http.Request
}

// PreReceive calls the pre-receive githook api of the gitness api server.
func (c *client) PreReceive(ctx context.Context,
	in *types.PreReceiveInput) (*types.ServerHookOutput, error) {
	return c.githook(ctx, "pre-receive", in)
}

// Update calls the update githook api of the gitness api server.
func (c *client) Update(ctx context.Context,
	in *types.UpdateInput) (*types.ServerHookOutput, error) {
	return c.githook(ctx, "update", in)
}

// PostReceive calls the post-receive githook api of the gitness api server.
func (c *client) PostReceive(ctx context.Context,
	in *types.PostReceiveInput) (*types.ServerHookOutput, error) {
	return c.githook(ctx, "post-receive", in)
}

// githook executes the requested githook type using the provided input.
func (c *client) githook(ctx context.Context, githookType string, in interface{}) (*types.ServerHookOutput, error) {
	uri := fmt.Sprintf("%s/v1/internal/git-hooks/%s", c.baseURL, githookType)
	bodyBytes, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create new http request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", fmt.Sprintf("Gitness/%s", version.Version)) //TODO: change once it's separate CLI.

	// prepare request if configured
	if c.requestPreparation != nil {
		req = c.requestPreparation(req)
	}

	// Execute the request
	resp, err := c.httpClient.Do(req)

	// ensure the body is closed after we read (independent of status code or error)
	if resp != nil && resp.Body != nil {
		// Use function to satisfy the linter which complains about unhandled errors otherwise
		defer func() { _ = resp.Body.Close() }()
	}

	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}

	return unmarshalResponse[types.ServerHookOutput](resp)
}

// unmarshalResponse reads the response body and if there are no errors marshall's it into
// the data struct.
func unmarshalResponse[T any](resp *http.Response) (*T, error) {
	if resp == nil {
		return nil, errors.New("http response is empty")
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("expected response code 200 but got: %s", resp.Status)
	}

	// ensure we actually got a body returned.
	if resp.Body == nil {
		return nil, errors.New("http response body is empty")
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body : %w", err)
	}

	body := new(T)
	err = json.Unmarshal(rawBody, body)
	if err != nil {
		return nil, fmt.Errorf("error deserializing response body: %w", err)
	}

	return body, nil
}
