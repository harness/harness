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
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
}

func (i *MoveInput) hasChanges(repo *types.Repository) bool {
	if i.Identifier != nil && *i.Identifier != repo.Identifier {
		return true
	}

	return false
}

// Move moves a repository to a new identifier.
// TODO: Add support for moving to other parents and aliases.
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

	if !in.hasChanges(repo) {
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
	renamedRepo, err := c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		if in.Identifier != nil {
			r.Identifier = *in.Identifier
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update repo: %w", err)
	}

	// clear old repo from cache
	c.repoFinder.MarkChanged(ctx, repo.Core())

	// set public access for the new repo path
	if err := c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, renamedRepo.Path, isPublic); err != nil {
		// ensure public access for new repo path is cleaned up first or we risk leaking it
		if dErr := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, renamedRepo.Path); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and public access cleanup: %w): %w", dErr, err)
		}

		// revert identifier changes first
		var dErr error
		_, dErr = c.repoStore.UpdateOptLock(ctx, renamedRepo, func(r *types.Repository) error {
			r.Identifier = repo.Identifier
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
		c.repoFinder.MarkChanged(ctx, renamedRepo.Core())

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

	renamedRepo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, renamedRepo.Path)
	renamedRepo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, renamedRepo.Path)

	return GetRepoOutput(ctx, c.publicAccess, renamedRepo)
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

	return nil
}
