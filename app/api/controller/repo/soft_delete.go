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
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
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

	if repo.Deleted != nil {
		return nil, usererror.BadRequest("repository has been already deleted")
	}

	log.Ctx(ctx).Info().
		Int64("repo.id", repo.ID).
		Str("repo.path", repo.Path).
		Msg("soft deleting repository")

	now := time.Now().UnixMilli()
	if err = c.SoftDeleteNoAuth(ctx, session, repo, now); err != nil {
		return nil, fmt.Errorf("failed to soft delete repo: %w", err)
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeRepository, repo.Identifier),
		audit.ActionDeleted,
		paths.Parent(repo.Path),
		audit.WithOldObject(repo),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete repository operation: %s", err)
	}

	return &SoftDeleteResponse{DeletedAt: now}, nil
}

func (c *Controller) SoftDeleteNoAuth(
	ctx context.Context,
	session *auth.Session,
	repo *types.Repository,
	deletedAt int64,
) error {
	if repo.Importing {
		return c.PurgeNoAuth(ctx, session, repo)
	}

	err := c.repoStore.SoftDelete(ctx, repo, deletedAt)
	if err != nil {
		return fmt.Errorf("failed to soft delete repo from db: %w", err)
	}

	return nil
}
