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
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type SoftDeleteResponse struct {
	DeletedAt int64 `json:"deleted_at"`
}

// SoftDelete soft deletes a repo and returns the deletedAt timestamp in epoch format.
func (c *Controller) SoftDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) (*SoftDeleteResponse, error) {
	// note: can't use c.getRepoCheckAccess because import job for repositories being imported must be cancelled.
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find the repo for soft delete: %w", err)
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoDelete, false); err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	log.Ctx(ctx).Info().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Msg("soft deleting repository")

	if repo.Deleted != nil {
		return nil, usererror.BadRequest("repository has been already deleted")
	}

	if repo.Importing {
		log.Ctx(ctx).Info().Msg("repository is importing. cancelling the import job and purge the repo.")
		err = c.importer.Cancel(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to cancel repository import")
		}
		return nil, c.PurgeNoAuth(ctx, session, repo)
	}

	now := time.Now().UnixMilli()
	if err = c.SoftDeleteNoAuth(ctx, repo, now); err != nil {
		return nil, fmt.Errorf("failed to soft delete repo: %w", err)
	}

	return &SoftDeleteResponse{DeletedAt: now}, nil
}

func (c *Controller) SoftDeleteNoAuth(
	ctx context.Context,
	repo *types.Repository,
	deletedAt int64,
) error {
	err := c.repoStore.SoftDelete(ctx, repo, deletedAt)
	if err != nil {
		return fmt.Errorf("failed to soft delete repo from db: %w", err)
	}

	return nil
}
