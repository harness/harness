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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// List returns a list of pull requests from the provided repository.
func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.PullReqFilter,
) ([]*types.PullReq, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	if filter.SourceRepoRef == repoRef {
		filter.SourceRepoID = repo.ID
	} else if filter.SourceRepoRef != "" {
		var sourceRepo *types.Repository
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, filter.SourceRepoRef, enum.PermissionRepoView)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
		filter.SourceRepoID = sourceRepo.ID
	}

	var list []*types.PullReq
	var count int64

	filter.TargetRepoID = repo.ID

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		list, err = c.pullreqStore.List(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list pull requests: %w", err)
		}

		err := c.labelSvc.BackfillMany(ctx, list)
		if err != nil {
			return fmt.Errorf("failed to backfill labels assigned to pull requests: %w", err)
		}

		if filter.Page == 1 && len(list) < filter.Size {
			count = int64(len(list))
			return nil
		}

		count, err = c.pullreqStore.Count(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to count pull requests: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	for _, pr := range list {
		if err := c.backfillStats(ctx, repo, pr); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to backfill PR stats")
		}
	}

	return list, count, nil
}
