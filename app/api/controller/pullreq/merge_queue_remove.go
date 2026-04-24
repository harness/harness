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
	"github.com/harness/gitness/app/services/mergequeue"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) MergeQueueRemove(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) error {
	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return fmt.Errorf("failed to get pull request by number: %w", err)
	}

	err = c.mergeQueueService.Remove(ctx, pr.ID, enum.MergeQueueRemovalReasonManual)
	if errors.Is(err, mergequeue.ErrNotInQueue) {
		return usererror.BadRequest("Pull request is not in merge queue.")
	}
	if err != nil {
		return fmt.Errorf("failed to remove pull request from merge queue: %w", err)
	}

	c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullReqMergeQueueRemoved, pr)

	return nil
}
