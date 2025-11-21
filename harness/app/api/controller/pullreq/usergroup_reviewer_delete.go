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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// UserGroupReviewerDelete deletes reviewer from the UserGroupReviewers store for the given PR.
func (c *Controller) UserGroupReviewerDelete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum,
	userGroupID int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to find pull request: %w", err)
	}

	err = c.userGroupReviewerStore.Delete(ctx, pr.ID, userGroupID)
	if err != nil {
		return fmt.Errorf("failed to delete user group reviewer: %w", err)
	}

	err = func() error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to increment pull request activity sequence: %w", err)
		}

		payload := &types.PullRequestActivityPayloadUserGroupReviewerDelete{
			UserGroupIDs: []int64{userGroupID},
		}

		metadata := &types.PullReqActivityMetadata{
			Mentions: &types.PullReqActivityMentionsMetadata{UserGroupIDs: []int64{userGroupID}},
		}

		_, err = c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload, metadata)
		if err != nil {
			return fmt.Errorf("failed to create pull request activity: %w", err)
		}

		return nil
	}()
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to write pull request activity after user group reviewer removal")
	}
	return err
}
