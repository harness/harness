// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type ImportInput struct {
	ParentRef   string `json:"parent_ref"`
	UID         string `json:"uid"`
	Description string `json:"description"`

	Provider     importer.Provider `json:"provider"`
	ProviderRepo string            `json:"provider_repo"`
}

// Import creates a new empty repository and starts git import to it from a remote repository.
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*types.Repository, error) {
	parentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	err = c.sanitizeImportInput(in)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	remoteRepository, err := importer.LoadRepositoryFromProvider(ctx, in.Provider, in.ProviderRepo)
	if err != nil {
		return nil, err
	}

	jobUID, err := job.UID()
	if err != nil {
		return nil, fmt.Errorf("error creating job UID: %w", err)
	}

	var repo *types.Repository
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		// lock parent space path to ensure it doesn't get updated while we setup new repo
		spacePath, err := c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, parentSpace.ID)
		if err != nil {
			return usererror.BadRequest("Parent not found'")
		}

		pathToRepo := paths.Concatinate(spacePath.Value, in.UID)
		repo = remoteRepository.ToRepo(parentSpace.ID, pathToRepo, in.UID, in.Description, jobUID, &session.Principal)

		err = c.repoStore.Create(ctx, repo)
		if err != nil {
			return fmt.Errorf("failed to create repository in storage: %w", err)
		}

		path := &types.Path{
			Version:    0,
			Value:      repo.Path,
			IsPrimary:  true,
			TargetType: enum.PathTargetTypeRepo,
			TargetID:   repo.ID,
			CreatedBy:  repo.CreatedBy,
			Created:    repo.Created,
			Updated:    repo.Updated,
		}

		err = c.pathStore.Create(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	err = c.importer.Run(ctx, in.Provider, repo, remoteRepository.CloneURL)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to start import repository job")
	}

	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeImportInput(in *ImportInput) error {
	if err := c.validateParentRef(in.ParentRef); err != nil {
		return err
	}

	if err := c.uidCheck(in.UID, false); err != nil {
		return err
	}

	return nil
}
