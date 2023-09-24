// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// HTTPRequestPathPreReceive is the subpath under the provided base url the client uses to call pre-receive.
	HTTPRequestPathPreReceive = "pre-receive"

	// HTTPRequestPathPostReceive is the subpath under the provided base url the client uses to call post-receive.
	HTTPRequestPathPostReceive = "post-receive"

	// HTTPRequestPathUpdate is the subpath under the provided base url the client uses to call update.
	HTTPRequestPathUpdate = "update"
)

var (
	ErrNotFound = fmt.Errorf("not found")
)

// Client is the Client used to call the githooks api of gitness api server.
type Client struct {
	httpClient *http.Client

	// baseURL is the base url of the gitness api server.
	baseURL string

	// requestPreparation is used to prepare the request before sending.
	// This can be used to inject required headers.
	requestPreparation func(*http.Request) *http.Request
}

func NewClient(httpClient *http.Client, baseURL string, requestPreparation func(*http.Request) *http.Request) *Client {
	return &Client{
		httpClient:         httpClient,
		baseURL:            strings.TrimRight(baseURL, "/"),
		requestPreparation: requestPreparation,
	}
}

// PreReceive calls the pre-receive githook api of the gitness api server.
func (c *Client) PreReceive(ctx context.Context,
	in *PreReceiveInput) (*Output, error) {
	return c.githook(ctx, HTTPRequestPathPreReceive, in)
}

// Update calls the update githook api of the gitness api server.
func (c *Client) Update(ctx context.Context,
	in *UpdateInput) (*Output, error) {
	return c.githook(ctx, HTTPRequestPathUpdate, in)
}

// PostReceive calls the post-receive githook api of the gitness api server.
func (c *Client) PostReceive(ctx context.Context,
	in *PostReceiveInput) (*Output, error) {
	return c.githook(ctx, HTTPRequestPathPostReceive, in)
}

// githook executes the requested githook type using the provided input.
func (c *Client) githook(ctx context.Context, githookType string, in interface{}) (*Output, error) {
	uri := c.baseURL + "/" + githookType
	bodyBytes, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create new http request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")

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

	return unmarshalResponse[Output](resp)
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
