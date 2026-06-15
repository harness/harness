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
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) MergeQueueGet(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) (*types.MergeQueueInfo, error) {
	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.State != enum.PullReqStateOpen || pr.SubState != enum.PullReqSubStateMergeQueue {
		return nil, usererror.BadRequest("Pull request is not in merge queue.")
	}

	entry, err := c.mergeQueueEntryStore.Find(ctx, pr.ID)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil, usererror.BadRequest("Pull request is not in merge queue.")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find merge queue entry: %w", err)
	}

	repoLevelBranchRules, err := c.protectionManager.ListOnlyRepoBranchRules(ctx, targetRepo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch protection repo-level rules for the repository: %w", err)
	}

	mqSetup, err := repoLevelBranchRules.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         targetRepo,
		TargetBranch: pr.TargetBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	info, err := c.mergeQueueService.BuildMergeQueueInfo(ctx, targetRepo, entry, mqSetup)
	if err != nil {
		return nil, fmt.Errorf("failed to build merge queue info: %w", err)
	}

	return info, nil
}
