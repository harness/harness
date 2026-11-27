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

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/services/refcache"
	storecache "github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	mockgit "github.com/harness/gitness/mocks/git"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mocksse "github.com/harness/gitness/mocks/sse"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"
)

const (
	branchUpdateOldSHA = "1111111111111111111111111111111111111111"
	branchUpdateNewSHA = "2222222222222222222222222222222222222222"
)

// stubRepoIDCache is a minimal store.RepoIDCache that always resolves to the same repo.
// refcache.RepoFinder is a concrete struct whose FindByID only delegates to this cache,
// so stubbing the cache is enough to drive the handler without a database.
type stubRepoIDCache struct {
	repo *types.RepositoryCore
}

func (c *stubRepoIDCache) Stats() (int64, int64) { return 0, 0 }

func (c *stubRepoIDCache) Get(_ context.Context, _ int64) (*types.RepositoryCore, error) {
	return c.repo, nil
}

func (c *stubRepoIDCache) Evict(_ context.Context, _ int64) {}

// branchUpdateMocks bundles the mocks a branch update test needs.
type branchUpdateMocks struct {
	pullreqStore   *mockstore.PullReqStore
	activityStore  *mockstore.PullReqActivityStore
	autoMergeStore *mockstore.AutoMergeStore
	git            *mockgit.Interface
	sse            *mocksse.Streamer
}

func (m *branchUpdateMocks) assertExpectations(t *testing.T) {
	t.Helper()
	m.pullreqStore.AssertExpectations(t)
	m.activityStore.AssertExpectations(t)
	m.autoMergeStore.AssertExpectations(t)
	m.git.AssertExpectations(t)
	m.sse.AssertExpectations(t)
}

// newBranchUpdateService builds a Service wired with the mocks needed by
// updatePullReqOnBranchUpdate and its callees.
func newBranchUpdateService(t *testing.T, m *branchUpdateMocks) *Service {
	t.Helper()

	repoCore := &types.RepositoryCore{
		ID:            1,
		ParentID:      10,
		GitUID:        "git-uid-1",
		DefaultBranch: "main",
	}

	return &Service{
		pullreqStore:   m.pullreqStore,
		activityStore:  m.activityStore,
		autoMergeStore: m.autoMergeStore,
		git:            m.git,
		sseStreamer:    m.sse,
		repoFinder: refcache.NewRepoFinder(
			nil, nil, &stubRepoIDCache{repo: repoCore}, nil,
			storecache.Evictor[*types.RepositoryCore]{},
		),
		pullreqEvReporter: mockpullreq.NewStubReporter(t),
		urlProvider:       &stubURLProvider{},
	}
}

// makeBranchUpdatedEvent builds a git branch updated event for the source branch.
func makeBranchUpdatedEvent() *events.Event[*gitevents.BranchUpdatedPayload] {
	return &events.Event[*gitevents.BranchUpdatedPayload]{
		Payload: &gitevents.BranchUpdatedPayload{
			RepoID:      1,
			Ref:         "refs/heads/feature",
			OldSHA:      branchUpdateOldSHA,
			NewSHA:      branchUpdateNewSHA,
			PrincipalID: 7,
		},
	}
}

// makeBranchUpdatePR builds an open PR on the source branch in the given sub-state.
func makeBranchUpdatePR(subState enum.PullReqSubState) *types.PullReq {
	srcRepoID := int64(1)
	return &types.PullReq{
		ID:           99,
		Number:       42,
		State:        enum.PullReqStateOpen,
		SubState:     subState,
		TargetRepoID: 1,
		SourceRepoID: &srcRepoID,
		TargetBranch: "main",
		SourceBranch: "feature",
		SourceSHA:    branchUpdateOldSHA,
		MergeBaseSHA: mergeBaseSHAStr,
		ActivitySeq:  5,
	}
}

// expectGitAndListCalls sets up every mock call the handler makes before and
// after the UpdateOptLock, except the store writes that each test asserts on.
func expectGitAndListCalls(m *branchUpdateMocks, pr *types.PullReq) {
	m.pullreqStore.On("ResetMergeCheckStatus", int64(1), "feature").Return(nil).Once()
	m.pullreqStore.On("List", mock.AnythingOfType("*types.PullReqFilter")).
		Return([]*types.PullReq{pr}, nil).Once()

	m.git.On("GetCommit", mock.Anything, mock.AnythingOfType("*git.GetCommitParams")).
		Return(&git.GetCommitOutput{Commit: git.Commit{Title: "new commit"}}, nil).Once()

	m.git.On("UpdateRef", mock.Anything, mock.AnythingOfType("git.UpdateRefParams")).
		Return(nil).Once()

	m.git.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
		return p.Name == "main" && p.Type == gitenum.RefTypeBranch
	})).Return(git.GetRefResponse{SHA: defaultSHA}, nil).Once()

	m.git.On("MergeBase", mock.Anything, mock.AnythingOfType("git.MergeBaseParams")).
		Return(git.MergeBaseOutput{MergeBaseSHA: mergeBaseSHA}, nil).Once()
}

// expectUpdateOptLock runs the handler's mutateFn against the PR so the test can
// observe the state the handler intended to persist.
func expectUpdateOptLock(m *branchUpdateMocks, pr *types.PullReq, mutateErr error) {
	call := m.pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Run(func(args mock.Arguments) {
			mutateFn, ok := args.Get(1).(func(pr *types.PullReq) error)
			if !ok {
				return
			}
			_ = mutateFn(pr)
		}).Once()

	if mutateErr != nil {
		call.Return((*types.PullReq)(nil), mutateErr)
		return
	}
	call.Return(pr, nil)
}

// TestUpdatePullReqOnBranchUpdate_AutoMergeCleared is the core assertion of CODE-5601:
// a push to the source branch clears the auto-merge sub-state, removes the stored
// auto-merge request, records it on the timeline and notifies the UI.
func TestUpdatePullReqOnBranchUpdate_AutoMergeCleared(t *testing.T) {
	t.Parallel()

	m := &branchUpdateMocks{
		pullreqStore:   &mockstore.PullReqStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		git:            &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	pr := makeBranchUpdatePR(enum.PullReqSubStateAutoMerge)
	expectGitAndListCalls(m, pr)
	expectUpdateOptLock(m, pr, nil)

	var branchUpdateSeq, autoMergeDisabledSeq int64

	m.activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(7),
		mock.AnythingOfType("*types.PullRequestActivityPayloadBranchUpdate"),
		(*types.PullReqActivityMetadata)(nil),
	).Run(func(args mock.Arguments) {
		activityPR, _ := args.Get(0).(*types.PullReq)
		branchUpdateSeq = activityPR.ActivitySeq
	}).Return((*types.PullReqActivity)(nil), nil).Once()

	m.autoMergeStore.On("Delete", int64(99)).Return(true, nil).Once()

	m.activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(7),
		mock.AnythingOfType("*types.PullRequestActivityPayloadAutoMergeDisabledBranchUpdate"),
		(*types.PullReqActivityMetadata)(nil),
	).Run(func(args mock.Arguments) {
		activityPR, _ := args.Get(0).(*types.PullReq)
		autoMergeDisabledSeq = activityPR.ActivitySeq

		payload, ok := args.Get(2).(*types.PullRequestActivityPayloadAutoMergeDisabledBranchUpdate)
		if !ok {
			t.Errorf("unexpected payload type %T", args.Get(2))
			return
		}
		if payload.Old != branchUpdateOldSHA {
			t.Errorf("payload.Old = %q, want %q", payload.Old, branchUpdateOldSHA)
		}
		if payload.New != branchUpdateNewSHA {
			t.Errorf("payload.New = %q, want %q", payload.New, branchUpdateNewSHA)
		}
	}).Return((*types.PullReqActivity)(nil), nil).Once()

	m.sse.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newBranchUpdateService(t, m)

	if err := svc.updatePullReqOnBranchUpdate(context.Background(), makeBranchUpdatedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.SubState != enum.PullReqSubStateNone {
		t.Errorf("sub-state = %q, want cleared", pr.SubState)
	}
	if pr.SourceSHA != branchUpdateNewSHA {
		t.Errorf("source SHA = %q, want %q", pr.SourceSHA, branchUpdateNewSHA)
	}
	if pr.SourceSHAPushedBy == nil || *pr.SourceSHAPushedBy != 7 {
		t.Errorf("source SHA pushed by = %v, want 7", pr.SourceSHAPushedBy)
	}
	// Two sequence numbers are reserved, so the counter advances from 5 to 7.
	if branchUpdateSeq != 6 {
		t.Errorf("branch update activity seq = %d, want 6", branchUpdateSeq)
	}
	if autoMergeDisabledSeq != 7 {
		t.Errorf("auto-merge-disabled activity seq = %d, want 7", autoMergeDisabledSeq)
	}
	if branchUpdateSeq >= autoMergeDisabledSeq {
		t.Errorf("branch update activity (%d) must precede auto-merge-disabled activity (%d)",
			branchUpdateSeq, autoMergeDisabledSeq)
	}

	m.assertExpectations(t)
}

// TestUpdatePullReqOnBranchUpdate_NoAutoMerge guards the common path: a PR without
// auto-merge must behave exactly as before, in particular it must not consume a
// second activity sequence number.
func TestUpdatePullReqOnBranchUpdate_NoAutoMerge(t *testing.T) {
	t.Parallel()

	m := &branchUpdateMocks{
		pullreqStore:   &mockstore.PullReqStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		git:            &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	pr := makeBranchUpdatePR(enum.PullReqSubStateNone)
	expectGitAndListCalls(m, pr)
	expectUpdateOptLock(m, pr, nil)

	var branchUpdateSeq int64

	m.activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(7),
		mock.AnythingOfType("*types.PullRequestActivityPayloadBranchUpdate"),
		(*types.PullReqActivityMetadata)(nil),
	).Run(func(args mock.Arguments) {
		activityPR, _ := args.Get(0).(*types.PullReq)
		branchUpdateSeq = activityPR.ActivitySeq
	}).Return((*types.PullReqActivity)(nil), nil).Once()

	m.sse.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newBranchUpdateService(t, m)

	if err := svc.updatePullReqOnBranchUpdate(context.Background(), makeBranchUpdatedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.SubState != enum.PullReqSubStateNone {
		t.Errorf("sub-state = %q, want unchanged", pr.SubState)
	}
	if branchUpdateSeq != 6 {
		t.Errorf("branch update activity seq = %d, want 6 (single increment)", branchUpdateSeq)
	}

	m.autoMergeStore.AssertNotCalled(t, "Delete", mock.Anything)
	m.assertExpectations(t)
}

// TestUpdatePullReqOnBranchUpdate_MergeQueueUntouched verifies the handler leaves the
// merge queue sub-state alone. The merge queue owns that state and re-merges against
// the new source SHA, and pushes to a queued PR are rejected at pre-receive anyway.
func TestUpdatePullReqOnBranchUpdate_MergeQueueUntouched(t *testing.T) {
	t.Parallel()

	m := &branchUpdateMocks{
		pullreqStore:   &mockstore.PullReqStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		git:            &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	pr := makeBranchUpdatePR(enum.PullReqSubStateMergeQueue)
	expectGitAndListCalls(m, pr)
	expectUpdateOptLock(m, pr, nil)

	m.activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(7),
		mock.AnythingOfType("*types.PullRequestActivityPayloadBranchUpdate"),
		(*types.PullReqActivityMetadata)(nil),
	).Return((*types.PullReqActivity)(nil), nil).Once()

	m.sse.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newBranchUpdateService(t, m)

	if err := svc.updatePullReqOnBranchUpdate(context.Background(), makeBranchUpdatedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.SubState != enum.PullReqSubStateMergeQueue {
		t.Errorf("sub-state = %q, want %q", pr.SubState, enum.PullReqSubStateMergeQueue)
	}

	m.autoMergeStore.AssertNotCalled(t, "Delete", mock.Anything)
	m.assertExpectations(t)
}

// TestUpdatePullReqOnBranchUpdate_AutoMergeDeleteErrorIgnored verifies that a failure
// to remove the stored auto-merge request does not resurrect the flag or fail the push.
func TestUpdatePullReqOnBranchUpdate_AutoMergeDeleteErrorIgnored(t *testing.T) {
	t.Parallel()

	m := &branchUpdateMocks{
		pullreqStore:   &mockstore.PullReqStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		git:            &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	pr := makeBranchUpdatePR(enum.PullReqSubStateAutoMerge)
	expectGitAndListCalls(m, pr)
	expectUpdateOptLock(m, pr, nil)

	m.autoMergeStore.On("Delete", int64(99)).Return(false, errors.New("db error")).Once()

	m.activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(7),
		mock.Anything,
		(*types.PullReqActivityMetadata)(nil),
	).Return((*types.PullReqActivity)(nil), nil).Times(2)

	m.sse.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newBranchUpdateService(t, m)

	if err := svc.updatePullReqOnBranchUpdate(context.Background(), makeBranchUpdatedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.SubState != enum.PullReqSubStateNone {
		t.Errorf("sub-state = %q, want cleared even when the auto merge delete fails", pr.SubState)
	}

	m.assertExpectations(t)
}

// TestUpdatePullReqOnBranchUpdate_PRClosedConcurrently verifies that a PR closed
// between the list and the update is skipped without any auto-merge cleanup.
func TestUpdatePullReqOnBranchUpdate_PRClosedConcurrently(t *testing.T) {
	t.Parallel()

	m := &branchUpdateMocks{
		pullreqStore:   &mockstore.PullReqStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		git:            &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	pr := makeBranchUpdatePR(enum.PullReqSubStateAutoMerge)
	pr.State = enum.PullReqStateClosed
	expectGitAndListCalls(m, pr)
	expectUpdateOptLock(m, pr, ErrPullReqNotOpen)

	svc := newBranchUpdateService(t, m)

	if err := svc.updatePullReqOnBranchUpdate(context.Background(), makeBranchUpdatedEvent()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pr.SubState != enum.PullReqSubStateAutoMerge {
		t.Errorf("sub-state = %q, want untouched on a closed PR", pr.SubState)
	}

	m.autoMergeStore.AssertNotCalled(t, "Delete", mock.Anything)
	m.activityStore.AssertNotCalled(t, "CreateWithPayload",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	m.sse.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
	m.assertExpectations(t)
}

// TestAutoMergeDisabledBranchUpdatePayloadIsRegistered guards the payload factory
// registration. Without it, reading the PR timeline fails for this activity type.
func TestAutoMergeDisabledBranchUpdatePayloadIsRegistered(t *testing.T) {
	t.Parallel()

	activityType := enum.PullReqActivityTypeAutoMergeDisabledBranchUpdate

	if _, ok := activityType.Sanitize(); !ok {
		t.Fatalf("activity type %q is not registered in the enum list", activityType)
	}

	activity := &types.PullReqActivity{
		Type:       enum.PullReqActivityTypeAutoMergeDisabledBranchUpdate,
		Kind:       enum.PullReqActivityKindSystem,
		PayloadRaw: []byte(`{"old":"` + branchUpdateOldSHA + `","new":"` + branchUpdateNewSHA + `"}`),
	}

	payload, err := activity.GetPayload()
	if err != nil {
		t.Fatalf("failed to resolve payload, is it missing from allPullReqActivityPayloads? %v", err)
	}

	typed, ok := payload.(*types.PullRequestActivityPayloadAutoMergeDisabledBranchUpdate)
	if !ok {
		t.Fatalf("unexpected payload type %T", payload)
	}
	if typed.Old != branchUpdateOldSHA || typed.New != branchUpdateNewSHA {
		t.Errorf("payload round-trip mismatch: got old=%q new=%q", typed.Old, typed.New)
	}
}
