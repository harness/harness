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
	"github.com/harness/gitness/app/services/dotrange"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Find returns a pull request from the provided repository.
func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	options types.PullReqMetadataOptions,
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

	if err := c.pullreqListService.BackfillMetadataForPullReq(ctx, repo, pr, options); err != nil {
		return nil, fmt.Errorf("failed to backfill pull request metadata: %w", err)
	}

	return pr, nil
}

// FindByBranches returns a pull request from the provided branch pair.
func (c *Controller) FindByBranches(
	ctx context.Context,
	session *auth.Session,
	repoRef,
	sourceBranch,
	targetBranch string,
	options types.PullReqMetadataOptions,
) (*types.PullReq, error) {
	if sourceBranch == "" || targetBranch == "" {
		return nil, usererror.BadRequest("A valid source/target branch must be provided.")
	}

	dotRange, err := dotrange.Make(targetBranch, sourceBranch, true)
	if err != nil {
		return nil, fmt.Errorf("failed to make dot range: %w", err)
	}

	if dotRange.HeadUpstream {
		return nil, usererror.BadRequest("Source branch can't be on the upstream repository.")
	}

	sourceRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
	}

	targetRepo := sourceRepo

	if dotRange.BaseUpstream {
		if sourceRepo.ForkID == 0 {
			return nil,
				usererror.BadRequest("The repository is not a fork repository, so upstream can't be used.")
		}

		targetRepo, err = c.repoFinder.FindByID(ctx, sourceRepo.ForkID)
		if err != nil {
			return nil, fmt.Errorf("failed to find upstream repo: %w", err)
		}
	}

	prs, err := c.pullreqStore.List(ctx, &types.PullReqFilter{
		SourceRepoID: sourceRepo.ID,
		SourceBranch: dotRange.HeadRef,
		TargetRepoID: targetRepo.ID,
		TargetBranch: dotRange.BaseRef,
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

	if err := c.pullreqListService.BackfillMetadataForPullReq(ctx, targetRepo, prs[0], options); err != nil {
		return nil, fmt.Errorf("failed to backfill pull request metadata: %w", err)
	}

	return prs[0], nil
}
