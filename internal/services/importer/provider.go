// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/types"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/driver/gitlab"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

type ProviderType string

const (
	ProviderTypeGitHub           ProviderType = "github"
	ProviderTypeGitHubEnterprise ProviderType = "github-enterprise"
	ProviderTypeGitLab           ProviderType = "gitlab"
	ProviderTypeGitLabEnterprise ProviderType = "gitlab-enterprise"
)

func (p ProviderType) Enum() []any {
	return []any{
		ProviderTypeGitHub,
		ProviderTypeGitHubEnterprise,
		ProviderTypeGitLab,
		ProviderTypeGitLabEnterprise,
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
	path string,
	uid string,
	description string,
	jobUID string,
	principal *types.Principal,
) *types.Repository {
	now := time.Now().UnixMilli()
	gitTempUID := "importing-" + jobUID
	return &types.Repository{
		Version:         0,
		ParentID:        spaceID,
		UID:             uid,
		GitUID:          gitTempUID, // the correct git UID will be set by the job handler
		Path:            path,
		Description:     description,
		IsPublic:        r.IsPublic,
		CreatedBy:       principal.ID,
		Created:         now,
		Updated:         now,
		ForkID:          0,
		DefaultBranch:   r.DefaultBranch,
		Importing:       true,
		ImportingJobUID: &jobUID,
	}
}

func getClient(provider Provider) (*scm.Client, error) {
	switch provider.Type {
	case "":
		return nil, usererror.BadRequest("provider can not be empty")

	case ProviderTypeGitHub:
		c := github.NewDefault()
		if provider.Password != "" {
			c.Client = &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
				},
			}
		}
		return c, nil

	case ProviderTypeGitHubEnterprise:
		c, err := github.New(provider.Host)
		if err != nil {
			return nil, usererror.BadRequestf("provider Host invalid: %s", err.Error())
		}

		if provider.Password != "" {
			c.Client = &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
				},
			}
		}
		return c, nil

	case ProviderTypeGitLab:
		c := gitlab.NewDefault()
		if provider.Password != "" {
			c.Client = &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
				},
			}
		}

		return c, nil

	case ProviderTypeGitLabEnterprise:
		c, err := gitlab.New(provider.Host)
		if err != nil {
			return nil, usererror.BadRequestf("provider Host invalid: %s", err.Error())
		}

		if provider.Password != "" {
			c.Client = &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
				},
			}
		}

		return c, nil

	default:
		return nil, usererror.BadRequestf("unsupported provider: %s", provider)
	}
}
func LoadRepositoryFromProvider(ctx context.Context, provider Provider, repoSlug string) (RepositoryInfo, error) {
	scmClient, err := getClient(provider)
	if err != nil {
		return RepositoryInfo{}, err
	}

	if repoSlug == "" {
		return RepositoryInfo{}, usererror.BadRequest("provider repository identifier is missing")
	}

	scmRepo, _, err := scmClient.Repositories.Find(ctx, repoSlug)
	if errors.Is(err, scm.ErrNotFound) {
		return RepositoryInfo{},
			usererror.BadRequestf("repository %s not found at %s", repoSlug, provider.Type)
	}
	if errors.Is(err, scm.ErrNotAuthorized) {
		return RepositoryInfo{},
			usererror.BadRequestf("bad credentials provided for %s at %s", repoSlug, provider.Type)
	}
	if err != nil {
		return RepositoryInfo{},
			fmt.Errorf("failed to fetch repository %s from %s: %w", repoSlug, provider.Type, err)
	}

	return RepositoryInfo{
		Space:         scmRepo.Namespace,
		UID:           scmRepo.Name,
		CloneURL:      scmRepo.Clone,
		IsPublic:      !scmRepo.Private,
		DefaultBranch: scmRepo.Branch,
	}, nil
}

func LoadRepositoriesFromProviderSpace(ctx context.Context, provider Provider, spaceSlug string) ([]RepositoryInfo, error) {
	scmClient, err := getClient(provider)
	if err != nil {
		return nil, err
	}

	if spaceSlug == "" {
		return nil, usererror.BadRequest("provider space identifier is missing")
	}

	repos := make([]RepositoryInfo, 0)

	const pageSize = 50
	page := 1
	for {
		scmRepos, scmResponse, err := scmClient.Repositories.ListV2(ctx, scm.RepoListOptions{
			ListOptions: scm.ListOptions{
				URL:  "",
				Page: page,
				Size: pageSize,
			},
			RepoSearchTerm: scm.RepoSearchTerm{
				RepoName: "",
				User:     spaceSlug,
			},
		})
		if errors.Is(err, scm.ErrNotFound) {
			return nil, usererror.BadRequestf("space %s not found at %s", spaceSlug, provider.Type)
		}
		if errors.Is(err, scm.ErrNotAuthorized) {
			return nil, usererror.BadRequestf("bad credentials provided for %s at %s", spaceSlug, provider.Type)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch space %s from %s: %w", spaceSlug, provider.Type, err)
		}

		for _, scmRepo := range scmRepos {
			if !scmRepo.Perm.Pull {
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

		if len(scmRepos) == 0 || page == scmResponse.Page.Last {
			break
		}

		page++
	}

	return repos, nil
}
