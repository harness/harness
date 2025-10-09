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

	"github.com/harness/gitness/app/api/request"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/version"
)

const (
	// HTTPRequestPathPreReceive is the subpath under the provided base url the client uses to call pre-receive.
	HTTPRequestPathPreReceive = "pre-receive"

	// HTTPRequestPathPostReceive is the subpath under the provided base url the client uses to call post-receive.
	HTTPRequestPathPostReceive = "post-receive"

	// HTTPRequestPathUpdate is the subpath under the provided base url the client uses to call update.
	HTTPRequestPathUpdate = "update"
)

// RestClientFactory creates clients that make rest api calls to Harness to execute githooks.
type RestClientFactory struct{}

func (f *RestClientFactory) NewClient(envVars map[string]string) (hook.Client, error) {
	payload, err := hook.LoadPayloadFromMap[Payload](envVars)
	if err != nil {
		return nil, fmt.Errorf("failed to load payload from provided map of environment variables: %w", err)
	}

	// ensure we return disabled message in case it's explicitly disabled
	if payload.Disabled {
		return hook.NewNoopClient([]string{"hook disabled"}), nil
	}

	if err := payload.Validate(); err != nil {
		return nil, fmt.Errorf("payload validation failed: %w", err)
	}

	return NewRestClient(payload), nil
}

// RestClient is the hook.Client used to call the githooks api of Harness api server.
type RestClient struct {
	httpClient *http.Client
	baseURL    string
	requestID  string
	baseInput  types.GithookInputBase
}

func NewRestClient(
	payload Payload,
) hook.Client {
	return &RestClient{
		httpClient: http.DefaultClient,
		baseURL:    strings.TrimRight(payload.BaseURL, "/"),
		requestID:  payload.RequestID,
		baseInput:  GetInputBaseFromPayload(payload),
	}
}

// PreReceive calls the pre-receive githook api of the Harness api server.
func (c *RestClient) PreReceive(
	ctx context.Context,
	in hook.PreReceiveInput,
) (hook.Output, error) {
	return c.githook(ctx, HTTPRequestPathPreReceive, types.GithookPreReceiveInput{
		GithookInputBase: c.baseInput,
		PreReceiveInput:  in,
	})
}

// Update calls the update githook api of the Harness api server.
func (c *RestClient) Update(
	ctx context.Context,
	in hook.UpdateInput,
) (hook.Output, error) {
	return c.githook(ctx, HTTPRequestPathUpdate, types.GithookUpdateInput{
		GithookInputBase: c.baseInput,
		UpdateInput:      in,
	})
}

// PostReceive calls the post-receive githook api of the Harness api server.
func (c *RestClient) PostReceive(
	ctx context.Context,
	in hook.PostReceiveInput,
) (hook.Output, error) {
	return c.githook(ctx, HTTPRequestPathPostReceive, types.GithookPostReceiveInput{
		GithookInputBase: c.baseInput,
		PostReceiveInput: in,
	})
}

// githook executes the requested githook type using the provided input.
func (c *RestClient) githook(ctx context.Context, githookType string, payload interface{}) (hook.Output, error) {
	uri := c.baseURL + "/" + githookType
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to serialize input: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return hook.Output{}, fmt.Errorf("failed to create new http request: %w", err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(request.HeaderUserAgent, fmt.Sprintf("Gitness/%s", version.Version))
	req.Header.Add(request.HeaderRequestID, c.requestID)

	// Execute the request
	resp, err := c.httpClient.Do(req)

	// ensure the body is closed after we read (independent of status code or error)
	if resp != nil && resp.Body != nil {
		// Use function to satisfy the linter which complains about unhandled errors otherwise
		defer func() { _ = resp.Body.Close() }()
	}

	if err != nil {
		return hook.Output{}, fmt.Errorf("request execution failed: %w", err)
	}

	return unmarshalResponse[hook.Output](resp)
}

// unmarshalResponse reads the response body and if there are no errors marshall's it into
// the data struct.
func unmarshalResponse[T any](resp *http.Response) (T, error) {
	var body T
	if resp == nil {
		return body, errors.New("http response is empty")
	}

	if resp.StatusCode == http.StatusNotFound {
		return body, hook.ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("expected response code 200 but got: %s", resp.Status)
	}

	// ensure we actually got a body returned.
	if resp.Body == nil {
		return body, errors.New("http response body is empty")
	}

	rawBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("error reading response body : %w", err)
	}

	err = json.Unmarshal(rawBody, &body)
	if err != nil {
		return body, fmt.Errorf("error deserializing response body: %w", err)
	}

	return body, nil
}
