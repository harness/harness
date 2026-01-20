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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) AutoMergeDisable(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
) error {
	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	err = controller.TxOptLock(ctx, c.tx, func(ctx context.Context) error {
		pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
		if err != nil {
			return fmt.Errorf("failed to get pull request by number: %w", err)
		}

		err = verifyIfAutoMergeable(pr)
		if err != nil {
			return fmt.Errorf("pull request is not mergeable: %w", err)
		}

		if pr.SubState != enum.PullReqSubStateAutoMerge {
			return nil
		}

		pr.SubState = enum.PullReqSubStateNone

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		_, err = c.autoMergeStore.Delete(ctx, pr.ID)
		if err != nil {
			return fmt.Errorf("failed to update auto merge state: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to disable auto merge for the pull request: %w", err)
	}

	return nil
}
