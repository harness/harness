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

package url

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Provider is an abstraction of a component that provides system related URLs.
// NOTE: Abstract to allow for custom implementation for more complex routing environments.
type Provider interface {
	// GetAPIBaseURLInternal returns the internally reachable base url of the api server.
	// NOTE: url is guaranteed to not have any trailing '/'.
	GetAPIBaseURLInternal() string

	// GenerateGITCloneURL generates the public git clone URL for the provided repo path.
	// NOTE: url is guaranteed to not have any trailing '/'.
	GenerateGITCloneURL(repoPath string) string

	// GenerateGITCloneURLContainer generates a URL that can be used by CI container builds to
	// interact with gitness and clone a repo.
	GenerateGITCloneURLContainer(repoPath string) string

	// GenerateUIPRURL returns the url for the UI screen of an existing pr.
	GenerateUIPRURL(repoPath string, prID int64) string

	// GenerateUICompareURL returns the url for the UI screen comparing two references.
	GenerateUICompareURL(repoPath string, ref1 string, ref2 string) string

	// GetAPIHostname returns the host for the api endpoint.
	GetAPIHostname() string

	// GetGITHostname returns the host for the git endpoint.
	GetGITHostname() string
}

// Provider provides the URLs of the gitness system.
type provider struct {
	// apiURL stores the raw URL the api endpoints are reachable at publicly.
	apiURL *url.URL

	// apiURLInternalRaw stores the raw URL the api endpoints are reachable at internally
	// (no need for internal services to go via public route).
	// NOTE: url is guaranteed to not have any trailing '/'.
	apiURLInternalRaw string

	// gitURL stores the URL the git endpoints are available at.
	// NOTE: we store it as url.URL so we can derive clone URLS without errors.
	gitURL *url.URL

	// gitURLContainer stores the rawURL that can be used to communicate with gitness from inside a
	// build container.
	gitURLContainer *url.URL

	// uiURL stores the raw URL to the ui endpoints.
	uiURL *url.URL
}

func NewProvider(
	apiURLRaw string,
	apiURLInternalRaw,
	gitURLRaw,
	gitURLContainerRaw string,
	uiURLRaw string,
) (Provider, error) {
	// remove trailing '/' to make usage easier
	apiURLRaw = strings.TrimRight(apiURLRaw, "/")
	apiURLInternalRaw = strings.TrimRight(apiURLInternalRaw, "/")
	gitURLRaw = strings.TrimRight(gitURLRaw, "/")
	gitURLContainerRaw = strings.TrimRight(gitURLContainerRaw, "/")
	uiURLRaw = strings.TrimRight(uiURLRaw, "/")

	// parseAPIURL
	apiURL, err := url.Parse(apiURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided apiURLRaw '%s' is invalid: %w", apiURLRaw, err)
	}

	// parse gitURL
	gitURL, err := url.Parse(gitURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided gitURLRaw '%s' is invalid: %w", gitURLRaw, err)
	}

	// parse gitURLContainer
	gitURLContainer, err := url.Parse(gitURLContainerRaw)
	if err != nil {
		return nil, fmt.Errorf("provided gitURLContainerRaw '%s' is invalid: %w", gitURLContainerRaw, err)
	}

	// parse uiURL
	uiURL, err := url.Parse(uiURLRaw)
	if err != nil {
		return nil, fmt.Errorf("provided uiURLRaw '%s' is invalid: %w", uiURLRaw, err)
	}

	return &provider{
		apiURL:            apiURL,
		apiURLInternalRaw: apiURLInternalRaw,
		gitURL:            gitURL,
		gitURLContainer:   gitURLContainer,
		uiURL:             uiURL,
	}, nil
}

func (p *provider) GetAPIBaseURLInternal() string {
	return p.apiURLInternalRaw
}

func (p *provider) GenerateGITCloneURL(repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return p.gitURL.JoinPath(repoPath).String()
}

func (p *provider) GenerateGITCloneURLContainer(repoPath string) string {
	repoPath = path.Clean(repoPath)
	if !strings.HasSuffix(repoPath, ".git") {
		repoPath += ".git"
	}

	return p.gitURLContainer.JoinPath(repoPath).String()
}

func (p *provider) GenerateUIPRURL(repoPath string, prID int64) string {
	return p.uiURL.JoinPath(repoPath, "pulls", fmt.Sprint(prID)).String()
}

func (p *provider) GenerateUICompareURL(repoPath string, ref1 string, ref2 string) string {
	return p.uiURL.JoinPath(repoPath, "pulls/compare", ref1+"..."+ref2).String()
}

func (p *provider) GetAPIHostname() string {
	return p.apiURL.Hostname()
}

func (p *provider) GetGITHostname() string {
	return p.gitURL.Hostname()
}
