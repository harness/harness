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

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	events "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
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

	var userGroupReviewer *types.UserGroupReviewer

	userGroupReviewer, err = addReviewerUserGroup(ctx, session, c, repo, pr, in)
	if err != nil {
		return nil, fmt.Errorf("failed to add usergroup_reviwer: %w", err)
	}

	c.reportUserGroupReviewerAddition(ctx, session, pr, userGroupReviewer)
	return userGroupReviewer, nil
}

func addReviewerUserGroup(
	ctx context.Context,
	session *auth.Session,
	c *Controller,
	repo *types.Repository,
	pr *types.PullReq,
	in *UserGroupReviewerAddInput,
) (*types.UserGroupReviewer, error) {
	addedByInfo := session.Principal.ToPrincipalInfo()

	var reviewerUserGroup *types.UserGroup
	reviewerUserGroup, err := c.userGroupStore.Find(ctx, in.UserGroupID)
	if err != nil {
		return nil, usererror.NotFound("failed to find usergroup")
	}

	userGroupReviewerInfo := reviewerUserGroup.ToUserGroupInfo()

	var userGroupReviewer *types.UserGroupReviewer
	userGroupReviewer, err = c.userGroupReviewerStore.Find(ctx, pr.ID, in.UserGroupID)
	if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
		return nil, usererror.NotFound("failed to find usergroup reviewer")
	}

	if userGroupReviewer != nil {
		addedByInfo, err = c.principalInfoCache.Get(ctx, userGroupReviewer.CreatedBy)
		if err != nil {
			return nil, fmt.Errorf("failed to get added by principal info: %w", err)
		}
		userGroupReviewer.AddedBy = *addedByInfo
		userGroupReviewer.UserGroup = *userGroupReviewerInfo
	}

	newUserGroupReviewer := newPullReqUserGroupReviewer(session, pr, repo, *userGroupReviewerInfo, addedByInfo, in)

	err = c.userGroupReviewerStore.Create(ctx, newUserGroupReviewer)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request reviewer: %w", err)
	}

	return userGroupReviewer, nil
}

func newPullReqUserGroupReviewer(
	session *auth.Session,
	pullReq *types.PullReq,
	repo *types.Repository,
	userGroupReviewerInfo types.UserGroupInfo,
	addedByInfo *types.PrincipalInfo,
	in *UserGroupReviewerAddInput,
) *types.UserGroupReviewer {
	now := time.Now().UnixMilli()
	reviewer := &types.UserGroupReviewer{
		PullReqID:   pullReq.ID,
		UserGroupID: in.UserGroupID,
		CreatedBy:   session.Principal.ID,
		Created:     now,
		Updated:     now,
		RepoID:      repo.ID,
		UserGroup:   userGroupReviewerInfo,
		AddedBy:     *addedByInfo,
	}
	return reviewer
}

func (c *Controller) reportUserGroupReviewerAddition(
	ctx context.Context,
	session *auth.Session,
	pr *types.PullReq,
	userGroupReviewer *types.UserGroupReviewer,
) {
	userGroupReviewerID := userGroupReviewer.UserGroupID
	c.eventReporter.UserGroupReviewerAdded(ctx, &events.UserGroupReviewerAddedPayload{
		Base:                eventBase(pr, &session.Principal),
		UserGroupReviewerID: userGroupReviewerID,
	})
}
