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
		return GetRepoOutput(ctx, c.publicAccess, c.repoFinder, repo)
	}

	// the moved repo inherits the root space of its new parent space.
	targetParentSpaceFull, err := c.spaceStore.Find(ctx, targetParentSpace.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find target parent space '%d': %w", targetParentSpace.ID, err)
	}

	movedRepo, err := c.MoveNoAuth(
		ctx,
		repo,
		in.Identifier,
		targetParentSpaceFull.ID,
		targetParentSpaceFull.RootSpaceID,
		targetParentSpaceFull.RootSpaceIdentifier,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to move repo: %w", err)
	}

	movedRepo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, movedRepo.Path)
	movedRepo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, movedRepo.Path)

	// TODO: add audit log
	log.Ctx(ctx).Info().Msgf(
		"Moved repository %s to %s operation performed by %s",
		repo.Path, movedRepo.Path, session.Principal.Email)

	return GetRepoOutput(ctx, c.publicAccess, c.repoFinder, movedRepo)
}

func (c *Controller) MoveNoAuth(
	ctx context.Context,
	repo *types.Repository,
	newIdentifier *string,
	targetParentSpaceID int64,
	rootSpaceID int64,
	rootSpaceIdentifier string,
) (*types.Repository, error) {
	isPublic, err := c.publicAccess.Get(ctx, enum.PublicResourceTypeRepo, repo.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo public access: %w", err)
	}

	// remember the original root space so it can be restored if the move fails
	origRootSpaceID := repo.RootSpaceID
	origRootSpaceIdentifier := repo.RootSpaceIdentifier

	// the root space only changes on a cross-root move; a rename or a move within
	// the same root space leaves it untouched
	rootSpaceChanged := origRootSpaceID != rootSpaceID

	// remove public access from old repo path to avoid leaking it
	if err := c.publicAccess.Delete(
		ctx,
		enum.PublicResourceTypeRepo,
		repo.Path,
	); err != nil {
		return nil, fmt.Errorf("failed to remove public access on the original path: %w", err)
	}

	// TODO add a repo level lock here to avoid racing condition or partial repo update w/o setting repo public access
	//
	// Update the repo's identifier/parent and its root space columns (on the repo
	// and its pull requests) in one transaction, so a failure rolls back the whole
	// move. NB: RepoStore.Update doesn't touch the root space columns.
	var movedRepo *types.Repository
	if err := c.tx.WithTx(ctx, func(ctx context.Context) error {
		var err error
		movedRepo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
			if newIdentifier != nil {
				r.Identifier = *newIdentifier
			}
			if targetParentSpaceID != r.ParentID {
				r.ParentID = targetParentSpaceID
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repo: %w", err)
		}

		if rootSpaceChanged {
			if err := c.repoStore.UpdateRootSpace(
				ctx, []int64{movedRepo.ID}, rootSpaceID, rootSpaceIdentifier,
			); err != nil {
				return fmt.Errorf("failed to update repo root space: %w", err)
			}

			if err := c.pullReqStore.UpdateRootSpace(
				ctx, []int64{movedRepo.ID}, rootSpaceID, rootSpaceIdentifier,
			); err != nil {
				return fmt.Errorf("failed to update pull requests root space: %w", err)
			}
		}

		return nil
	}); err != nil {
		// public access isn't part of the transaction (its set on the new path can
		// only run post-commit, so both halves stay outside), so the rollback didn't
		// undo the delete above - manually restore it so we don't leave it stripped.
		if dErr := c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, isPublic); dErr != nil {
			return nil, fmt.Errorf(
				"failed to move repo (and restoring public access on the original path: %w): %w",
				dErr,
				err,
			)
		}
		return nil, err
	}

	movedRepo.RootSpaceID = rootSpaceID
	movedRepo.RootSpaceIdentifier = rootSpaceIdentifier

	// clear old repo from cache
	c.repoFinder.MarkChanged(ctx, repo.Core())

	// set public access for the new repo path
	if err := c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, movedRepo.Path, isPublic); err != nil {
		// ensure public access for new repo path is cleaned up first or we risk leaking it
		if dErr := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, movedRepo.Path); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and public access cleanup: %w): %w", dErr, err)
		}

		// revert identifier/parent and the root space of the repo and its pull
		// requests in a single transaction, or they'd keep the target's root
		// space while the repo lives under its old parent.
		var dErr error
		if dErr = c.tx.WithTx(ctx, func(ctx context.Context) error {
			movedRepo, dErr = c.repoStore.UpdateOptLock(ctx, movedRepo, func(r *types.Repository) error {
				r.Identifier = repo.Identifier
				r.ParentID = repo.ParentID
				return nil
			})
			if dErr != nil {
				return fmt.Errorf("failed to revert repo identifier and parent: %w", dErr)
			}

			// only revert the root space if the move actually changed it.
			if rootSpaceChanged {
				if err := c.repoStore.UpdateRootSpace(
					ctx, []int64{movedRepo.ID}, origRootSpaceID, origRootSpaceIdentifier,
				); err != nil {
					return fmt.Errorf("failed to revert repo root space: %w", err)
				}

				if err := c.pullReqStore.UpdateRootSpace(
					ctx, []int64{movedRepo.ID}, origRootSpaceID, origRootSpaceIdentifier,
				); err != nil {
					return fmt.Errorf("failed to revert pull requests root space: %w", err)
				}
			}

			return nil
		}); dErr != nil {
			return nil, fmt.Errorf(
				"failed to set public access for new path (and reverting of move: %w): %w",
				dErr,
				err,
			)
		}

		movedRepo.RootSpaceID = origRootSpaceID
		movedRepo.RootSpaceIdentifier = origRootSpaceIdentifier

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

	return movedRepo, nil
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
