// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package importer

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/harness/gitness/internal/api/usererror"

	"github.com/drone/go-scm/scm"
	"github.com/drone/go-scm/scm/driver/github"
	"github.com/drone/go-scm/scm/transport/oauth2"
)

type ProviderType string

const (
	ProviderTypeGitHub ProviderType = "github"
)

type ProviderInfo struct {
	Type ProviderType
	Host string
	User string
	Pass string
}

type RepositoryInfo struct {
	Space         string
	UID           string
	CloneURL      string
	IsPublic      bool
	DefaultBranch string
}

func getClient(provider ProviderInfo) (*scm.Client, error) {
	var scmClient *scm.Client

	switch provider.Type {
	case "":
		return nil, usererror.BadRequest("provider can not be empty")
	case ProviderTypeGitHub:
		scmClient = github.NewDefault()
		if provider.Pass != "" {
			scmClient.Client = &http.Client{
				Transport: &oauth2.Transport{
					Source: oauth2.StaticTokenSource(&scm.Token{Token: provider.Pass}),
				},
			}
		}
	default:
		return nil, usererror.BadRequestf("unsupported provider: %s", provider)
	}

	return scmClient, nil
}
func Repo(ctx context.Context, provider ProviderInfo, repoSlug string) (RepositoryInfo, error) {
	scmClient, err := getClient(provider)
	if err != nil {
		return RepositoryInfo{}, err
	}

	scmRepo, _, err := scmClient.Repositories.Find(ctx, repoSlug)
	if errors.Is(err, scm.ErrNotFound) {
		return RepositoryInfo{}, usererror.BadRequestf("repository %s not found at %s", repoSlug, provider)
	}
	if errors.Is(err, scm.ErrNotAuthorized) {
		return RepositoryInfo{}, usererror.BadRequestf("bad credentials provided for %s at %s", repoSlug, provider)
	}
	if err != nil {
		return RepositoryInfo{}, fmt.Errorf("failed to fetch repository %s from %s: %w", repoSlug, provider, err)
	}

	return RepositoryInfo{
		Space:         scmRepo.Namespace,
		UID:           scmRepo.Name,
		CloneURL:      scmRepo.Clone,
		IsPublic:      !scmRepo.Private,
		DefaultBranch: scmRepo.Branch,
	}, nil
}

func Space(ctx context.Context, provider ProviderInfo, space string) (map[string]RepositoryInfo, error) {
	scmClient, err := getClient(provider)
	if err != nil {
		return nil, err
	}

	repoMap := make(map[string]RepositoryInfo)

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
				User:     space,
			},
		})
		if errors.Is(err, scm.ErrNotFound) {
			return nil, usererror.BadRequestf("space %s not found at %s", space, provider)
		}
		if errors.Is(err, scm.ErrNotAuthorized) {
			return nil, usererror.BadRequestf("bad credentials provided for %s at %s", space, provider)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to fetch space %s from %s: %w", space, provider, err)
		}

		for _, scmRepo := range scmRepos {
			if !scmRepo.Perm.Pull {
				continue
			}

			repoMap[scmRepo.Name] = RepositoryInfo{
				Space:         scmRepo.Namespace,
				UID:           scmRepo.Name,
				CloneURL:      scmRepo.Clone,
				IsPublic:      !scmRepo.Private,
				DefaultBranch: scmRepo.Branch,
			}
		}

		if len(scmRepos) == 0 || page == scmResponse.Page.Last {
			break
		}

		page++
	}

	return repoMap, nil
}
