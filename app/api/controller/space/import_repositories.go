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

package space

import (
	"context"
	"errors"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ImportRepositoriesInput struct {
	ProviderInput
}

type ImportRepositoriesOutput struct {
	ImportingRepos []*types.Repository `json:"importing_repos"`
	DuplicateRepos []*types.Repository `json:"duplicate_repos"` // repos which already exist in the space.
}

// getSpaceCheckAuthRepoCreation checks whether the user has permissions to create repos
// in the given space.
func (c *Controller) getSpaceCheckAuthRepoCreation(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (*types.Space, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("parent space not found: %w", err)
	}

	// create is a special case - check permission without specific resource
	scope := &types.Scope{SpacePath: space.Path}
	resource := &types.Resource{
		Type:       enum.ResourceTypeRepo,
		Identifier: "",
	}

	err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	return space, nil
}

// ImportRepositories imports repositories into an existing space. It ignores and continues on
// repo naming conflicts.
func (c *Controller) ImportRepositories(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *ImportRepositoriesInput,
) (ImportRepositoriesOutput, error) {
	space, err := c.getSpaceCheckAuthRepoCreation(ctx, session, spaceRef)
	if err != nil {
		return ImportRepositoriesOutput{}, err
	}

	remoteRepositories, provider, err :=
		importer.LoadRepositoriesFromProviderSpace(ctx, in.Provider, in.ProviderSpace)
	if err != nil {
		return ImportRepositoriesOutput{}, err
	}

	if len(remoteRepositories) == 0 {
		return ImportRepositoriesOutput{}, usererror.BadRequestf("found no repositories at %s", in.ProviderSpace)
	}

	repoIDs := make([]int64, 0, len(remoteRepositories))
	cloneURLs := make([]string, 0, len(remoteRepositories))
	repos := make([]*types.Repository, 0, len(remoteRepositories))
	duplicateRepos := make([]*types.Repository, 0, len(remoteRepositories))

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(
			ctx, space.ID, len(remoteRepositories)); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		for _, remoteRepository := range remoteRepositories {
			repo := remoteRepository.ToRepo(
				space.ID,
				remoteRepository.Identifier,
				"",
				&session.Principal,
				c.publicResourceCreationEnabled,
			)

			err = c.repoStore.Create(ctx, repo)
			if errors.Is(err, store.ErrDuplicate) {
				log.Ctx(ctx).Warn().Err(err).Msg("skipping duplicate repo")
				duplicateRepos = append(duplicateRepos, repo)
				continue
			} else if err != nil {
				return fmt.Errorf("failed to create repository in storage: %w", err)
			}
			repos = append(repos, repo)
			repoIDs = append(repoIDs, repo.ID)
			cloneURLs = append(cloneURLs, remoteRepository.CloneURL)
		}
		if len(repoIDs) == 0 {
			return nil
		}

		jobGroupID := fmt.Sprintf("space-import-%d", space.ID)
		err = c.importer.RunMany(ctx, jobGroupID, provider, repoIDs, cloneURLs, in.Pipelines)
		if err != nil {
			return fmt.Errorf("failed to start import repository jobs: %w", err)
		}

		return nil
	})
	if err != nil {
		return ImportRepositoriesOutput{}, err
	}

	return ImportRepositoriesOutput{ImportingRepos: repos, DuplicateRepos: duplicateRepos}, nil
}
