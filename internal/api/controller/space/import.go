// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type ImportInput struct {
	CreateInput
	Provider      importer.Provider `json:"provider"`
	ProviderSpace string            `json:"provider_space"`
}

// Import creates new space and starts import of all repositories from the remote provider's space into it.
func (c *Controller) Import(ctx context.Context, session *auth.Session, in *ImportInput) (*types.Space, error) {
	parentSpace, err := c.getSpaceCheckAuthSpaceCreation(ctx, session, in.ParentRef)
	if err != nil {
		return nil, err
	}

	if in.UID == "" {
		in.UID = in.ProviderSpace
	}

	err = c.sanitizeCreateInput(&in.CreateInput)
	if err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	remoteRepositories, err := importer.LoadRepositoriesFromProviderSpace(ctx, in.Provider, in.ProviderSpace)
	if err != nil {
		return nil, err
	}

	if len(remoteRepositories) == 0 {
		return nil, usererror.BadRequestf("found no repositories at %s", in.ProviderSpace)
	}

	localRepositories := make([]*types.Repository, len(remoteRepositories))
	cloneURLs := make([]string, len(remoteRepositories))

	var space *types.Space
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		spacePath := in.UID
		parentSpaceID := int64(0)
		if parentSpace != nil {
			parentSpaceID = parentSpace.ID
			// lock parent space path to ensure it doesn't get updated while we setup new space
			parentPath, err := c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, parentSpaceID)
			if err != nil {
				return usererror.BadRequest("Parent not found")
			}
			spacePath = paths.Concatinate(parentPath.Value, in.UID)

			// ensure path is within accepted depth!
			err = check.PathDepth(spacePath, true)
			if err != nil {
				return fmt.Errorf("path is invalid: %w", err)
			}
		}

		now := time.Now().UnixMilli()
		space = &types.Space{
			Version:     0,
			ParentID:    parentSpaceID,
			UID:         in.UID,
			Path:        spacePath,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   session.Principal.ID,
			Created:     now,
			Updated:     now,
		}
		err = c.spaceStore.Create(ctx, space)
		if err != nil {
			return fmt.Errorf("space creation failed: %w", err)
		}

		path := &types.Path{
			Version:    0,
			Value:      space.Path,
			IsPrimary:  true,
			TargetType: enum.PathTargetTypeSpace,
			TargetID:   space.ID,
			CreatedBy:  space.CreatedBy,
			Created:    now,
			Updated:    now,
		}
		err = c.pathStore.Create(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		// add space membership to top level space only (as the user doesn't have inherited permissions already)
		parentRefAsID, err := strconv.ParseInt(in.ParentRef, 10, 64)
		if (err == nil && parentRefAsID == 0) || (len(strings.TrimSpace(in.ParentRef)) == 0) {
			membership := &types.Membership{
				MembershipKey: types.MembershipKey{
					SpaceID:     space.ID,
					PrincipalID: session.Principal.ID,
				},
				Role: enum.MembershipRoleSpaceOwner,

				// membership has been created by the system
				CreatedBy: bootstrap.NewSystemServiceSession().Principal.ID,
				Created:   now,
				Updated:   now,
			}
			err = c.membershipStore.Create(ctx, membership)
			if err != nil {
				return fmt.Errorf("failed to make user owner of the space: %w", err)
			}
		}

		for i, remoteRepository := range remoteRepositories {
			var jobUID string

			jobUID, err = job.UID()
			if err != nil {
				return fmt.Errorf("error creating job UID: %w", err)
			}

			pathToRepo := paths.Concatinate(path.Value, remoteRepository.UID)
			repo := remoteRepository.ToRepo(
				space.ID, pathToRepo, remoteRepository.UID, "", jobUID, &session.Principal)

			err = c.repoStore.Create(ctx, repo)
			if err != nil {
				return fmt.Errorf("failed to create repository in storage: %w", err)
			}

			repoPath := &types.Path{
				Version:    0,
				Value:      repo.Path,
				IsPrimary:  true,
				TargetType: enum.PathTargetTypeRepo,
				TargetID:   repo.ID,
				CreatedBy:  repo.CreatedBy,
				Created:    repo.Created,
				Updated:    repo.Updated,
			}

			err = c.pathStore.Create(ctx, repoPath)
			if err != nil {
				return fmt.Errorf("failed to create path: %w", err)
			}

			localRepositories[i] = repo
			cloneURLs[i] = remoteRepository.CloneURL
		}

		jobGroupID := fmt.Sprintf("space-import-%d", space.ID)
		err = c.importer.RunMany(ctx, jobGroupID, in.Provider, localRepositories, cloneURLs)
		if err != nil {
			return fmt.Errorf("failed to start import repository jobs: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return space, nil
}
