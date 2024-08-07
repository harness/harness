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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/audit"

	"github.com/rs/zerolog/log"
)

type ImportInput struct {
	ParentRef string `json:"parent_ref"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID         string `json:"uid" deprecated:"true"`
	Identifier  string `json:"identifier"`
	Description string `json:"description"`

	Provider     importer.Provider `json:"provider"`
	ProviderRepo string            `json:"provider_repo"`

	Pipelines importer.PipelineOption `json:"pipelines"`
}

// Import creates a new empty repository and starts git import to it from a remote repository.
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*RepositoryOutput, error) {
	if err := c.sanitizeImportInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	remoteRepository, provider, err := importer.LoadRepositoryFromProvider(ctx, in.Provider, in.ProviderRepo)
	if err != nil {
		return nil, err
	}

	repo, isPublic := remoteRepository.ToRepo(
		parentSpace.ID,
		parentSpace.Path,
		in.Identifier,
		in.Description,
		&session.Principal,
	)

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, parentSpace.ID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		// lock the space for update during repo creation to prevent racing conditions with space soft delete.
		parentSpace, err = c.spaceStore.FindForUpdate(ctx, parentSpace.ID)
		if err != nil {
			return fmt.Errorf("failed to find the parent space: %w", err)
		}

		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to create repository in storage: %w", err)
		}

		err = c.importer.Run(ctx,
			provider,
			repo,
			isPublic,
			remoteRepository.CloneURL,
			in.Pipelines,
		)
		if err != nil {
			return fmt.Errorf("failed to start import repository job: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
		audit.ActionCreated,
		paths.Parent(repo.Path),
		audit.WithNewObject(audit.RepositoryObject{
			Repository: *repo,
			IsPublic:   false,
		}),
	)
	if err != nil {
		log.Warn().Msgf("failed to insert audit log for import repository operation: %s", err)
	}

	return GetRepoOutputWithAccess(ctx, false, repo), nil
}

func (c *Controller) sanitizeImportInput(in *ImportInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := ValidateParentRef(in.ParentRef); err != nil {
		return err
	}

	if err := c.identifierCheck(in.Identifier); err != nil {
		return err
	}

	if in.Pipelines == "" {
		in.Pipelines = importer.PipelineOptionConvert
	}

	return nil
}
