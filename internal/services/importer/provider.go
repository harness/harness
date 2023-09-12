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
	if provider.Username == "" || provider.Password == "" {
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

	c.Client = &http.Client{
		Transport: &oauth2.Transport{
			Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Password}),
		},
	}

	return c, nil
}
func LoadRepositoryFromProvider(ctx context.Context, provider Provider, repoSlug string) (RepositoryInfo, error) {
	scmClient, err := getClient(provider)
	if err != nil {
		return RepositoryInfo{}, err
	}

	if repoSlug == "" {
		return RepositoryInfo{}, usererror.BadRequest("provider repository identifier is missing")
	}

	var statusCode int
	scmRepo, scmResp, err := scmClient.Repositories.Find(ctx, repoSlug)
	if scmResp != nil {
		statusCode = scmResp.Status
	}

	if errors.Is(err, scm.ErrNotFound) || statusCode == http.StatusNotFound {
		return RepositoryInfo{},
			usererror.BadRequestf("repository %s not found at %s", repoSlug, provider.Type)
	}
	if errors.Is(err, scm.ErrNotAuthorized) || statusCode == http.StatusUnauthorized {
		return RepositoryInfo{},
			usererror.BadRequestf("bad credentials provided for %s at %s", repoSlug, provider.Type)
	}
	if err != nil || statusCode > 299 {
		return RepositoryInfo{},
			fmt.Errorf("failed to fetch repository %s from %s, status=%d: %w",
				repoSlug, provider.Type, statusCode, err)
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

	const pageSize = 100
	opts := scm.ListOptions{Page: 0, Size: pageSize}

	repos := make([]RepositoryInfo, 0)
	for {
		opts.Page++

		var statusCode int
		scmRepos, scmResp, err := scmClient.Repositories.List(ctx, opts)
		if scmResp != nil {
			statusCode = scmResp.Status
		}

		if errors.Is(err, scm.ErrNotFound) || statusCode == http.StatusNotFound {
			return nil, usererror.BadRequestf("space %s not found at %s", spaceSlug, provider.Type)
		}
		if errors.Is(err, scm.ErrNotAuthorized) || statusCode == http.StatusUnauthorized {
			return nil, usererror.BadRequestf("bad credentials provided for %s at %s", spaceSlug, provider.Type)
		}
		if err != nil || statusCode > 299 {
			return nil, fmt.Errorf("failed to fetch space %s from %s, status=%d: %w",
				spaceSlug, provider.Type, statusCode, err)
		}

		if len(scmRepos) == 0 {
			break
		}

		for _, scmRepo := range scmRepos {
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
