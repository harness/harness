// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// go:build harness

package exporter

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/harness/gitness/internal/api/controller/repo"
	"io"
	"net/http"
	"strings"
)

const (
	pathCreateRepo      = "/v1/accounts/%s/orgs/%s/projects/%s/repos/%s?routingId=%s"
	pathDeleteRepo      = "/v1/accounts/%s/orgs/%s/projects/%s/repos/%s?routingId=%s"
	headerAuthorization = "X-Api-Key"
)

var (
	ErrNotFound = fmt.Errorf("not found")
)

type HarnessCodeClient struct {
	client *Client
}

type Client struct {
	baseURL    string
	httpClient http.Client

	accountId string
	orgId     string
	projectId string

	token string
}

// NewClient creates a new harness Client for interacting with the platforms APIs.
func NewClient(baseURL string, accountID string, orgId string, projectId string, token string) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseUrl required")
	}
	if accountID == "" {
		return nil, fmt.Errorf("accountID required")
	}
	if orgId == "" {
		return nil, fmt.Errorf("orgId required")
	}
	if projectId == "" {
		return nil, fmt.Errorf("projectId required")
	}
	if token == "" {
		return nil, fmt.Errorf("token required")
	}

	return &Client{
		baseURL:   baseURL,
		accountId: accountID,
		orgId:     orgId,
		projectId: projectId,
		token:     token,
		httpClient: http.Client{
			Transport: &http.Transport{
				// #nosec
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		},
	}, nil
}

func NewHarnessCodeClient(baseUrl string, accountID string, orgId string, projectId string, token string) (*HarnessCodeClient, error) {
	client, err := NewClient(baseUrl, accountID, orgId, projectId, token)
	if err != nil {
		return nil, err
	}
	return &HarnessCodeClient{
		client: client,
	}, nil
}

func (c *HarnessCodeClient) CreateRepo(ctx context.Context, input repo.CreateInput) (*Repository, error) {
	path := fmt.Sprintf(pathCreateRepo, c.client.accountId, c.client.orgId, c.client.projectId, input.UID, c.client.accountId)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, appendPath(c.client.baseURL, path), nil)

	if err != nil {
		return nil, fmt.Errorf("unable to create new http request : %w", err)
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("request execution failed: %w", err)
	}

	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}

	var repository Repository
	err = unmarshalResponse(resp, repository)

	if err != nil {
		return nil, err
	}
	return &repository, err
}

func (c *HarnessCodeClient) DeleteRepo(ctx context.Context, input repo.CreateInput) error {
	path := fmt.Sprintf(pathDeleteRepo, c.client.accountId, c.client.orgId, c.client.projectId, input.UID, c.client.accountId)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, appendPath(c.client.baseURL, path), nil)

	if err != nil {
		return fmt.Errorf("unable to create new http request : %w", err)
	}

	resp, err := c.client.Do(req)

	if err != nil {
		return fmt.Errorf("request execution failed: %w", err)
	}

	if resp != nil && resp.Body != nil {
		defer func() { _ = resp.Body.Close() }()
	}

	if resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return fmt.Errorf("recieved error with status code %d", resp.StatusCode)

}

func appendPath(uri string, path string) string {
	if path == "" {
		return uri
	}

	return strings.TrimRight(uri, "/") + "/" + strings.TrimLeft(path, "/")
}

func (c *Client) Do(r *http.Request) (*http.Response, error) {
	addAuthHeader(r, c.token)
	return c.httpClient.Do(r)
}

// addAuthHeader adds the Authorization header to the request.
func addAuthHeader(req *http.Request, token string) {
	req.Header.Add(headerAuthorization, token)
}

func unmarshalResponse(resp *http.Response, data interface{}) error {
	if resp == nil {
		return fmt.Errorf("http response is empty")
	}

	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("expected response code 200 but got: %s", resp.Status)
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
