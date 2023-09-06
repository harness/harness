// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package url

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Provider provides the URLs of the gitness system.
type Provider struct {
	// apiURLRaw stores the raw URL the api endpoints are reachable at publicly.
	// NOTE: url is guaranteed to not have any trailing '/'.
	apiURLRaw string

	// apiURLInternalRaw stores the raw URL the api endpoints are reachable at internally
	// (no need for internal services to go via public route).
	// NOTE: url is guaranteed to not have any trailing '/'.
	apiURLInternalRaw string

	// gitURL stores the URL the git endpoints are available at.
	// NOTE: we store it as url.URL so we can derive clone URLS without errors.
	gitURL *url.URL
}

func NewProvider(apiURLRaw string, apiURLInternalRaw, gitURLRaw string) (*Provider, error) {
	// remove trailing '/' to make usage easier
	apiURLRaw = strings.TrimRight(apiURLRaw, "/")
	apiURLInternalRaw = strings.TrimRight(apiURLInternalRaw, "/")
	gitURLRaw = strings.TrimRight(gitURLRaw, "/")

	// parse gitURL
	gitURL, err := url.Parse(gitURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided gitURLRaw '%s' is invalid: %w", gitURLRaw, err)
	}

	return &Provider{
		apiURLRaw:         apiURLRaw,
		apiURLInternalRaw: apiURLInternalRaw,
		gitURL:            gitURL,
	}, nil
}

// GetAPIBaseURL returns the publicly reachable base url of the api server.
// NOTE: url is guaranteed to not have any trailing '/'.
func (p *Provider) GetAPIBaseURL() string {
	return p.apiURLRaw
}

// GetAPIBaseURLInternal returns the internally reachable base url of the api server.
// NOTE: url is guaranteed to not have any trailing '/'.
func (p *Provider) GetAPIBaseURLInternal() string {
	return p.apiURLInternalRaw
}

// GenerateRepoCloneURL generates the public git clone URL for the provided repo path.
// NOTE: url is guaranteed to not have any trailing '/'.
func (p *Provider) GenerateRepoCloneURL(repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return p.gitURL.JoinPath(repoPath).String()
}

// GenerateCustomRepoCloneURL generates a custom clone URL given the base URL.
// Example: url is http://host.docker.internal:3000 and repoPath is test/gitness-demo
// it will return http://host.docker.internal:3000/git/test/gitness-demo.git
func (p *Provider) GenerateCustomRepoCloneURL(baseURL *url.URL, repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return baseURL.JoinPath("git").JoinPath(repoPath).String()
}
