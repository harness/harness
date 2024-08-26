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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CombinedListResponse struct {
	Reviewers          []*types.PullReqReviewer   `json:"reviewers"`
	UserGroupReviewers []*types.UserGroupReviewer `json:"user_group_reviewers"`
}

// ReviewersListCombined returns the combined reviewer list for the pull request.
func (c *Controller) ReviewersListCombined(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) (*CombinedListResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
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
	reviewersMap := createReviewerMap(reviewers)

	userGroupReviewers, err := c.userGroupReviewerStore.List(ctx, pr.ID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to list user group reviewers: %w", err)
	}

	for _, userGroupReviewer := range userGroupReviewers {
		userGroup, err := c.userGroupStore.Find(ctx, userGroupReviewer.UserGroupID)
		if err != nil {
			return nil, fmt.Errorf("failed to find user group: %w", err)
		}

		usersUIDs, err := c.userGroupService.ListUsers(ctx, session, userGroup)
		if err != nil {
			return nil, fmt.Errorf("failed to find users belonging: %w", err)
		}
		if usersUIDs == nil {
			continue
		}
		users, err := c.principalStore.FindManyByUID(ctx, usersUIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to find user group users: %w", err)
		}
		if len(users) == 0 {
			continue
		}
		var userGroupReviewerDecisions []types.UserGroupReviewerDecision
		for _, user := range users {
			if reviewer, ok := reviewersMap[user.ID]; ok {
				userGroupReviewerDecisions = append(userGroupReviewerDecisions, types.UserGroupReviewerDecision{
					ReviewDecision: reviewer.ReviewDecision,
					SHA:            reviewer.SHA,
					Reviewer:       reviewer.Reviewer,
				})
			}
		}
		userGroupReviewer.Reviewers = userGroupReviewerDecisions
	}

	return &CombinedListResponse{
		Reviewers:          reviewers,
		UserGroupReviewers: userGroupReviewers,
	}, nil
}

func createReviewerMap(reviewers []*types.PullReqReviewer) map[int64]*types.PullReqReviewer {
	reviewerMap := make(map[int64]*types.PullReqReviewer)
	for _, reviewer := range reviewers {
		reviewerMap[reviewer.PrincipalID] = reviewer
	}
	return reviewerMap
}
