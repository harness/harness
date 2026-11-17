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

package importer

import (
	"context"
	"crypto/sha512"
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/tracing"
	"github.com/harness/gitness/types"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/azure"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/gitea"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/gogs"
	"github.com/drone/go-scm/scm/driver/harness"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/drone/go-scm/scm/transport"
	"github.com/drone/go-scm/scm/transport/oauth2"
	"github.com/rs/zerolog/log"
)

type ProviderType string

const (
	ProviderTypeGitHub    ProviderType = "github"
	ProviderTypeGitLab    ProviderType = "gitlab"
	ProviderTypeBitbucket ProviderType = "bitbucket"
	ProviderTypeStash     ProviderType = "stash"
	ProviderTypeGitea     ProviderType = "gitea"
	ProviderTypeGogs      ProviderType = "gogs"
	ProviderTypeAzure     ProviderType = "azure"
	ProviderTypeHarness   ProviderType = "harness"
)

func (p ProviderType) Enum() []any {
	return []any{
		ProviderTypeGitHub,
		ProviderTypeGitLab,
		ProviderTypeBitbucket,
		ProviderTypeStash,
		ProviderTypeGitea,
		ProviderTypeGogs,
		ProviderTypeAzure,
		ProviderTypeHarness,
	}
}

type Provider struct {
	Type     ProviderType `json:"type"`
	Host     string       `json:"host"`
	Username string       `json:"username"`
	Password string       `json:"password"`
}

type RepositoryInfo struct {
	Space         string
	Identifier    string
	CloneURL      string
	IsPublic      bool
	DefaultBranch string
}

// ToRepo converts the RepositoryInfo into the types.Repository object marked as being imported and is-public flag.
func (r *RepositoryInfo) ToRepo(
	spaceID int64,
	spacePath string,
	identifier string,
	description string,
	principal *types.Principal,
) (*types.Repository, bool) {
	return NewRepo(
		spaceID,
		spacePath,
		identifier,
		description,
		principal,
		r.DefaultBranch,
	), r.IsPublic
}

func hash(s string) string {
	h := sha512.New()
	_, _ = h.Write([]byte(s))
	return base32.StdEncoding.EncodeToString(h.Sum(nil)[:10])
}

func oauthTransport(base http.RoundTripper, token string, scheme string) http.RoundTripper {
	if token == "" {
		return base
	}
	return &oauth2.Transport{
		Base:   base,
		Scheme: scheme,
		Source: oauth2.StaticTokenSource(&scm.Token{Token: token}),
	}
}

func authHeaderTransport(base http.RoundTripper, token string) http.RoundTripper {
	if token == "" {
		return base
	}
	return &transport.Authorization{
		Base:        base,
		Scheme:      "token",
		Credentials: token,
	}
}

func basicAuthTransport(base http.RoundTripper, username, password string) http.RoundTripper {
	if username == "" && password == "" {
		return base
	}
	return &transport.BasicAuth{
		Base:     base,
		Username: username,
		Password: password,
	}
}

func apiKeyTransport(base http.RoundTripper, apiKey string) http.RoundTripper {
	if apiKey == "" {
		return base
	}
	return &transport.Custom{
		Base: base,
		Before: func(r *http.Request) {
			r.Header.Set("x-api-key", apiKey)
		},
	}
}

// getScmClientWithTransport creates an SCM client along with the necessary transport
// layer depending on the provider. For example, for bitbucket we support app passwords
// so the auth transport is BasicAuth whereas it's Oauth for other providers.
// It validates that auth credentials are provided if authReq is true.
//
//nolint:gocognit
func (r *Importer) getScmClientWithTransport(
	provider Provider,
	slug string,
	authReq bool,
) (*scm.Client, error) {
	if authReq && (provider.Username == "" || provider.Password == "") {
		return nil, usererror.BadRequest("SCM provider authentication credentials missing")
	}
	// an empty host means the public endpoint of the provider is used.
	if provider.Host != "" {
		if err := r.validateProviderHost(provider.Host); err != nil {
			return nil, err
		}
	}
	var c *scm.Client
	var err error
	var transport http.RoundTripper
	switch provider.Type {
	case "":
		return nil, errors.New("scm provider can not be empty")

	case ProviderTypeGitHub:
		if provider.Host != "" {
			c, err = github.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = github.NewDefault()
		}
		transport = oauthTransport(r.baseTransport, provider.Password, oauth2.SchemeBearer)

	case ProviderTypeGitLab:
		if provider.Host != "" {
			c, err = gitlab.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = gitlab.NewDefault()
		}
		transport = oauthTransport(r.baseTransport, provider.Password, oauth2.SchemeBearer)

	case ProviderTypeBitbucket:
		if provider.Host != "" {
			c, err = bitbucket.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = bitbucket.NewDefault()
		}
		transport = basicAuthTransport(r.baseTransport, provider.Username, provider.Password)

	case ProviderTypeStash:
		if provider.Host != "" {
			c, err = stash.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = stash.NewDefault()
		}
		transport = oauthTransport(r.baseTransport, provider.Password, oauth2.SchemeBearer)

	case ProviderTypeGitea:
		if provider.Host == "" {
			return nil, errors.New("scm provider Host missing")
		}
		c, err = gitea.New(provider.Host)
		if err != nil {
			return nil, fmt.Errorf("scm provider Host invalid: %w", err)
		}
		transport = authHeaderTransport(r.baseTransport, provider.Password)

	case ProviderTypeGogs:
		if provider.Host == "" {
			return nil, errors.New("scm provider Host missing")
		}
		c, err = gogs.New(provider.Host)
		if err != nil {
			return nil, fmt.Errorf("scm provider Host invalid: %w", err)
		}
		transport = oauthTransport(r.baseTransport, provider.Password, oauth2.SchemeToken)

	case ProviderTypeAzure:
		org, project, err := extractOrgAndProjectFromSlug(slug)
		if err != nil {
			return nil, fmt.Errorf("invalid slug format: %w", err)
		}
		if provider.Host != "" {
			c, err = azure.New(provider.Host, org, project)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = azure.NewDefault(org, project)
		}
		transport = basicAuthTransport(r.baseTransport, provider.Username, provider.Password)

	case ProviderTypeHarness:
		if provider.Host == "" {
			return nil, errors.New("scm provider Host missing")
		}
		account, org, project, _, err := extractHarnessScope(slug)
		if err != nil {
			return nil, fmt.Errorf("invalid slug format: %w", err)
		}
		c, err = harness.New(provider.Host, account, org, project)
		if err != nil {
			return nil, fmt.Errorf("scm provider Host invalid: %w", err)
		}
		transport = apiKeyTransport(r.baseTransport, provider.Password)

	default:
		return nil, fmt.Errorf("unsupported scm provider: %s", provider)
	}

	if r.traceHTTP {
		transport = tracing.ExternalHTTPTransport(transport)
	}
	c.Client = &http.Client{
		Transport: transport,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return c, nil
}

func (r *Importer) LoadRepositoryFromProvider(
	ctx context.Context,
	provider Provider,
	repoSlug string,
) (RepositoryInfo, Provider, error) {
	if err := validateProviderRepoSlug(provider.Type, repoSlug); err != nil {
		return RepositoryInfo{}, provider, err
	}

	scmClient, err := r.getScmClientWithTransport(provider, repoSlug, false)
	if err != nil {
		return RepositoryInfo{}, provider, usererror.BadRequestf("Could not create client: %s", err)
	}

	// Augment user information if it's not provided for certain vendors.
	if provider.Password != "" && provider.Username == "" && provider.Type != ProviderTypeHarness {
		user, scmResp, err := scmClient.Users.Find(ctx)
		if err = convertSCMError(ctx, provider, "user", scmResp, err); err != nil {
			return RepositoryInfo{}, provider, err
		}
		provider.Username = user.Login
	}

	if provider.Type == ProviderTypeAzure {
		repoSlug, err = extractRepoFromSlug(repoSlug)
		if err != nil {
			return RepositoryInfo{}, provider, usererror.BadRequestf("Invalid slug format: %s", err)
		}
	}

	if provider.Type == ProviderTypeHarness {
		_, _, _, repoSlug, err = extractHarnessScope(repoSlug)
		if err != nil {
			return RepositoryInfo{}, provider, usererror.BadRequestf("Invalid slug format: %s", err)
		}
	}
	scmRepo, scmResp, err := scmClient.Repositories.Find(ctx, repoSlug)
	if err = convertSCMError(ctx, provider, repoSlug, scmResp, err); err != nil {
		return RepositoryInfo{}, provider, err
	}

	return RepositoryInfo{
		Space:         scmRepo.Namespace,
		Identifier:    scmRepo.Name,
		CloneURL:      scmRepo.Clone,
		IsPublic:      !scmRepo.Private,
		DefaultBranch: scmRepo.Branch,
	}, provider, nil
}

//nolint:gocognit
func (r *Importer) LoadRepositoriesFromProviderSpace(
	ctx context.Context,
	provider Provider,
	spaceSlug string,
	includeSubgroupsRepos bool,
) ([]RepositoryInfo, Provider, error) {
	if err := validateProviderSpaceSlug(provider.Type, spaceSlug); err != nil {
		return nil, provider, err
	}

	var err error
	scmClient, err := r.getScmClientWithTransport(provider, spaceSlug, false)
	if err != nil {
		return nil, provider, usererror.BadRequestf("Could not create client: %s", err)
	}

	opts := scm.ListOptions{
		Size: 100,
	}

	// Augment user information if it's not provided for certain vendors.
	if provider.Password != "" && provider.Username == "" {
		user, scmResp, err := scmClient.Users.Find(ctx)
		if err = convertSCMError(ctx, provider, "user", scmResp, err); err != nil {
			return nil, provider, err
		}
		provider.Username = user.Login
	}

	var optsv2 scm.RepoListOptions
	listv2 := false
	//nolint:exhaustive
	switch provider.Type {
	case ProviderTypeGitHub:
		listv2 = true
		optsv2 = scm.RepoListOptions{
			ListOptions: opts,
			RepoSearchTerm: scm.RepoSearchTerm{
				User: spaceSlug + "+fork:true",
			},
		}
	case ProviderTypeGitLab:
		listv2 = true
		optsv2 = scm.RepoListOptions{
			ListOptions:      opts,
			Group:            spaceSlug,
			IncludeSubgroups: includeSubgroupsRepos,
		}
	}

	repos := make([]RepositoryInfo, 0)
	var scmRepos []*scm.Repository
	var scmResp *scm.Response

	for {
		if listv2 {
			scmRepos, scmResp, err = scmClient.Repositories.ListV2(ctx, optsv2)
			if err = convertSCMError(ctx, provider, spaceSlug, scmResp, err); err != nil {
				return nil, provider, err
			}
			optsv2.Page = scmResp.Page.Next
			optsv2.URL = scmResp.Page.NextURL
		} else {
			scmRepos, scmResp, err = scmClient.Repositories.List(ctx, opts)
			if err = convertSCMError(ctx, provider, spaceSlug, scmResp, err); err != nil {
				return nil, provider, err
			}
			opts.Page = scmResp.Page.Next
			opts.URL = scmResp.Page.NextURL
		}

		if len(scmRepos) == 0 {
			break
		}

		for _, scmRepo := range scmRepos {
			// in some cases the namespace filter isn't working (e.g. Gitlab)
			// For GitLab with subgroups, namespace could be "group/subgroup", so use prefix match
			if !matchesNamespace(provider.Type, scmRepo.Namespace, spaceSlug, includeSubgroupsRepos) {
				continue
			}

			repos = append(repos, RepositoryInfo{
				Space:         scmRepo.Namespace,
				Identifier:    scmRepo.Name,
				CloneURL:      scmRepo.Clone,
				IsPublic:      !scmRepo.Private,
				DefaultBranch: scmRepo.Branch,
			})
		}

		if listv2 {
			if optsv2.Page == 0 && optsv2.URL == "" {
				break
			}
		} else {
			if opts.Page == 0 && opts.URL == "" {
				break
			}
		}
	}

	return repos, provider, nil
}

func extractOrgAndProjectFromSlug(slug string) (string, string, error) {
	res := strings.Split(slug, "/")
	if len(res) < 2 {
		return "", "", fmt.Errorf("organization or project info missing")
	}
	if len(res) > 3 {
		return "", "", fmt.Errorf("too many parts")
	}
	return res[0], res[1], nil
}

func extractRepoFromSlug(slug string) (string, error) {
	res := strings.Split(slug, "/")
	if len(res) == 3 {
		return res[2], nil
	}
	return "", fmt.Errorf("repo name missing")
}

// extractHarnessScope splits a Harness Code provider_repo slug into the account/org/project
// scope and the repo identifier. Only account is required; org and project are optional and
// returned empty if not present, matching account/repo, account/org/repo and
// account/org/project/repo slug forms.
func extractHarnessScope(slug string) (account, org, project, repo string, err error) {
	parts := strings.Split(slug, "/")
	switch len(parts) {
	case 2:
		return parts[0], "", "", parts[1], nil
	case 3:
		return parts[0], parts[1], "", parts[2], nil
	case 4:
		return parts[0], parts[1], parts[2], parts[3], nil
	default:
		return "", "", "", "", fmt.Errorf(
			"expected account/repo, account/org/repo or account/org/project/repo, got %q", slug)
	}
}

// matchesNamespace checks if the repository namespace matches the expected space slug.
// For GitLab with includeSubgroups enabled, it matches the exact group or any subgroup
// (e.g., "group" and "group/subgroup" match "group", but "grouptoo" does not).
// For other providers or GitLab without includeSubgroups, it uses exact case-insensitive matching.
func matchesNamespace(providerType ProviderType, repoNamespace, spaceSlug string, includeSubgroups bool) bool {
	if providerType == ProviderTypeGitLab && includeSubgroups {
		lower := strings.ToLower(repoNamespace)
		slug := strings.ToLower(spaceSlug)
		return lower == slug || strings.HasPrefix(lower, slug+"/")
	}
	return strings.EqualFold(repoNamespace, spaceSlug)
}

// convertSCMError translates a failed provider call into a user facing error.
//
// IMPORTANT: the provider host is user provided, so the details of a failure
// must not be described to the caller. Neither the transport error (which names
// the resolved address and the syscall) nor the upstream status code may be
// exposed: both let a caller tell an open internal port apart from a closed one
// and use repository import as a network scanner. The details are logged
// instead.
func convertSCMError(ctx context.Context, provider Provider, slug string, r *scm.Response, err error) error {
	if err == nil {
		return nil
	}

	logger := log.Ctx(ctx).Warn().Err(err).
		Str("provider_type", string(provider.Type)).
		Str("provider_host", provider.Host)

	if r == nil {
		logger.Msg("failed to make http request to import provider")

		return usererror.BadRequestf(
			"Failed to make an HTTP request to %s. Verify that the provider host is correct and reachable.",
			provider.Type)
	}

	logger = logger.Int("provider_status", r.Status)

	switch r.Status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		logger.Msg("import provider responded with a redirect")
		return usererror.BadRequest("Redirects are not supported.")
	case http.StatusNotFound:
		logger.Msg("import provider could not find the requested resource")
		return usererror.BadRequestf("Couldn't find %s at %s.", slug, provider.Type)
	case http.StatusUnauthorized:
		logger.Msg("import provider rejected the provided credentials")
		return usererror.BadRequestf("Bad credentials provided for %s at %s.", slug, provider.Type)
	case http.StatusForbidden:
		logger.Msg("import provider denied access to the requested resource")
		return usererror.BadRequestf("Access denied to %s at %s.", slug, provider.Type)
	default:
		logger.Msg("failed to fetch resource from import provider")
		return usererror.BadRequestf("Failed to fetch %s from %s.", slug, provider.Type)
	}
}
