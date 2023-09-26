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
	"fmt"
	"net/http"
	"time"

	"github.com/harness/gitness/pkg/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

type ProviderType string

const (
	ProviderTypeGitHub ProviderType = "github"
	ProviderTypeGitLab ProviderType = "gitlab"
)

func (p ProviderType) Enum() []any {
	return []any{
		ProviderTypeGitHub,
		ProviderTypeGitLab,
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
	UID           string
	CloneURL      string
	IsPublic      bool
	DefaultBranch string
}

// ToRepo converts the RepositoryInfo into the types.Repository object marked as being imported.
func (r *RepositoryInfo) ToRepo(
	spaceID int64,
	uid string,
	description string,
	principal *types.Principal,
) *types.Repository {
	now := time.Now().UnixMilli()
	gitTempUID := fmt.Sprintf("importing-%s-%d", hash(fmt.Sprintf("%d:%s", spaceID, uid)), now)
	return &types.Repository{
		Version:       0,
		ParentID:      spaceID,
		UID:           uid,
		GitUID:        gitTempUID, // the correct git UID will be set by the job handler
		Description:   description,
		IsPublic:      r.IsPublic,
		CreatedBy:     principal.ID,
		Created:       now,
		Updated:       now,
		ForkID:        0,
		DefaultBranch: r.DefaultBranch,
		Importing:     true,
	}
}

func hash(s string) string {
	h := sha512.New()
	_, _ = h.Write([]byte(s))
	return base32.StdEncoding.EncodeToString(h.Sum(nil)[:10])
}

func getClient(provider Provider, authReq bool) (*scm.Client, error) {
	if authReq && (provider.Username == "" || provider.Password == "") {
		return nil, usererror.BadRequest("scm provider authentication credentials missing")
	}

	var c *scm.Client
	var err error

	switch provider.Type {
	case "":
		return nil, usererror.BadRequest("scm provider can not be empty")

	case ProviderTypeGitHub:
		if provider.Host != "" {
			c, err = github.New(provider.Host)
			if err != nil {
				return nil, usererror.BadRequestf("scm provider Host invalid: %s", err.Error())
			}
		} else {
			c = github.NewDefault()
		}

	case ProviderTypeGitLab:
		if provider.Host != "" {
			c, err = gitlab.New(provider.Host)
			if err != nil {
				return nil, usererror.BadRequestf("scm provider Host invalid: %s", err.Error())
			}
		} else {
			c = gitlab.NewDefault()
		}

	default:
		return nil, usererror.BadRequestf("unsupported scm provider: %s", provider)
	}

	if provider.Password != "" {
		c.Client = &http.Client{
			Transport: &oauth2.Transport{
				Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
			},
		}
	}

	return c, nil
}
func LoadRepositoryFromProvider(ctx context.Context, provider Provider, repoSlug string) (RepositoryInfo, error) {
	scmClient, err := getClient(provider, false)
	if err != nil {
		return RepositoryInfo{}, err
	}

	if repoSlug == "" {
		return RepositoryInfo{}, usererror.BadRequest("provider repository identifier is missing")
	}

	scmRepo, scmResp, err := scmClient.Repositories.Find(ctx, repoSlug)
	if err = convertSCMError(provider, repoSlug, scmResp, err); err != nil {
		return RepositoryInfo{}, err
	}

	return RepositoryInfo{
		Space:         scmRepo.Namespace,
		UID:           scmRepo.Name,
		CloneURL:      scmRepo.Clone,
		IsPublic:      !scmRepo.Private,
		DefaultBranch: scmRepo.Branch,
	}, nil
}

func LoadRepositoriesFromProviderSpace(
	ctx context.Context,
	provider Provider,
	spaceSlug string,
) ([]RepositoryInfo, error) {
	scmClient, err := getClient(provider, true)
	if err != nil {
		return nil, err
	}

	if spaceSlug == "" {
		return nil, usererror.BadRequest("provider space identifier is missing")
	}

	const pageSize = 100
	opts := scm.RepoListOptions{
		ListOptions: scm.ListOptions{
			Page: 0,
			Size: pageSize,
		},
		RepoSearchTerm: scm.RepoSearchTerm{
			User: spaceSlug,
		},
	}

	repos := make([]RepositoryInfo, 0)
	for {
		opts.Page++

		scmRepos, scmResp, err := scmClient.Repositories.ListV2(ctx, opts)
		if err = convertSCMError(provider, spaceSlug, scmResp, err); err != nil {
			return nil, err
		}

		if len(scmRepos) == 0 {
			break
		}

		for _, scmRepo := range scmRepos {
			// in some cases the namespace filter isn't working (e.g. Gitlab)
			if scmRepo.Namespace != spaceSlug {
				continue
			}

			repos = append(repos, RepositoryInfo{
				Space:         scmRepo.Namespace,
				UID:           scmRepo.Name,
				CloneURL:      scmRepo.Clone,
				IsPublic:      !scmRepo.Private,
				DefaultBranch: scmRepo.Branch,
			})
		}
	}

	return repos, nil
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
