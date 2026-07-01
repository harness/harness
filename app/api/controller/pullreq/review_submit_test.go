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
	"testing"

	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	reviewSubmitRepoID      int64 = 1
	reviewSubmitParentID    int64 = 10
	reviewSubmitPRID        int64 = 55
	reviewSubmitPRNum       int64 = 7
	reviewSubmitAuthorID    int64 = 99  // PR creator
	reviewSubmitReviewerID  int64 = 100 // testSession principal
	reviewSubmitResolvedSHA       = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
)

// noopInstrumentation swallows Track calls so ReviewSubmit's post-tx
// instrumentation doesn't need a live analytics backend.
type noopInstrumentation struct{}

func (noopInstrumentation) Track(context.Context, instrument.Event) error { return nil }
func (noopInstrumentation) Close(context.Context) error                   { return nil }

// reviewSubmitFixture bundles the mocks a ReviewSubmit test cares about so a
// single-line assertion at the end of the test can verify every expectation.
type reviewSubmitFixture struct {
	ctrl          *Controller
	tx            *txAssertStub
	pullreqStore  *mockstore.PullReqStore
	reviewStore   *mockstore.PullReqReviewStore
	reviewerStore *mockstore.PullReqReviewerStore
	activityStore *mockstore.PullReqActivityStore
	gitClient     *mockgit.Interface
}

func (f *reviewSubmitFixture) assertExpectations(t *testing.T) {
	t.Helper()
	f.pullreqStore.AssertExpectations(t)
	f.reviewStore.AssertExpectations(t)
	f.reviewerStore.AssertExpectations(t)
	f.activityStore.AssertExpectations(t)
	f.gitClient.AssertExpectations(t)
}

// newReviewSubmitFixture wires a Controller with mocks common to all
// ReviewSubmit tests. The PR is Open, has a different author than the session
// principal, and the git.GetCommit call resolves to reviewSubmitResolvedSHA.
func newReviewSubmitFixture(t *testing.T, pr *types.PullReq) *reviewSubmitFixture {
	t.Helper()

	repo := &types.RepositoryCore{
		ID:       reviewSubmitRepoID,
		ParentID: reviewSubmitParentID,
		Path:     "space/repo",
		State:    enum.RepoStateActive,
	}

	tx := &txAssertStub{}
	pullreqStore := &mockstore.PullReqStore{}
	reviewStore := &mockstore.PullReqReviewStore{}
	reviewerStore := &mockstore.PullReqReviewerStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitClient := &mockgit.Interface{}

	pullreqStore.On("FindByNumber", reviewSubmitRepoID, reviewSubmitPRNum).
		Return(pr, nil).Once()

	// Resolve the input revision to a canonical SHA.
	gitClient.On("GetCommit", mock.Anything, mock.Anything).
		Return(&git.GetCommitOutput{
			Commit: git.Commit{SHA: sha.Must(reviewSubmitResolvedSHA)},
		}, nil).Maybe()

	ctrl := &Controller{
		tx:              tx,
		authorizer:      &allowAuthorizer{},
		repoFinder:      testRepoFinder(repo),
		pullreqStore:    pullreqStore,
		reviewStore:     reviewStore,
		reviewerStore:   reviewerStore,
		activityStore:   activityStore,
		git:             gitClient,
		eventReporter:   mockpullreq.NewStubReporter(t),
		instrumentation: noopInstrumentation{},
	}

	return &reviewSubmitFixture{
		ctrl:          ctrl,
		tx:            tx,
		pullreqStore:  pullreqStore,
		reviewStore:   reviewStore,
		reviewerStore: reviewerStore,
		activityStore: activityStore,
		gitClient:     gitClient,
	}
}

func reviewSubmitPR() *types.PullReq {
	return &types.PullReq{
		ID:        reviewSubmitPRID,
		Number:    reviewSubmitPRNum,
		State:     enum.PullReqStateOpen,
		CreatedBy: reviewSubmitAuthorID,
	}
}

func reviewSubmitInput(decision enum.PullReqReviewDecision) *ReviewSubmitInput {
	return &ReviewSubmitInput{
		CommitSHA: reviewSubmitResolvedSHA,
		Decision:  decision,
	}
}

// TestReviewSubmit_NewReviewer_CreatesReviewAndReviewer: no reviewer row
// exists yet; a review row is inserted and a reviewer row is created with
// type=SelfAssigned inside the same transaction.
func TestReviewSubmit_NewReviewer_CreatesReviewAndReviewer(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(nil, basestore.ErrResourceNotFound).Once()

	f.reviewStore.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		require.True(t, f.tx.inTx, "review create must run inside transaction")
		review, ok := args.Get(0).(*types.PullReqReview)
		require.True(t, ok, "reviewStore.Create arg must be *types.PullReqReview")
		review.ID = 501 // simulate DB-assigned id used by reviewer.LatestReviewID
		require.Equal(t, enum.PullReqReviewDecisionApproved, review.Decision)
		require.Equal(t, reviewSubmitResolvedSHA, review.SHA)
	}).Return(nil).Once()

	f.reviewerStore.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		require.True(t, f.tx.inTx, "reviewer create must run inside transaction")
		reviewer, ok := args.Get(0).(*types.PullReqReviewer)
		require.True(t, ok, "reviewerStore.Create arg must be *types.PullReqReviewer")
		require.Equal(t, enum.PullReqReviewerTypeSelfAssigned, reviewer.Type)
		require.NotNil(t, reviewer.LatestReviewID)
		require.Equal(t, int64(501), *reviewer.LatestReviewID)
		require.Equal(t, enum.PullReqReviewDecisionApproved, reviewer.ReviewDecision)
		require.Equal(t, reviewSubmitResolvedSHA, reviewer.SHA)
	}).Return(nil).Once()

	f.pullreqStore.On("UpdateActivitySeq", mock.Anything).
		Return(reviewSubmitPR(), nil).Once()
	f.activityStore.On("CreateWithPayload", mock.Anything, reviewSubmitReviewerID, mock.Anything, mock.Anything).
		Return(&types.PullReqActivity{}, nil).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.NoError(t, err)
	require.True(t, f.tx.called, "transaction must be executed")
	f.assertExpectations(t)
}

// TestReviewSubmit_IdempotentReplay_ShortCircuits: reviewer row already
// records the same decision on the same SHA — the endpoint returns success
// without creating any review row, updating the reviewer, or writing activity.
func TestReviewSubmit_IdempotentReplay_ShortCircuits(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	existingReviewID := int64(42)
	existing := &types.PullReqReviewer{
		PullReqID:      reviewSubmitPRID,
		PrincipalID:    reviewSubmitReviewerID,
		Type:           enum.PullReqReviewerTypeSelfAssigned,
		LatestReviewID: &existingReviewID,
		ReviewDecision: enum.PullReqReviewDecisionApproved,
		SHA:            reviewSubmitResolvedSHA,
	}
	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(existing, nil).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.NoError(t, err)
	require.False(t, f.tx.called, "idempotent replay must not open a transaction")
	// No Create/Update/UpdateActivitySeq/CreateWithPayload expectations were
	// set; assertExpectations catches any accidental call via strict mocks.
	f.assertExpectations(t)
}

// TestReviewSubmit_ExistingReviewer_DecisionChange_UpdatesReviewer: reviewer
// row exists with a different decision — a new review row is inserted and
// the reviewer row is updated (not re-created).
func TestReviewSubmit_ExistingReviewer_DecisionChange_UpdatesReviewer(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	oldReviewID := int64(42)
	existing := &types.PullReqReviewer{
		PullReqID:      reviewSubmitPRID,
		PrincipalID:    reviewSubmitReviewerID,
		Type:           enum.PullReqReviewerTypeAssigned, // pre-existing type must be preserved
		LatestReviewID: &oldReviewID,
		ReviewDecision: enum.PullReqReviewDecisionApproved,
		SHA:            reviewSubmitResolvedSHA,
	}
	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(existing, nil).Once()

	f.reviewStore.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		require.True(t, f.tx.inTx)
		review, ok := args.Get(0).(*types.PullReqReview)
		require.True(t, ok)
		review.ID = 502
	}).Return(nil).Once()

	f.reviewerStore.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		require.True(t, f.tx.inTx)
		reviewer, ok := args.Get(0).(*types.PullReqReviewer)
		require.True(t, ok)
		require.Equal(t, enum.PullReqReviewerTypeAssigned, reviewer.Type,
			"reviewer.Type must be preserved on decision change")
		require.NotNil(t, reviewer.LatestReviewID)
		require.Equal(t, int64(502), *reviewer.LatestReviewID)
		require.Equal(t, enum.PullReqReviewDecisionChangeReq, reviewer.ReviewDecision)
	}).Return(nil).Once()

	f.pullreqStore.On("UpdateActivitySeq", mock.Anything).
		Return(reviewSubmitPR(), nil).Once()
	f.activityStore.On("CreateWithPayload", mock.Anything, reviewSubmitReviewerID, mock.Anything, mock.Anything).
		Return(&types.PullReqActivity{}, nil).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionChangeReq),
	)
	require.NoError(t, err)
	require.True(t, f.tx.called)
	f.assertExpectations(t)
}

// TestReviewSubmit_ExistingReviewer_SHAChange_UpdatesReviewer: reviewer row
// exists with the same decision but on an older SHA — treated as a new
// review (not a no-op).
func TestReviewSubmit_ExistingReviewer_SHAChange_UpdatesReviewer(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	oldReviewID := int64(42)
	existing := &types.PullReqReviewer{
		PullReqID:      reviewSubmitPRID,
		PrincipalID:    reviewSubmitReviewerID,
		Type:           enum.PullReqReviewerTypeSelfAssigned,
		LatestReviewID: &oldReviewID,
		ReviewDecision: enum.PullReqReviewDecisionApproved,
		SHA:            "old-sha",
	}
	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(existing, nil).Once()

	f.reviewStore.On("Create", mock.Anything).Run(func(args mock.Arguments) {
		review, ok := args.Get(0).(*types.PullReqReview)
		require.True(t, ok)
		review.ID = 503
	}).Return(nil).Once()

	f.reviewerStore.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		reviewer, ok := args.Get(0).(*types.PullReqReviewer)
		require.True(t, ok)
		require.Equal(t, reviewSubmitResolvedSHA, reviewer.SHA)
	}).Return(nil).Once()

	f.pullreqStore.On("UpdateActivitySeq", mock.Anything).
		Return(reviewSubmitPR(), nil).Once()
	f.activityStore.On("CreateWithPayload", mock.Anything, reviewSubmitReviewerID, mock.Anything, mock.Anything).
		Return(&types.PullReqActivity{}, nil).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.NoError(t, err)
	f.assertExpectations(t)
}

// TestReviewSubmit_ActivityWriteFailure_IsSwallowed: the pullreq activity
// write is best-effort — failure at UpdateActivitySeq or CreateWithPayload
// must not surface to the caller, and the follow-up eventReporter /
// reportReviewerAddition calls must still see a valid PR (regression guard
// against the nil-pr shadowing bug that previously panicked here).
func TestReviewSubmit_ActivityWriteFailure_IsSwallowed(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(nil, basestore.ErrResourceNotFound).Once()
	f.reviewStore.On("Create", mock.Anything).
		Run(func(args mock.Arguments) {
			review, ok := args.Get(0).(*types.PullReqReview)
			require.True(t, ok)
			review.ID = 504
		}).
		Return(nil).Once()
	f.reviewerStore.On("Create", mock.Anything).Return(nil).Once()
	// Return (nil, err) — the store's real failure shape. If the controller
	// re-uses the outer pr after this call, we'd panic on eventBase(pr, ...).
	f.pullreqStore.On("UpdateActivitySeq", mock.Anything).
		Return(nil, errors.New("db gone")).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.NoError(t, err, "activity write failure must not fail the request")
	f.assertExpectations(t)
}

// TestReviewSubmit_InvalidDecision_ReturnsBadRequest: pending decision is
// rejected upfront — no DB or git access happens.
func TestReviewSubmit_InvalidDecision_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	err := ctrl.ReviewSubmit(context.Background(), testSession(), "1", reviewSubmitPRNum,
		&ReviewSubmitInput{CommitSHA: reviewSubmitResolvedSHA, Decision: enum.PullReqReviewDecisionPending})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Decision must be")
}

// TestReviewSubmit_MissingCommitSHA_ReturnsBadRequest: empty CommitSHA is
// rejected upfront.
func TestReviewSubmit_MissingCommitSHA_ReturnsBadRequest(t *testing.T) {
	t.Parallel()

	ctrl := &Controller{}
	err := ctrl.ReviewSubmit(context.Background(), testSession(), "1", reviewSubmitPRNum,
		&ReviewSubmitInput{CommitSHA: "", Decision: enum.PullReqReviewDecisionApproved})
	require.Error(t, err)
	require.Contains(t, err.Error(), "CommitSHA")
}

// TestReviewSubmit_OwnPullRequest_Rejected: the PR author cannot review their
// own PR.
func TestReviewSubmit_OwnPullRequest_Rejected(t *testing.T) {
	t.Parallel()

	pr := reviewSubmitPR()
	pr.CreatedBy = reviewSubmitReviewerID // testSession principal owns the PR

	f := newReviewSubmitFixture(t, pr)

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "own pull requests")
	require.False(t, f.tx.called)
}

// TestReviewSubmit_MergedPullRequest_Rejected: reviews cannot be submitted on
// merged PRs.
func TestReviewSubmit_MergedPullRequest_Rejected(t *testing.T) {
	t.Parallel()

	mergedAt := int64(1_700_000_000_000)
	pr := reviewSubmitPR()
	pr.State = enum.PullReqStateMerged
	pr.Merged = &mergedAt

	f := newReviewSubmitFixture(t, pr)

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "merged pull requests")
	require.False(t, f.tx.called)
}

// TestReviewSubmit_ReviewerFindError_Fails: a non-NotFound reviewerStore.Find
// error propagates as a wrapped error.
func TestReviewSubmit_ReviewerFindError_Fails(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(nil, errors.New("db exploded")).Once()

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to fetch reviewer")
	require.False(t, f.tx.called, "must not open a transaction on Find error")
}

// TestReviewSubmit_ReviewStoreCreateError_RollsBack: reviewStore.Create
// failure inside the transaction propagates and no reviewer row is written.
func TestReviewSubmit_ReviewStoreCreateError_RollsBack(t *testing.T) {
	t.Parallel()

	f := newReviewSubmitFixture(t, reviewSubmitPR())

	f.reviewerStore.On("Find", reviewSubmitPRID, reviewSubmitReviewerID).
		Return(nil, basestore.ErrResourceNotFound).Once()
	f.reviewStore.On("Create", mock.Anything).
		Return(errors.New("constraint violation")).Once()
	// Deliberately no reviewerStore.Create expectation — asserting it isn't
	// called (strict mocks fail on unexpected calls).

	err := f.ctrl.ReviewSubmit(
		context.Background(), testSession(), "1", reviewSubmitPRNum,
		reviewSubmitInput(enum.PullReqReviewDecisionApproved),
	)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to create review")
	require.True(t, f.tx.called)
	f.assertExpectations(t)
}
