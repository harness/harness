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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
	// ParentRef can be either a space ID or space path
	ParentRef *string `json:"parent_ref"`
}

func (i *MoveInput) hasChanges(
	repo *types.Repository,
	parentSpace *types.SpaceCore,
	targetParentSpace *types.SpaceCore,
) bool {
	if i.Identifier != nil && *i.Identifier != repo.Identifier {
		return true
	}

	if i.ParentRef != nil && targetParentSpace.ID != parentSpace.ID {
		return true
	}

	return false
}

// Move moves a repository to a new identifier and/or parent space.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *MoveInput,
) (*RepositoryOutput, error) {
	if err := c.sanitizeMoveInput(in, session); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repoCore, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to find or acquire access to repo: %w", err)
	}

	repo, err := c.repoStore.Find(ctx, repoCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ID: %w", err)
	}

	currentParentSpace, err := c.spaceFinder.FindByID(ctx, repo.ParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find current parent space: %w", err)
	}

	targetParentSpace := currentParentSpace
	if in.ParentRef != nil {
		targetParentSpace, err = c.getSpaceCheckAuthRepoCreation(ctx, session, *in.ParentRef)
		if err != nil {
			return nil, fmt.Errorf("failed to access target parent space: %w", err)
		}
	}

	if !in.hasChanges(repo, currentParentSpace, targetParentSpace) {
		return GetRepoOutput(ctx, c.publicAccess, repo)
	}

	isPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo public access: %w", err)
	}

	// remove public access from old repo path to avoid leaking it
	if err := c.publicAccess.Delete(
		ctx,
		enum.PublicResourceTypeRepo,
		repo.Path,
	); err != nil {
		return nil, fmt.Errorf("failed to remove public access on the original path: %w", err)
	}

	// TODO add a repo level lock here to avoid racing condition or partial repo update w/o setting repo public access
	movedRepo, err := c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		if in.Identifier != nil {
			r.Identifier = *in.Identifier
		}
		if targetParentSpace.ID != r.ParentID {
			r.ParentID = targetParentSpace.ID
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update repo: %w", err)
	}

	// clear old repo from cache
	c.repoFinder.MarkChanged(ctx, repo.Core())

	// set public access for the new repo path
	if err := c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, movedRepo.Path, isPublic); err != nil {
		// ensure public access for new repo path is cleaned up first or we risk leaking it
		if dErr := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, movedRepo.Path); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and public access cleanup: %w): %w", dErr, err)
		}

		// revert identifier and parent changes first
		var dErr error
		_, dErr = c.repoStore.UpdateOptLock(ctx, movedRepo, func(r *types.Repository) error {
			r.Identifier = repo.Identifier
			r.ParentID = repo.ParentID
			return nil
		})
		if dErr != nil {
			return nil, fmt.Errorf(
				"failed to set public access for new path (and reverting of move: %w): %w",
				dErr,
				err,
			)
		}

		// clear updated repo from cache
		c.repoFinder.MarkChanged(ctx, movedRepo.Core())

		// revert public access changes only after we successfully restored original path
		if dErr = c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, isPublic); dErr != nil {
			return nil, fmt.Errorf(
				"failed to set public access for new path (and reverting of public access: %w): %w",
				dErr,
				err,
			)
		}

		return nil, fmt.Errorf("failed to set repo public access for new path (cleanup successful): %w", err)
	}

	movedRepo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, movedRepo.Path)
	movedRepo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, movedRepo.Path)

	// TODO: add audit log
	log.Ctx(ctx).Info().Msgf(
		"Moved repository %s to %s operation perofrmed by %s",
		repo.Path, movedRepo.Path, session.Principal.Email)

	return GetRepoOutput(ctx, c.publicAccess, movedRepo)
}

func (c *Controller) sanitizeMoveInput(in *MoveInput, session *auth.Session) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := c.identifierCheck(*in.Identifier, session); err != nil {
			return err
		}
	}

	if in.ParentRef != nil {
		if err := ValidateParentRef(*in.ParentRef); err != nil {
			return fmt.Errorf("invalid space reference: %w", err)
		}
	}

	return nil
}
