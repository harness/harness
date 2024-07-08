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
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/azure"
	"github.com/drone/go-scm/scm/driver/bitbucket"
	"github.com/drone/go-scm/scm/driver/gitea"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/driver/gogs"
	"github.com/drone/go-scm/scm/driver/stash"
	"github.com/drone/go-scm/scm/transport"
	"github.com/drone/go-scm/scm/transport/oauth2"
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
	now := time.Now().UnixMilli()
	gitTempUID := fmt.Sprintf("importing-%s-%d", hash(fmt.Sprintf("%d:%s", spaceID, identifier)), now)
	return &types.Repository{
		Version:       0,
		ParentID:      spaceID,
		Identifier:    identifier,
		GitUID:        gitTempUID, // the correct git UID will be set by the job handler
		Description:   description,
		CreatedBy:     principal.ID,
		Created:       now,
		Updated:       now,
		ForkID:        0,
		DefaultBranch: r.DefaultBranch,
		State:         enum.RepoStateGitImport,
		Path:          paths.Concatenate(spacePath, identifier),
	}, r.IsPublic
}

func hash(s string) string {
	h := sha512.New()
	_, _ = h.Write([]byte(s))
	return base32.StdEncoding.EncodeToString(h.Sum(nil)[:10])
}

func oauthTransport(token string, scheme string) http.RoundTripper {
	if token == "" {
		return nil
	}
	return &oauth2.Transport{
		Scheme: scheme,
		Source: oauth2.StaticTokenSource(&scm.Token{Token: token}),
	}
}

func authHeaderTransport(token string) http.RoundTripper {
	if token == "" {
		return nil
	}
	return &transport.Authorization{
		Scheme:      "token",
		Credentials: token,
	}
}

func basicAuthTransport(username, password string) http.RoundTripper {
	if username == "" && password == "" {
		return nil
	}
	return &transport.BasicAuth{
		Username: username,
		Password: password,
	}
}

// getScmClientWithTransport creates an SCM client along with the necessary transport
// layer depending on the provider. For example, for bitbucket we support app passwords
// so the auth transport is BasicAuth whereas it's Oauth for other providers.
// It validates that auth credentials are provided if authReq is true.
func getScmClientWithTransport(provider Provider, slug string, authReq bool) (*scm.Client, error) { //nolint:gocognit
	if authReq && (provider.Username == "" || provider.Password == "") {
		return nil, usererror.BadRequest("scm provider authentication credentials missing")
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
		transport = oauthTransport(provider.Password, oauth2.SchemeBearer)

	case ProviderTypeGitLab:
		if provider.Host != "" {
			c, err = gitlab.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = gitlab.NewDefault()
		}
		transport = oauthTransport(provider.Password, oauth2.SchemeBearer)

	case ProviderTypeBitbucket:
		if provider.Host != "" {
			c, err = bitbucket.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = bitbucket.NewDefault()
		}
		transport = basicAuthTransport(provider.Username, provider.Password)

	case ProviderTypeStash:
		if provider.Host != "" {
			c, err = stash.New(provider.Host)
			if err != nil {
				return nil, fmt.Errorf("scm provider Host invalid: %w", err)
			}
		} else {
			c = stash.NewDefault()
		}
		transport = oauthTransport(provider.Password, oauth2.SchemeBearer)

	case ProviderTypeGitea:
		if provider.Host == "" {
			return nil, errors.New("scm provider Host missing")
		}
		c, err = gitea.New(provider.Host)
		if err != nil {
			return nil, fmt.Errorf("scm provider Host invalid: %w", err)
		}
		transport = authHeaderTransport(provider.Password)

	case ProviderTypeGogs:
		if provider.Host == "" {
			return nil, errors.New("scm provider Host missing")
		}
		c, err = gogs.New(provider.Host)
		if err != nil {
			return nil, fmt.Errorf("scm provider Host invalid: %w", err)
		}
		transport = oauthTransport(provider.Password, oauth2.SchemeToken)

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
		transport = basicAuthTransport(provider.Username, provider.Password)

	default:
		return nil, fmt.Errorf("unsupported scm provider: %s", provider)
	}

	// override default transport if available
	if transport != nil {
		c.Client = &http.Client{Transport: transport}
	}

	return c, nil
}

func LoadRepositoryFromProvider(
	ctx context.Context,
	provider Provider,
	repoSlug string,
) (RepositoryInfo, Provider, error) {
	if repoSlug == "" {
		return RepositoryInfo{}, provider, usererror.BadRequest("provider repository identifier is missing")
	}

	scmClient, err := getScmClientWithTransport(provider, repoSlug, false)
	if err != nil {
		return RepositoryInfo{}, provider, usererror.BadRequestf("could not create client: %s", err)
	}

	// Augment user information if it's not provided for certain vendors.
	if provider.Password != "" && provider.Username == "" {
		user, _, err := scmClient.Users.Find(ctx)
		if err != nil {
			return RepositoryInfo{}, provider, usererror.BadRequestf("could not find user: %s", err)
		}
		provider.Username = user.Login
	}

	if provider.Type == ProviderTypeAzure {
		repoSlug, err = extractRepoFromSlug(repoSlug)
		if err != nil {
			return RepositoryInfo{}, provider, usererror.BadRequestf("invalid slug format: %s", err)
		}
	}
	scmRepo, scmResp, err := scmClient.Repositories.Find(ctx, repoSlug)
	if err = convertSCMError(provider, repoSlug, scmResp, err); err != nil {
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
func LoadRepositoriesFromProviderSpace(
	ctx context.Context,
	provider Provider,
	spaceSlug string,
) ([]RepositoryInfo, Provider, error) {
	if spaceSlug == "" {
		return nil, provider, usererror.BadRequest("provider space identifier is missing")
	}

	var err error
	scmClient, err := getScmClientWithTransport(provider, spaceSlug, false)
	if err != nil {
		return nil, provider, usererror.BadRequestf("could not create client: %s", err)
	}

	opts := scm.ListOptions{
		Size: 100,
	}

	// Augment user information if it's not provided for certain vendors.
	if provider.Password != "" && provider.Username == "" {
		user, _, err := scmClient.Users.Find(ctx)
		if err != nil {
			return nil, provider, usererror.BadRequestf("could not find user: %s", err)
		}
		provider.Username = user.Login
	}

	var optsv2 scm.RepoListOptions
	listv2 := false
	if provider.Type == ProviderTypeGitHub {
		listv2 = true
		optsv2 = scm.RepoListOptions{
			ListOptions: opts,
			RepoSearchTerm: scm.RepoSearchTerm{
				User: spaceSlug,
			},
		}
	}

	repos := make([]RepositoryInfo, 0)
	var scmRepos []*scm.Repository
	var scmResp *scm.Response

	for {
		if listv2 {
			scmRepos, scmResp, err = scmClient.Repositories.ListV2(ctx, optsv2)
			if err = convertSCMError(provider, spaceSlug, scmResp, err); err != nil {
				return nil, provider, err
			}
			optsv2.Page = scmResp.Page.Next
			optsv2.URL = scmResp.Page.NextURL
		} else {
			scmRepos, scmResp, err = scmClient.Repositories.List(ctx, opts)
			if err = convertSCMError(provider, spaceSlug, scmResp, err); err != nil {
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
			if !strings.EqualFold(scmRepo.Namespace, spaceSlug) {
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

func convertSCMError(provider Provider, slug string, r *scm.Response, err error) error {
	if err == nil {
		return nil
	}

	if r == nil {
		if provider.Host != "" {
			return usererror.BadRequestf("failed to make HTTP request to %s (host=%s): %s",
				provider.Type, provider.Host, err)
		}

		return usererror.BadRequestf("failed to make HTTP request to %s: %s",
			provider.Type, err)
	}

	switch r.Status {
	case http.StatusNotFound:
		return usererror.BadRequestf("couldn't find %s at %s: %s",
			slug, provider.Type, err.Error())
	case http.StatusUnauthorized:
		return usererror.BadRequestf("bad credentials provided for %s at %s: %s",
			slug, provider.Type, err.Error())
	case http.StatusForbidden:
		return usererror.BadRequestf("access denied to %s at %s: %s",
			slug, provider.Type, err.Error())
	default:
		return usererror.BadRequestf("failed to fetch %s from %s (HTTP status %d): %s",
			slug, provider.Type, r.Status, err.Error())
	}
}
