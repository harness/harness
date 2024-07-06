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
	"database/sql"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type RestoreInput struct {
	NewIdentifier *string `json:"new_identifier,omitempty"`
	NewParentRef  *string `json:"new_parent_ref,omitempty"`
}

func (c *Controller) Restore(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	deletedAt int64,
	in *RestoreInput,
) (*RepositoryOutput, error) {
	repo, err := c.repoStore.FindByRefAndDeletedAt(ctx, repoRef, deletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	if repo.Deleted == nil {
		return nil, usererror.BadRequest("cannot restore a repo that hasn't been deleted")
	}

	parentID := repo.ParentID
	if in.NewParentRef != nil {
		space, err := c.spaceStore.FindByRef(ctx, *in.NewParentRef)
		if errors.Is(err, store.ErrResourceNotFound) {
			return nil, usererror.BadRequest("The provided new parent ref wasn't found.")
		}
		if err != nil {
			return nil, fmt.Errorf("failed to find the parent ref '%s': %w", *in.NewParentRef, err)
		}

		parentID = space.ID
	}

	return c.RestoreNoAuth(ctx, repo, in.NewIdentifier, parentID)
}

func (c *Controller) RestoreNoAuth(
	ctx context.Context,
	repo *types.Repository,
	newIdentifier *string,
	newParentID int64,
) (*RepositoryOutput, error) {
	var err error
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		if err := c.resourceLimiter.RepoCount(ctx, newParentID, 1); err != nil {
			return fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
		}

		repo, err = c.repoStore.Restore(ctx, repo, newIdentifier, &newParentID)
		if err != nil {
			return fmt.Errorf("failed to restore the repo: %w", err)
		}

		return nil
	}, sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("failed to restore the repo: %w", err)
	}

	// Repos restored as private since public access data has been deleted upon deletion.
	return GetRepoOutputWithAccess(ctx, false, repo), nil
}
