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
	// apiURLRaw stores the raw URL the api endpoints are available at.
	apiURLRaw string

	// gitURL stores the URL the git endpoints are available at.
	// NOTE: we store it as url.URL so we can derive clone URLS without errors.
	gitURL *url.URL
}

func NewProvider(apiURLRaw string, rawGitURL string) (*Provider, error) {
	gitURL, err := url.Parse(rawGitURL)
	if err != nil {
		return nil, fmt.Errorf("provided rawGitURL '%s' is invalid: %w", rawGitURL, err)
	}

	return &Provider{
		apiURLRaw: apiURLRaw,
		gitURL:    gitURL,
	}, nil
}

// GetAPIBaseURL returns the base url of the api server.
func (p *Provider) GetAPIBaseURL() string {
	return p.apiURLRaw
}

// GenerateRepoCloneURL generates the git clone URL for the provided repo path.
func (p *Provider) GenerateRepoCloneURL(repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return p.gitURL.JoinPath(repoPath).String()
}
