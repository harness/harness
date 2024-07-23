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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
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
	if err := c.sanitizeMoveInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if repo.State != enum.RepoStateActive {
		return nil, usererror.BadRequest("Can't move a repo that isn't ready to use.")
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit); err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return GetRepoOutput(ctx, c.publicAccess, repo)
	}

	oldIdentifier := repo.Identifier

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
	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		if in.Identifier != nil {
			r.Identifier = *in.Identifier
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update repo: %w", err)
	}

	// set public access for the new repo path
	if err := c.publicAccess.Set(ctx, enum.PublicResourceTypeRepo, repo.Path, isPublic); err != nil {
		// ensure public access for new repo path is cleaned up first or we risk leaking it
		if dErr := c.publicAccess.Delete(ctx, enum.PublicResourceTypeRepo, repo.Path); dErr != nil {
			return nil, fmt.Errorf("failed to set repo public access (and public access cleanup: %w): %w", dErr, err)
		}

		// revert identifier changes first
		var dErr error
		repo, dErr = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
			r.Identifier = oldIdentifier
			return nil
		})
		if dErr != nil {
			return nil, fmt.Errorf(
				"failed to set public access for new path (and reverting of move: %w): %w",
				dErr,
				err,
			)
		}

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

	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	return GetRepoOutput(ctx, c.publicAccess, repo)
}

func (c *Controller) sanitizeMoveInput(in *MoveInput) error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := c.identifierCheck(*in.Identifier); err != nil {
			return err
		}
	}

	return nil
}
