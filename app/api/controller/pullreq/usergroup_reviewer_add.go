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
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type UserGroupReviewerAddInput struct {
	UserGroupID int64 `json:"usergroup_id"`
}

// UserGroupReviewerAdd adds a new usergroup to the pull request in the usergroups db.
func (c *Controller) UserGroupReviewerAdd(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	in *UserGroupReviewerAddInput,
) (*types.UserGroupReviewer, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoReview)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	reviewers, err := c.reviewerStore.List(ctx, pr.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to list reviewers: %w", err)
	}

	userGroup, err := c.userGroupStore.Find(ctx, in.UserGroupID)
	if err != nil {
		return nil, fmt.Errorf("failed to find user group: %w", err)
	}
	userIDs, err := c.userGroupService.ListUserIDsByGroupIDs(ctx, []int64{userGroup.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to list user ids by group id: %w", err)
	}

	reviewersMap := reviewersMap(reviewers)
	decision := enum.PullReqReviewDecisionPending
	for _, userGroupID := range userIDs {
		if reviewer, ok := reviewersMap[userGroupID]; ok {
			decision = getHighestOrderDecision(decision, reviewer.ReviewDecision)
		}
	}

	userGroupReviewer, err := c.userGroupReviewerStore.Find(ctx, pr.ID, in.UserGroupID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to find user group reviewer: %w", err)
	}

	if userGroupReviewer != nil {
		addedBy, err := c.principalInfoCache.Get(ctx, userGroupReviewer.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to get added by principal info: %w", err)
		}

		userGroupReviewer.AddedBy = *addedBy
		userGroupReviewer.UserGroup = *userGroup.ToUserGroupInfo()

		userGroupReviewer.Decision = decision

		return userGroupReviewer, nil
	}

	now := time.Now().UnixMilli()
	userGroupReviewer = &types.UserGroupReviewer{
		PullReqID:   pr.ID,
		UserGroupID: in.UserGroupID,
		CreatedBy:   session.Principal.ID,
		Created:     now,
		Updated:     now,
		RepoID:      repo.ID,
		UserGroup:   *userGroup.ToUserGroupInfo(),
		AddedBy:     *session.Principal.ToPrincipalInfo(),
		Decision:    decision,
	}

	err = c.userGroupReviewerStore.Create(ctx, userGroupReviewer)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request reviewer: %w", err)
	}

	c.reportUserGroupReviewerAdded(ctx, &session.Principal, pr, userGroupReviewer.UserGroupID)

	err = func() error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to increment pull request activity sequence: %w", err)
		}

		payload := &types.PullRequestActivityPayloadUserGroupReviewerAdd{
			UserGroupIDs: []int64{userGroupReviewer.UserGroupID},
			ReviewerType: enum.PullReqReviewerTypeRequested,
		}

		if _, err := c.activityStore.CreateWithPayload(
			ctx, pr, session.Principal.ID, payload, nil,
		); err != nil {
			return err
		}

		return nil
	}()
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf(
			"failed to write pull request activity after reviewer addition: %s", err,
		)
	}

	return userGroupReviewer, nil
}

func (c *Controller) reportUserGroupReviewerAdded(
	ctx context.Context,
	principal *types.Principal,
	pr *types.PullReq,
	userGroupReviewerID int64,
) {
	c.eventReporter.UserGroupReviewerAdded(
		ctx,
		&events.UserGroupReviewerAddedPayload{
			Base:                eventBase(pr, principal),
			UserGroupReviewerID: userGroupReviewerID,
		},
	)
}

func reviewersMap(reviewers []*types.PullReqReviewer) map[int64]*types.PullReqReviewer {
	reviewersMap := make(map[int64]*types.PullReqReviewer, len(reviewers))
	for _, reviewer := range reviewers {
		reviewersMap[reviewer.PrincipalID] = reviewer
	}

	return reviewersMap
}
