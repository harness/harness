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
	"math"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Purge deletes the space and all its subspaces and repositories permanently.
func (c *Controller) Purge(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	deletedAt int64,
) error {
	space, err := c.spaceStore.FindByRefAndDeletedAt(ctx, spaceRef, deletedAt)
	if err != nil {
		return err
	}

	// authz will check the permission within the first existing parent since space was deleted.
	// purge top level space is limited to admin only.
	err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceDelete)
	if err != nil {
		return fmt.Errorf("failed to authorize on space purge: %w", err)
	}

	return c.PurgeNoAuth(ctx, session, space)
}

// PurgeNoAuth purges the space - no authorization is verified.
// WARNING For internal calls only.
func (c *Controller) PurgeNoAuth(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
) error {
	// the max time we give a purge space to succeed
	const timeout = 15 * time.Minute
	// create new, time-restricted context to guarantee space purge completion, even if request is canceled.
	ctx, cancel := context.WithTimeout(
		contextutil.WithNewValues(context.Background(), ctx),
		timeout,
	)
	defer cancel()

	var toBeDeletedRepos []*types.Repository
	var err error
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		toBeDeletedRepos, err = c.purgeSpaceInnerInTx(ctx, space.ID, *space.Deleted)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to purge space %d in a tnx: %w", space.ID, err)
	}

	// permanently purge all repositories in the space and its subspaces after successful space purge tnx.
	// cleanup will handle failed repository deletions.
	for _, repo := range toBeDeletedRepos {
		err := c.repoCtrl.DeleteGitRepository(ctx, session, repo.GitUID)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).
				Str("repo_identifier", repo.Identifier).
				Int64("repo_id", repo.ID).
				Int64("repo_parent_id", repo.ParentID).
				Msg("failed to delete repository")
		}
	}

	return nil
}

func (c *Controller) purgeSpaceInnerInTx(
	ctx context.Context,
	spaceID int64,
	deletedAt int64,
) ([]*types.Repository, error) {
	filter := &types.RepoFilter{
		Page:              1,
		Size:              int(math.MaxInt),
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.RepoAttrDeleted,
		DeletedBeforeOrAt: &deletedAt,
		Recursive:         true,
	}
	repos, err := c.repoStore.List(ctx, spaceID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list space repositories: %w", err)
	}

	// purge cascade deletes all the child spaces from DB.
	err = c.spaceStore.Purge(ctx, spaceID, &deletedAt)
	if err != nil {
		return nil, fmt.Errorf("spaceStore failed to delete space %d: %w", spaceID, err)
	}

	return repos, nil
}
