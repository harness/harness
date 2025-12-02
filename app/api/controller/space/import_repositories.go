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
	"time"

	"github.com/harness/gitness/app/api/controller/limiter"
	repoctrl "github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type ImportRepositoriesInput struct {
	ProviderInput
}

type ImportRepositoriesOutput struct {
	ImportingRepos []*repoctrl.RepositoryOutput `json:"importing_repos"`
}

// ImportRepositories imports repositories into an existing space.
//
//nolint:gocognit
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
		return ImportRepositoriesOutput{}, usererror.BadRequestf("Found no repositories in %q", in.ProviderSpace)
	}

	repos := make([]*types.Repository, 0, len(remoteRepositories))
	repoIDs := make([]int64, 0, len(remoteRepositories))
	repoIsPublicVals := make([]bool, 0, len(remoteRepositories))
	cloneURLs := make([]string, 0, len(remoteRepositories))

	for _, remoteRepository := range remoteRepositories {
		repo, isPublic := remoteRepository.ToRepo(
			space.ID,
			space.Path,
			remoteRepository.Identifier,
			"",
			&session.Principal,
		)

		if err := c.repoIdentifierCheck(repo.Identifier, session); err != nil {
			return ImportRepositoriesOutput{}, fmt.Errorf("failed to sanitize the repo %s: %w", repo.Identifier, err)
		}

		repos = append(repos, repo)
		repoIsPublicVals = append(repoIsPublicVals, isPublic)
		cloneURLs = append(cloneURLs, remoteRepository.CloneURL)
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		// lock the space for update during repo creation to prevent racing conditions with space soft delete.
		_, err = c.spaceStore.FindForUpdate(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to find the parent space: %w", err)
		}

		if err := c.resourceLimiter.RepoCount(
			ctx, space.ID, len(remoteRepositories)); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		for _, repo := range repos {
			err = c.repoStore.Create(ctx, repo)
			if errors.Is(err, store.ErrDuplicate) {
				return fmt.Errorf("failed to create duplicate repo %s", repo.Identifier)
			} else if err != nil {
				return fmt.Errorf("failed to create repository in storage: %w", err)
			}

			repoIDs = append(repoIDs, repo.ID)
		}
		if len(repoIDs) == 0 {
			return nil
		}

		jobGroupID := fmt.Sprintf("space-import-%d", space.ID)
		err = c.importer.RunMany(ctx,
			jobGroupID,
			provider,
			repoIDs,
			repoIsPublicVals,
			cloneURLs,
			in.Pipelines,
		)
		if err != nil {
			return fmt.Errorf("failed to start import repository jobs: %w", err)
		}

		return nil
	})
	if err != nil {
		return ImportRepositoriesOutput{}, err
	}

	reposOut := make([]*repoctrl.RepositoryOutput, len(repos))
	for i, repo := range repos {
		reposOut[i], err = repoctrl.GetRepoOutputWithAccess(ctx, c.repoFinder, false, repo)
		if err != nil {
			return ImportRepositoriesOutput{}, fmt.Errorf("failed to get repo output: %w", err)
		}

		err = c.auditService.Log(ctx,
			session.Principal,
			audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
			audit.ActionCreated,
			paths.Parent(repo.Path),
			audit.WithNewObject(audit.RepositoryObject{
				Repository: *repo,
				IsPublic:   false, // in import we configure public access and create a new audit log.
			}),
		)
		if err != nil {
			log.Warn().Msgf("failed to insert audit log for import repository operation: %s", err)
		}
		err = c.instrumentation.Track(ctx, instrument.Event{
			Type:      instrument.EventTypeRepositoryCreate,
			Principal: session.Principal.ToPrincipalInfo(),
			Timestamp: time.Now(),
			Path:      space.Path,
			Properties: map[instrument.Property]any{
				instrument.PropertyRepositoryID:           repo.ID,
				instrument.PropertyRepositoryName:         repo.Identifier,
				instrument.PropertyRepositoryCreationType: instrument.CreationTypeImport,
			},
		})
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for import repository operation: %s", err)
		}
	}

	return ImportRepositoriesOutput{ImportingRepos: reposOut}, nil
}
