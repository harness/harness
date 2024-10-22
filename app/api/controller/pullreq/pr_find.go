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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Find returns a pull request from the provided repository.
func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (*types.PullReq, error) {
	if pullreqNum <= 0 {
		return nil, usererror.BadRequest("A valid pull request number must be provided.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return nil, err
	}

	err = c.labelSvc.Backfill(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to backfill labels assigned to pull request: %w", err)
	}

	if err := c.pullreqListService.BackfillStats(ctx, pr); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to backfill PR stats")
	}

	return pr, nil
}

// Find returns a pull request from the provided repository.
func (c *Controller) FindByBranches(
	ctx context.Context,
	session *auth.Session,
	repoRef,
	sourceRepoRef,
	sourceBranch,
	targetBranch string,
) (*types.PullReq, error) {
	if sourceBranch == "" || targetBranch == "" {
		return nil, usererror.BadRequest("A valid source/target branch must be provided.")
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to the repo: %w", err)
	}

	sourceRepo := targetRepo
	if sourceRepoRef != repoRef {
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, sourceRepoRef, enum.PermissionRepoPush)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	prs, err := c.pullreqStore.List(ctx, &types.PullReqFilter{
		SourceRepoID: sourceRepo.ID,
		SourceBranch: sourceBranch,
		TargetRepoID: targetRepo.ID,
		TargetBranch: targetBranch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Size:         1,
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch existing pull request: %w", err)
	}

	if len(prs) == 0 {
		return nil, usererror.ErrNotFound
	}

	return prs[0], nil
}
