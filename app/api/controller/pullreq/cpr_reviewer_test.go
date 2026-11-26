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
	"net/http"
	"testing"

	apiauth "github.com/harness/gitness/app/api/auth"
	pullreqsvc "github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func (e *cprEnv) addReviewer(
	pr *types.PullReq,
	user *types.User,
	decision enum.PullReqReviewDecision,
) {
	e.t.Helper()

	reviewer := pullreqsvc.NewPullReqReviewer(
		pr, e.repo.Core(), user.ToPrincipalInfo(), e.author.ToPrincipalInfo(),
		enum.PullReqReviewerTypeRequested, user.ID)
	reviewer.ReviewDecision = decision
	require.NoError(e.t, e.reviewerStore.Create(e.ctx, reviewer))
}

func TestCprReviewerAdd(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()

	t.Run("requested by author", func(t *testing.T) {
		reviewer, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.NoError(t, err)
		require.Equal(t, e.reviewer.ID, reviewer.PrincipalID)
		require.Equal(t, enum.PullReqReviewerTypeRequested, reviewer.Type)
		require.Contains(t, e.streamer.published, enum.SSETypePullReqReviewerAdded)
	})

	t.Run("adding twice is idempotent", func(t *testing.T) {
		reviewer, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.NoError(t, err)
		require.Equal(t, e.reviewer.ID, reviewer.PrincipalID)

		list, err := e.reviewerStore.List(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Len(t, list, 1)
	})

	t.Run("self assigned", func(t *testing.T) {
		reviewer, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.other), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: e.other.ID})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqReviewerTypeSelfAssigned, reviewer.Type)
	})

	t.Run("assigned by third party", func(t *testing.T) {
		pr2 := e.createPR()
		reviewer, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.other), e.repoRef(), pr2.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.NoError(t, err)
		require.Equal(t, enum.PullReqReviewerTypeAssigned, reviewer.Type)
	})

	t.Run("author can't be a reviewer", func(t *testing.T) {
		_, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: e.author.ID})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reviewer id is required", func(t *testing.T) {
		_, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&ReviewerAddInput{})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("merged pull request", func(t *testing.T) {
		merged := e.createPR(func(p *types.PullReq) {
			now := int64(10)
			p.Merged = &now
			p.State = enum.PullReqStateMerged
		})
		_, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), merged.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("unknown reviewer", func(t *testing.T) {
		_, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: 4242})
		require.Error(t, err)
	})

	t.Run("reviewer without repo access", func(t *testing.T) {
		pr3 := e.createPR()
		ctrl := *e.ctrl
		ctrl.authorizer = cprSelectiveAuthorizer{denied: map[int64]bool{e.reviewer.ID: true}}
		ctrl.pullreqService = e.newPullReqServiceFor(&ctrl)
		_, err := ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr3.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, err := e.ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), 4242,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.ReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&ReviewerAddInput{ReviewerID: e.reviewer.ID})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprReviewerList(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionPending)
	e.addReviewer(pr, e.other, enum.PullReqReviewDecisionChangeReq)

	t.Run("sorted by decision", func(t *testing.T) {
		list, err := e.ctrl.ReviewerList(e.ctx, e.session(e.author), e.repoRef(), pr.Number)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, enum.PullReqReviewDecisionChangeReq, list[0].ReviewDecision)
		require.Equal(t, enum.PullReqReviewDecisionPending, list[1].ReviewDecision)
	})

	t.Run("errors", func(t *testing.T) {
		_, err := e.ctrl.ReviewerList(e.ctx, e.session(e.author), e.repoRef(), 4242)
		require.Error(t, err)

		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err = ctrl.ReviewerList(e.ctx, e.session(e.author), e.repoRef(), pr.Number)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprReviewDecisionRank(t *testing.T) {
	t.Parallel()

	tests := []struct {
		decision enum.PullReqReviewDecision
		want     int
	}{
		{enum.PullReqReviewDecisionChangeReq, 0},
		{enum.PullReqReviewDecisionApproved, 1},
		{enum.PullReqReviewDecisionReviewed, 2},
		{enum.PullReqReviewDecisionPending, 3},
		{"other", 4},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, reviewDecisionRank(tt.decision), tt.decision)
	}
}

func TestCprSortUserGroupReviewersByDecision(t *testing.T) {
	t.Parallel()

	reviewers := []*types.UserGroupReviewer{
		{UserGroupID: 1, Decision: enum.PullReqReviewDecisionPending},
		{UserGroupID: 2, Decision: enum.PullReqReviewDecisionApproved},
		{UserGroupID: 3, Decision: enum.PullReqReviewDecisionChangeReq},
		{UserGroupID: 4, Decision: enum.PullReqReviewDecisionApproved},
	}
	sortUserGroupReviewersByDecision(reviewers)

	ids := make([]int64, len(reviewers))
	for i, r := range reviewers {
		ids[i] = r.UserGroupID
	}
	require.Equal(t, []int64{3, 2, 4, 1}, ids)
}

func TestCprReviewerDelete(t *testing.T) {
	e := cprSetup(t)

	t.Run("pending reviewer removed by pusher", func(t *testing.T) {
		pr := e.createPR()
		e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionPending)

		require.NoError(t, e.ctrl.ReviewerDelete(
			e.ctx, e.session(e.author), e.repoRef(), pr.Number, e.reviewer.ID))

		list, err := e.reviewerStore.List(e.ctx, pr.ID)
		require.NoError(t, err)
		require.Empty(t, list)

		acts, err := e.activityStore.List(e.ctx, pr.ID, &types.PullReqActivityFilter{})
		require.NoError(t, err)
		var found bool
		for _, a := range acts {
			found = found || a.Type == enum.PullReqActivityTypeReviewerDelete
		}
		require.True(t, found)
	})

	t.Run("own review", func(t *testing.T) {
		pr := e.createPR()
		e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionApproved)
		require.NoError(t, e.ctrl.ReviewerDelete(
			e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number, e.reviewer.ID))
	})

	t.Run("submitted review needs edit permission", func(t *testing.T) {
		pr := e.createPR()
		e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionApproved)
		require.NoError(t, e.ctrl.ReviewerDelete(
			e.ctx, e.session(e.author), e.repoRef(), pr.Number, e.reviewer.ID))
	})

	t.Run("denied", func(t *testing.T) {
		pr := e.createPR()
		e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionPending)
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		err := ctrl.ReviewerDelete(e.ctx, e.session(e.author), e.repoRef(), pr.Number, e.reviewer.ID)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("merged pull request", func(t *testing.T) {
		pr := e.createPR(func(p *types.PullReq) {
			now := int64(10)
			p.Merged = &now
			p.State = enum.PullReqStateMerged
		})
		e.addReviewer(pr, e.reviewer, enum.PullReqReviewDecisionPending)
		err := e.ctrl.ReviewerDelete(e.ctx, e.session(e.author), e.repoRef(), pr.Number, e.reviewer.ID)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("reviewer not found", func(t *testing.T) {
		pr := e.createPR()
		require.Error(t, e.ctrl.ReviewerDelete(e.ctx, e.session(e.author), e.repoRef(), pr.Number, 4242))
	})

	t.Run("pull request not found", func(t *testing.T) {
		require.Error(t, e.ctrl.ReviewerDelete(e.ctx, e.session(e.author), e.repoRef(), 4242, e.reviewer.ID))
	})

	t.Run("repo ref is required", func(t *testing.T) {
		err := e.ctrl.ReviewerDelete(e.ctx, e.session(e.author), "", 1, e.reviewer.ID)
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})
}

func (e *cprEnv) createUserGroup(identifier string) *types.UserGroup {
	e.t.Helper()

	group := &types.UserGroup{Identifier: identifier, Name: identifier, SpaceID: e.space.ID}
	require.NoError(e.t, e.ctrl.userGroupStore.Create(e.ctx, e.space.ID, group))

	return group
}

func TestCprUserGroupReviewers(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	group := e.createUserGroup("cpr_group")

	t.Run("add", func(t *testing.T) {
		got, err := e.ctrl.UserGroupReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&UserGroupReviewerAddInput{UserGroupID: group.ID})
		require.NoError(t, err)
		require.Equal(t, group.ID, got.UserGroupID)
		require.Equal(t, group.Identifier, got.UserGroup.Identifier)
		require.Equal(t, e.author.ID, got.AddedBy.ID)
	})

	t.Run("add again returns existing", func(t *testing.T) {
		got, err := e.ctrl.UserGroupReviewerAdd(e.ctx, e.session(e.reviewer), e.repoRef(), pr.Number,
			&UserGroupReviewerAddInput{UserGroupID: group.ID})
		require.NoError(t, err)
		require.Equal(t, group.ID, got.UserGroupID)
		require.Equal(t, e.author.ID, got.AddedBy.ID)
	})

	t.Run("unknown group", func(t *testing.T) {
		_, err := e.ctrl.UserGroupReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&UserGroupReviewerAddInput{UserGroupID: 4242})
		require.Error(t, err)
	})

	t.Run("unknown pull request", func(t *testing.T) {
		_, err := e.ctrl.UserGroupReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), 4242,
			&UserGroupReviewerAddInput{UserGroupID: group.ID})
		require.Error(t, err)
	})

	t.Run("add denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, err := ctrl.UserGroupReviewerAdd(e.ctx, e.session(e.author), e.repoRef(), pr.Number,
			&UserGroupReviewerAddInput{UserGroupID: group.ID})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, e.ctrl.UserGroupReviewerDelete(
			e.ctx, e.session(e.author), e.repoRef(), pr.Number, group.ID))

		_, err := e.ctrl.userGroupReviewerStore.Find(e.ctx, pr.ID, group.ID)
		require.Error(t, err)
	})

	t.Run("delete errors", func(t *testing.T) {
		require.Error(t, e.ctrl.UserGroupReviewerDelete(
			e.ctx, e.session(e.author), e.repoRef(), 4242, group.ID))
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		err := ctrl.UserGroupReviewerDelete(e.ctx, e.session(e.author), e.repoRef(), pr.Number, group.ID)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
	})
}

func TestCprReviewerSuggestions(t *testing.T) {
	e := cprSetup(t)
	pr := e.createPR()
	session := e.session(e.author)

	t.Run("batch creates suggestions and list shows them", func(t *testing.T) {
		_, err := e.ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), pr.Number,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{e.reviewer.ID, e.other.ID, e.other.ID}})
		require.NoError(t, err)

		out, count, err := e.ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), pr.Number,
			types.Pagination{Page: 1, Size: 10})
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
		require.Len(t, out.Suggestions, 2)
		require.Equal(t, e.author.ID, out.Suggestions[0].SuggestedBy.ID)
	})

	t.Run("paginated list counts through the store", func(t *testing.T) {
		out, count, err := e.ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), pr.Number,
			types.Pagination{Page: 1, Size: 1})
		require.NoError(t, err)
		require.EqualValues(t, 2, count)
		require.Len(t, out.Suggestions, 1)
	})

	t.Run("empty list", func(t *testing.T) {
		empty := e.createPR()
		out, count, err := e.ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), empty.Number,
			types.Pagination{Page: 1, Size: 10})
		require.NoError(t, err)
		require.Zero(t, count)
		require.Empty(t, out.Suggestions)
	})

	t.Run("apply adds reviewer and removes suggestion", func(t *testing.T) {
		reviewer, err := e.ctrl.ReviewerSuggestApply(e.ctx, session, e.repoRef(), pr.Number, e.reviewer.ID)
		require.NoError(t, err)
		require.Equal(t, e.reviewer.ID, reviewer.PrincipalID)

		_, count, err := e.ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), pr.Number,
			types.Pagination{Page: 1, Size: 10})
		require.NoError(t, err)
		require.EqualValues(t, 1, count)
	})

	t.Run("apply unknown suggestion", func(t *testing.T) {
		_, err := e.ctrl.ReviewerSuggestApply(e.ctx, session, e.repoRef(), pr.Number, 4242)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("delete suggestion", func(t *testing.T) {
		require.NoError(t, e.ctrl.ReviewerSuggestDelete(e.ctx, session, e.repoRef(), pr.Number, e.other.ID))
		err := e.ctrl.ReviewerSuggestDelete(e.ctx, session, e.repoRef(), pr.Number, e.other.ID)
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("batch skips existing reviewers", func(t *testing.T) {
		_, err := e.ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), pr.Number,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{e.reviewer.ID}})
		require.NoError(t, err)
	})

	t.Run("batch rejects the author", func(t *testing.T) {
		_, err := e.ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), pr.Number,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{e.author.ID}})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("batch rejects unknown principal", func(t *testing.T) {
		_, err := e.ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), pr.Number,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{4242}})
		require.Equal(t, http.StatusNotFound, cprStatus(err))
	})

	t.Run("batch rejects reviewer without access", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprSelectiveAuthorizer{denied: map[int64]bool{e.other.ID: true}}
		_, err := ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), pr.Number,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{e.other.ID}})
		require.Equal(t, http.StatusBadRequest, cprStatus(err))
	})

	t.Run("pull request not found", func(t *testing.T) {
		_, _, err := e.ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), 4242, types.Pagination{})
		require.Error(t, err)
		_, err = e.ctrl.ReviewerSuggestApply(e.ctx, session, e.repoRef(), 4242, 1)
		require.Error(t, err)
		require.Error(t, e.ctrl.ReviewerSuggestDelete(e.ctx, session, e.repoRef(), 4242, 1))
		_, err = e.ctrl.ReviewerSuggestBatch(e.ctx, session, e.repoRef(), 4242,
			&pullreqsvc.ReviewerSuggestBatchInput{ReviewerIDs: []int64{e.other.ID}})
		require.Error(t, err)
	})

	t.Run("denied", func(t *testing.T) {
		ctrl := *e.ctrl
		ctrl.authorizer = cprDenyAuthorizer()
		_, _, err := ctrl.ListSuggestedReviewers(e.ctx, session, e.repoRef(), pr.Number, types.Pagination{})
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		_, err = ctrl.ReviewerSuggestApply(e.ctx, session, e.repoRef(), pr.Number, 1)
		require.ErrorIs(t, err, apiauth.ErrForbidden)
		require.ErrorIs(t, ctrl.ReviewerSuggestDelete(e.ctx, session, e.repoRef(), pr.Number, 1), apiauth.ErrForbidden)
	})
}
