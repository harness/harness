// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	// ciURL stores the rawURL that can be used to communicate with gitness from inside a
	// container.
	ciURL *url.URL

	// gitURL stores the URL the git endpoints are available at.
	// NOTE: we store it as url.URL so we can derive clone URLS without errors.
	gitURL *url.URL

	// harnessCodeApiUrl stores the URL for communicating with SaaS harness code.
	harnessCodeApiUrl *url.URL
}

func NewProvider(apiURLRaw string, apiURLInternalRaw, gitURLRaw, ciURLRaw string, harnessCodeApiUrlRaw string) (*Provider, error) {
	// remove trailing '/' to make usage easier
	apiURLRaw = strings.TrimRight(apiURLRaw, "/")
	apiURLInternalRaw = strings.TrimRight(apiURLInternalRaw, "/")
	gitURLRaw = strings.TrimRight(gitURLRaw, "/")
	ciURLRaw = strings.TrimRight(ciURLRaw, "/")

	// parse gitURL
	gitURL, err := url.Parse(gitURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided gitURLRaw '%s' is invalid: %w", gitURLRaw, err)
	}

	// parse ciURL
	ciURL, err := url.Parse(ciURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided ciURLRaw '%s' is invalid: %w", ciURLRaw, err)
	}

	harnessCodeApiUrlRaw = strings.TrimRight(harnessCodeApiUrlRaw, "/")
	harnessCodeApiUrl, err := url.Parse(harnessCodeApiUrlRaw)
	if err != nil {
		return nil, fmt.Errorf("provided harnessCodeAPIURLRaw '%s' is invalid: %w", harnessCodeAPIURLRaw, err)
	}

	return &Provider{
		apiURLRaw:         apiURLRaw,
		apiURLInternalRaw: apiURLInternalRaw,
		gitURL:            gitURL,
		ciURL:             ciURL,
		harnessCodeApiUrl: harnessCodeApiUrl,
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

// GenerateCICloneURL generates a URL that can be used by CI container builds to
// interact with gitness and clone a repo.
func (p *Provider) GenerateCICloneURL(repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return p.ciURL.JoinPath(repoPath).String()
}

func (p *Provider) GetHarnessCodeInternalUrl() string {
	return p.harnessCodeApiUrl.String()
}
