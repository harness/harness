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

// Package github wraps drone/go-scm for the linked-PR link-create flow.
// Per-event handlers do not use this client.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/drone/go-scm/scm"
	gh "github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

// Client wraps *scm.Client; we only use its http.Client so requests inherit
// whatever auth transport (PAT, OAuth2) the connector configured.
type Client struct {
	scm *scm.Client
}

func NewClient(c *scm.Client) *Client { return &Client{scm: c} }

// NewClientFromConnector builds a Client; apiURL is "" for github.com.
func NewClientFromConnector(apiURL, token string) (*Client, error) {
	var c *scm.Client
	var err error
	if apiURL == "" {
		c = gh.NewDefault()
	} else {
		c, err = gh.New(apiURL)
		if err != nil {
			return nil, fmt.Errorf("github: parse api url %q: %w", apiURL, err)
		}
	}
	if token != "" {
		c.Client = &http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.StaticTokenSource(&scm.Token{Token: token}),
				Scheme: oauth2.SchemeBearer,
			},
		}
	}
	return NewClient(c), nil
}

// Repository carries the GitHub metadata the link-create flow needs. We hit
// REST directly because go-scm doesn't expose node_id, our rename-proof key.
type Repository struct {
	NodeID string `json:"node_id"`
}

// GetRepository fetches /repos/{owner}/{repo} and returns the rename-proof
// node_id used to populate linked_repositories.linked_repo_provider_id.
//
// repoFullName must be "<owner>/<repo>".
func (c *Client) GetRepository(ctx context.Context, repoFullName string) (*Repository, error) {
	base := strings.TrimRight(c.scm.BaseURL.String(), "/")
	url := base + "/repos/" + repoFullName

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("github: build get-repo request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	httpClient := c.scm.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github: get repository %s: %w", repoFullName, err)
	}
	defer resp.Body.Close()

	// On non-2xx, snippet the body to aid diagnosis without OOMing on an
	// HTML error page from a reverse proxy.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("github: get repository %s: status %d: %s",
			repoFullName, resp.StatusCode, snippet)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MiB cap
	if err != nil {
		return nil, fmt.Errorf("github: read get-repo body for %s: %w", repoFullName, err)
	}

	var out Repository
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("github: decode get-repo body for %s: %w", repoFullName, err)
	}
	if out.NodeID == "" {
		return nil, fmt.Errorf("github: get repository %s: missing node_id in response", repoFullName)
	}
	return &out, nil
}
