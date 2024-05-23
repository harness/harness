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
	"fmt"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type ProviderInput struct {
	Provider      importer.Provider       `json:"provider"`
	ProviderSpace string                  `json:"provider_space"`
	Pipelines     importer.PipelineOption `json:"pipelines"`
}

type ImportInput struct {
	CreateInput
	ProviderInput
}

// Import creates new space and starts import of all repositories from the remote provider's space into it.
//
//nolint:gocognit
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*SpaceOutput, error) {
	parentSpace, err := c.getSpaceCheckAuthSpaceCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	if in.Identifier == "" && in.UID == "" {
		in.Identifier = in.ProviderSpace
	}

	err = c.sanitizeImportInput(in)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	remoteRepositories, provider, err :=
		importer.LoadRepositoriesFromProviderSpace(ctx, in.Provider, in.ProviderSpace)
	if err != nil {
		return nil, err
	}

	if len(remoteRepositories) == 0 {
		return nil, usererror.BadRequestf("found no repositories at %s", in.ProviderSpace)
	}

	repoIDs := make([]int64, len(remoteRepositories))
	repoIsPublicVals := make([]bool, len(remoteRepositories))
	cloneURLs := make([]string, len(remoteRepositories))
	repos := make([]*types.Repository, 0, len(remoteRepositories))

	var space *types.Space
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(
			ctx, parentSpace.ID, len(remoteRepositories)); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		space, err = c.createSpaceInnerInTX(ctx, session, parentSpace.ID, &in.CreateInput)
		if err != nil {
			return err
		}

		for i, remoteRepository := range remoteRepositories {
			repo, isPublic := remoteRepository.ToRepo(
				space.ID,
				space.Path,
				remoteRepository.Identifier,
				"",
				&session.Principal,
			)

			err = c.repoStore.Create(ctx, repo)
			if err != nil {
				return fmt.Errorf("failed to create repository in storage: %w", err)
			}
			repos = append(repos, repo)
			repoIDs[i] = repo.ID
			cloneURLs[i] = remoteRepository.CloneURL
			repoIsPublicVals[i] = isPublic
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
		return nil, err
	}

	for _, repo := range repos {
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
	}

	return GetSpaceOutput(ctx, c.publicAccess, space)
}

func (c *Controller) sanitizeImportInput(in *ImportInput) error {
	if err := c.sanitizeCreateInput(&in.CreateInput); err != nil {
		return err
	}

	if in.Pipelines == "" {
		in.Pipelines = importer.PipelineOptionConvert
	}

	return nil
}
