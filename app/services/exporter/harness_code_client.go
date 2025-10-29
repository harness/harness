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

package exporter

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/types"
)

const (
	pathCreateRepo = "/v1/accounts/%s/orgs/%s/projects/%s/repos"
	pathDeleteRepo = "/v1/accounts/%s/orgs/%s/projects/%s/repos/%s"
	//nolint:gosec // wrong flagging
	headerAPIKey = "X-Api-Key"
	routingID    = "routingId"
)

var (
	errHTTPNotFound   = fmt.Errorf("not found")
	errHTTPBadRequest = fmt.Errorf("bad request")
	errHTTPInternal   = fmt.Errorf("internal error")
	errHTTPDuplicate  = fmt.Errorf("resource already exists")
)

type harnessCodeClient struct {
	client *client
}

type client struct {
	baseURL    string
	httpClient http.Client

	accountID string
	orgID     string
	projectID string

	token string
}

// newClient creates a new harness Client for interacting with the platforms APIs.
func newClient(baseURL string, accountID string, orgID string, projectID string, token string) (*client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseUrl required")
	}
	if accountID == "" {
		return nil, fmt.Errorf("accountID required")
	}
	if orgID == "" {
		return nil, fmt.Errorf("orgId required")
	}
	if projectID == "" {
		return nil, fmt.Errorf("projectId required")
	}
	if token == "" {
		return nil, fmt.Errorf("token required")
	}

	return &client{
		baseURL:   baseURL,
		accountID: accountID,
		orgID:     orgID,
		projectID: projectID,
		token:     token,
		httpClient: http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: false,
					MinVersion:         tls.VersionTLS12,
				},
			},
		},
	}, nil
}

func newHarnessCodeClient(
	baseURL string,
	accountID string,
	orgID string,
	projectID string,
	token string,
) (*harnessCodeClient, error) {
	client, err := newClient(baseURL, accountID, orgID, projectID, token)
	if err != nil {
		return nil, err
	}
	return &harnessCodeClient{
		client: client,
	}, nil
}

func (c *harnessCodeClient) CreateRepo(ctx context.Context, input repo.CreateInput) (*types.Repository, error) {
	path := fmt.Sprintf(pathCreateRepo, c.client.accountID, c.client.orgID, c.client.projectID)
	bodyBytes, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize body: %w", err)
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		appendPath(c.client.baseURL, path), bytes.NewBuffer(bodyBytes),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create new http request : %w", err)
	}

	q := map[string]string{routingID: c.client.accountID}
	addQueryParams(req, q)
	req.Header.Add("Content-Type", "application/json")
	req.ContentLength = int64(len(bodyBytes))

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}

	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}

	repository := new(types.Repository)
	err = mapStatusCodeToError(resp.StatusCode)
	if err != nil {
		return nil, err
	}

	err = unmarshalResponse(resp, repository)
	if err != nil {
		return nil, err
	}
	return repository, err
}

func addQueryParams(req *http.Request, params map[string]string) {
	if len(params) > 0 {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}
}

func (c *harnessCodeClient) DeleteRepo(ctx context.Context, repoIdentifier string) error {
	path := fmt.Sprintf(pathDeleteRepo, c.client.accountID, c.client.orgID, c.client.projectID, repoIdentifier)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendPath(c.client.baseURL, path), nil)
	if err != nil {
		return fmt.Errorf("unable to create new http request : %w", err)
	}

	q := map[string]string{routingID: c.client.accountID}
	addQueryParams(req, q)
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("request execution failed: %w", err)
	}

	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}
	return mapStatusCodeToError(resp.StatusCode)
}

func appendPath(uri string, path string) string {
	if path == "" {
		return uri
	}

	return strings.TrimRight(uri, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *client) Do(r *http.Request) (*http.Response, error) {
	addAuthHeader(r, c.token)
	return c.httpClient.Do(r)
}

// addAuthHeader adds the Authorization header to the request.
func addAuthHeader(req *http.Request, token string) {
	req.Header.Add(headerAPIKey, token)
}

func unmarshalResponse(resp *http.Response, data any) error {
	if resp == nil {
		return fmt.Errorf("http response is empty")
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body : %w", err)
	}
	err = json.Unmarshal(body, data)
	if err != nil {
		return fmt.Errorf("error deserializing response body : %w", err)
	}
	return nil
}

func mapStatusCodeToError(statusCode int) error {
	switch {
	case statusCode == 500:
		return errHTTPInternal
	case statusCode >= 500:
		return fmt.Errorf("received server side error status code %d", statusCode)
	case statusCode == 404:
		return errHTTPNotFound
	case statusCode == 400:
		return errHTTPBadRequest
	case statusCode == 409:
		return errHTTPDuplicate
	case statusCode >= 400:
		return fmt.Errorf("received client side error status code %d", statusCode)
	case statusCode >= 300:
		return fmt.Errorf("received further action required status code %d", statusCode)
	default:
		// TODO: definitely more things to consider here ...
		return nil
	}
}
