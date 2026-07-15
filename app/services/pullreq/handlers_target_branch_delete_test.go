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
	"os"
	"strings"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	gitapi "github.com/harness/gitness/git/api"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	mocksse "github.com/harness/gitness/mocks/sse"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/mock"

	_ "unsafe"
)

// bootstrapSystemPrincipal links to the unexported package-level var in bootstrap
// so tests can initialize it without a full DB setup.
//
//go:linkname bootstrapSystemPrincipal github.com/harness/gitness/app/bootstrap.systemServicePrincipal
var bootstrapSystemPrincipal *types.Principal

// TestMain initializes bootstrap globals required by createRPCSystemReferencesWriteParams
// and CloseBecauseNonUniqueMergeBase.
func TestMain(m *testing.M) {
	bootstrapSystemPrincipal = &types.Principal{
		ID:          -1,
		UID:         "system",
		Email:       "system@harness.io",
		DisplayName: "Harness System",
		Type:        enum.PrincipalTypeServiceAccount,
	}
	os.Exit(m.Run())
}

// sha values used across tests.
var (
	sourceSHAStr    = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	defaultSHAStr   = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	mergeBaseSHAStr = "cccccccccccccccccccccccccccccccccccccccc"

	defaultSHA, _   = sha.New(defaultSHAStr)
	mergeBaseSHA, _ = sha.New(mergeBaseSHAStr)
)

// stubURLProvider is a minimal url.Provider stub for tests.
// It embeds the interface so only GetInternalAPIURL needs to be implemented.
type stubURLProvider struct{ url.Provider }

func (s *stubURLProvider) GetInternalAPIURL(_ context.Context) string { return "http://localhost" }

// makeEvent constructs a BranchDeletedPayload event for the given branch.
func makeEvent(branch string) *events.Event[*gitevents.BranchDeletedPayload] {
	return &events.Event[*gitevents.BranchDeletedPayload]{
		Payload: &gitevents.BranchDeletedPayload{
			RepoID:      1,
			Ref:         "refs/heads/" + branch,
			SHA:         sourceSHAStr,
			PrincipalID: 1,
		},
	}
}

// makeRepo builds a minimal *types.Repository.
func makeRepo(defaultBranch string) *types.Repository {
	return &types.Repository{
		ID:            1,
		ParentID:      10,
		GitUID:        "git-uid-1",
		DefaultBranch: defaultBranch,
	}
}

// makePR builds a minimal open *types.PullReq targeting the given branch.
func makePR(number int64, targetBranch, sourceSHAValue, mergeBaseSHAValue string) *types.PullReq {
	srcRepoID := int64(1)
	return &types.PullReq{
		ID:           1,
		Number:       number,
		State:        enum.PullReqStateOpen,
		TargetRepoID: 1,
		SourceRepoID: &srcRepoID,
		TargetBranch: targetBranch,
		SourceBranch: "feature",
		SourceSHA:    sourceSHAValue,
		MergeBaseSHA: mergeBaseSHAValue,
	}
}

// newTestService creates a Service with only the fields needed for
// updatePullReqTargetOnBranchDelete and its callees.
func newTestService(
	pullreqStore *mockstore.PullReqStore,
	repoStore *mockstore.RepoStore,
	activityStore *mockstore.PullReqActivityStore,
	gitMock *mockgit.Interface,
	sseMock *mocksse.Streamer,
) *Service {
	return &Service{
		pullreqStore:      pullreqStore,
		repoStore:         repoStore,
		activityStore:     activityStore,
		git:               gitMock,
		sseStreamer:       sseMock,
		pullreqEvReporter: mockpullreq.NewStubReporter(&testing.T{}),
		urlProvider:       &stubURLProvider{},
	}
}

// TestUpdatePullReqTargetOnBranchDelete_NoPRs verifies that when no open PRs
// target the deleted branch, nothing is done.
func TestUpdatePullReqTargetOnBranchDelete_NoPRs(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}

	pullreqStore.On("List", mock.AnythingOfType("*types.PullReqFilter")).
		Return([]*types.PullReq{}, nil).Once()

	svc := newTestService(pullreqStore, repoStore, nil, nil, nil)

	err := svc.updatePullReqTargetOnBranchDelete(context.Background(), makeEvent("feature"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	// repoStore.Find should NOT be called when there are no PRs
	repoStore.AssertNotCalled(t, "Find", mock.Anything)
}

// TestUpdatePullReqTargetOnBranchDelete_ListError verifies that a DB error
// on List is propagated.
func TestUpdatePullReqTargetOnBranchDelete_ListError(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	pullreqStore.On("List", mock.AnythingOfType("*types.PullReqFilter")).
		Return(([]*types.PullReq)(nil), errors.New("db error")).Once()

	svc := newTestService(pullreqStore, &mockstore.RepoStore{}, nil, nil, nil)

	err := svc.updatePullReqTargetOnBranchDelete(context.Background(), makeEvent("feature"))
	if err == nil {
		t.Fatal("expected error")
	}
	pullreqStore.AssertExpectations(t)
}

// TestUpdatePullReqTargetOnBranchDelete_NoDefaultBranch verifies that when
// the repo has no default branch set, no update is attempted.
func TestUpdatePullReqTargetOnBranchDelete_NoDefaultBranch(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}

	pr := makePR(1, "feature", sourceSHAStr, mergeBaseSHAStr)
	pullreqStore.On("List", mock.AnythingOfType("*types.PullReqFilter")).
		Return([]*types.PullReq{pr}, nil).Once()

	repo := makeRepo("")
	repoStore.On("Find", int64(1)).Return(repo, nil).Once()

	svc := newTestService(pullreqStore, repoStore, nil, nil, nil)

	err := svc.updatePullReqTargetOnBranchDelete(context.Background(), makeEvent("feature"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
}

// TestUpdatePullReqTargetOnBranchDelete_DefaultBranchDeleted verifies that
// when the deleted branch IS the default branch, no update is attempted.
func TestUpdatePullReqTargetOnBranchDelete_DefaultBranchDeleted(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}

	pr := makePR(1, "main", sourceSHAStr, mergeBaseSHAStr)
	pullreqStore.On("List", mock.AnythingOfType("*types.PullReqFilter")).
		Return([]*types.PullReq{pr}, nil).Once()

	repo := makeRepo("main")
	repoStore.On("Find", int64(1)).Return(repo, nil).Once()

	svc := newTestService(pullreqStore, repoStore, nil, nil, nil)

	err := svc.updatePullReqTargetOnBranchDelete(context.Background(), makeEvent("main"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
}

// TestUpdatePullReqTargetOnBranchDelete_InvalidRef verifies that a bad ref
// results in a DiscardEventError (not a retry-able error).
func TestUpdatePullReqTargetOnBranchDelete_InvalidRef(t *testing.T) {
	t.Parallel()

	svc := newTestService(nil, nil, nil, nil, nil)

	ev := &events.Event[*gitevents.BranchDeletedPayload]{
		Payload: &gitevents.BranchDeletedPayload{
			RepoID: 1,
			Ref:    "not-a-heads-ref",
		},
	}

	err := svc.updatePullReqTargetOnBranchDelete(context.Background(), ev)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.HasPrefix(err.Error(), "discarding requested") {
		t.Fatalf("expected DiscardEventError, got: %v", err)
	}
}

// TestUpdatePRToDefaultBranch_Happy verifies the full happy path:
// PR is retargeted, activity written, git refs updated, events published.
func TestUpdatePRToDefaultBranch_Happy(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitMock := &mockgit.Interface{}
	sseMock := &mocksse.Streamer{}

	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	// GetRef for default branch
	gitMock.On("GetRef", mock.Anything, mock.MatchedBy(func(p git.GetRefParams) bool {
		return p.Name == "main" && p.Type == gitenum.RefTypeBranch
	})).Return(git.GetRefResponse{SHA: defaultSHA}, nil).Once()

	// MergeBase returns a different SHA (PR has new commits)
	gitMock.On("MergeBase", mock.Anything, mock.AnythingOfType("git.MergeBaseParams")).
		Return(git.MergeBaseOutput{MergeBaseSHA: mergeBaseSHA}, nil).Once()

	// DiffStats
	gitMock.On("DiffStats", mock.Anything, mock.AnythingOfType("*git.DiffParams")).
		Return(git.DiffStatsOutput{Commits: 2, FilesChanged: 3}, nil).Once()

	// UpdateOptLock - simulate successful update
	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return(pr, nil).Once()

	// Activity write (non-critical, success)
	activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(1),
		mock.AnythingOfType("*types.PullRequestActivityPayloadTargetBranchDeleted"),
		(*types.PullReqActivityMetadata)(nil),
	).Return((*types.PullReqActivity)(nil), nil).Once()

	// UpdateRef to delete stale merge ref
	gitMock.On("UpdateRef", mock.Anything, mock.AnythingOfType("git.UpdateRefParams")).
		Return(nil).Once()

	// SSE publish
	sseMock.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newTestService(pullreqStore, repoStore, activityStore, gitMock, sseMock)

	err := svc.updatePRToDefaultBranch(context.Background(), pr, repo, "feature", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	activityStore.AssertExpectations(t)
	gitMock.AssertExpectations(t)
	sseMock.AssertExpectations(t)
}

// TestUpdatePRToDefaultBranch_PRAlreadyClosed verifies that if UpdateOptLock
// returns ErrPullReqNotOpen (PR was closed concurrently), the function
// returns nil (not an error).
func TestUpdatePRToDefaultBranch_PRAlreadyClosed(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitMock := &mockgit.Interface{}
	sseMock := &mocksse.Streamer{}

	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	gitMock.On("GetRef", mock.Anything, mock.AnythingOfType("git.GetRefParams")).
		Return(git.GetRefResponse{SHA: defaultSHA}, nil).Once()

	gitMock.On("MergeBase", mock.Anything, mock.AnythingOfType("git.MergeBaseParams")).
		Return(git.MergeBaseOutput{MergeBaseSHA: mergeBaseSHA}, nil).Once()

	gitMock.On("DiffStats", mock.Anything, mock.AnythingOfType("*git.DiffParams")).
		Return(git.DiffStatsOutput{}, nil).Once()

	// Simulate concurrent close: UpdateOptLock returns ErrPullReqNotOpen
	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return((*types.PullReq)(nil), ErrPullReqNotOpen).Once()

	svc := newTestService(pullreqStore, repoStore, activityStore, gitMock, sseMock)

	err := svc.updatePRToDefaultBranch(context.Background(), pr, repo, "feature", 1)
	if err != nil {
		t.Fatalf("expected nil when PR already closed, got: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	activityStore.AssertNotCalled(t, "CreateWithPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	sseMock.AssertNotCalled(t, "Publish", mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdatePRToDefaultBranch_GetRefError verifies that a git error resolving
// the default branch is propagated.
func TestUpdatePRToDefaultBranch_GetRefError(t *testing.T) {
	t.Parallel()

	gitMock := &mockgit.Interface{}
	gitMock.On("GetRef", mock.Anything, mock.AnythingOfType("git.GetRefParams")).
		Return(git.GetRefResponse{}, errors.New("ref not found")).Once()

	svc := newTestService(nil, nil, nil, gitMock, nil)
	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	err := svc.updatePRToDefaultBranch(context.Background(), pr, repo, "feature", 1)
	if err == nil {
		t.Fatal("expected error")
	}
	gitMock.AssertExpectations(t)
}

// TestUpdatePRToDefaultBranch_UnrelatedHistories verifies that when
// MergeBase returns an unrelated-histories error, the PR is closed via
// CloseBecauseNonUniqueMergeBase (which calls repoStore.Find internally).
func TestUpdatePRToDefaultBranch_UnrelatedHistories(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitMock := &mockgit.Interface{}
	sseMock := &mocksse.Streamer{}

	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	gitMock.On("GetRef", mock.Anything, mock.AnythingOfType("git.GetRefParams")).
		Return(git.GetRefResponse{SHA: defaultSHA}, nil).Once()

	// Return unrelated histories error
	gitMock.On("MergeBase", mock.Anything, mock.AnythingOfType("git.MergeBaseParams")).
		Return(git.MergeBaseOutput{}, &gitapi.UnrelatedHistoriesError{}).Once()

	// writeTargetBranchDeletedActivity reserves an activity sequence number
	pullreqStore.On("UpdateActivitySeq", pr).Return(pr, nil).Once()

	// CloseBecauseNonUniqueMergeBase will call repoStore.Find and pullreqStore.UpdateOptLock
	repoStore.On("Find", int64(1)).Return(repo, nil).Once()

	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return(pr, nil).Once()

	repoStore.On("UpdateOptLock", repo, mock.AnythingOfType("func(*types.Repository) error")).
		Return(repo, nil).Once()

	// 1 from writeTargetBranchDeletedActivity + 2 from CloseBecauseNonUniqueMergeBase
	activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		mock.AnythingOfType("int64"),
		mock.Anything,
		mock.Anything,
	).Return((*types.PullReqActivity)(nil), nil).Times(3)

	sseMock.On("Publish", mock.Anything, mock.Anything, mock.Anything).Once()

	svc := newTestService(pullreqStore, repoStore, activityStore, gitMock, sseMock)

	err := svc.updatePRToDefaultBranch(context.Background(), pr, repo, "feature", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
	gitMock.AssertExpectations(t)
}

// TestUpdatePRToDefaultBranch_NoNewCommits verifies that when the merge base
// equals the source SHA (PR has no new commits), the PR is closed instead.
func TestUpdatePRToDefaultBranch_NoNewCommits(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitMock := &mockgit.Interface{}
	sseMock := &mocksse.Streamer{}

	repo := makeRepo("main")
	// Source SHA equals what MergeBase will return → no new commits
	pr := makePR(42, "feature", mergeBaseSHAStr, mergeBaseSHAStr)

	gitMock.On("GetRef", mock.Anything, mock.AnythingOfType("git.GetRefParams")).
		Return(git.GetRefResponse{SHA: defaultSHA}, nil).Once()

	// MergeBase returns the same SHA as SourceSHA
	gitMock.On("MergeBase", mock.Anything, mock.AnythingOfType("git.MergeBaseParams")).
		Return(git.MergeBaseOutput{MergeBaseSHA: mergeBaseSHA}, nil).Once()

	// closePRWithNoChanges → UpdateOptLock to close PR
	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return(pr, nil).Once()

	// repo counter update
	repoStore.On("UpdateOptLock", repo, mock.AnythingOfType("func(*types.Repository) error")).
		Return(repo, nil).Once()

	// UpdateRef to delete merge ref
	gitMock.On("UpdateRef", mock.Anything, mock.AnythingOfType("git.UpdateRefParams")).
		Return(nil).Once()

	// Two activity writes: target-branch-deleted + state-change
	activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(1),
		mock.AnythingOfType("*types.PullRequestActivityPayloadTargetBranchDeleted"),
		(*types.PullReqActivityMetadata)(nil),
	).Return((*types.PullReqActivity)(nil), nil).Once()

	activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(1),
		mock.AnythingOfType("*types.PullRequestActivityPayloadStateChange"),
		(*types.PullReqActivityMetadata)(nil),
	).Return((*types.PullReqActivity)(nil), nil).Once()

	sseMock.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newTestService(pullreqStore, repoStore, activityStore, gitMock, sseMock)

	err := svc.updatePRToDefaultBranch(context.Background(), pr, repo, "feature", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	repoStore.AssertExpectations(t)
	activityStore.AssertExpectations(t)
	gitMock.AssertExpectations(t)
	sseMock.AssertExpectations(t)
}

// TestClosePRWithNoChanges_PRAlreadyClosed verifies that if UpdateOptLock
// signals ErrPullReqNotOpen, closePRWithNoChanges returns nil.
func TestClosePRWithNoChanges_PRAlreadyClosed(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return((*types.PullReq)(nil), ErrPullReqNotOpen).Once()

	svc := newTestService(
		pullreqStore, &mockstore.RepoStore{}, &mockstore.PullReqActivityStore{}, &mockgit.Interface{}, &mocksse.Streamer{},
	)

	err := svc.closePRWithNoChanges(context.Background(), pr, repo, "feature", 1, mergeBaseSHAStr)
	if err != nil {
		t.Fatalf("expected nil when PR already closed, got: %v", err)
	}
	pullreqStore.AssertExpectations(t)
}

// TestClosePRWithNoChanges_UpdateOptLockError verifies that an unexpected
// UpdateOptLock error is propagated.
func TestClosePRWithNoChanges_UpdateOptLockError(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return((*types.PullReq)(nil), errors.New("db error")).Once()

	svc := newTestService(pullreqStore, &mockstore.RepoStore{}, nil, nil, nil)

	err := svc.closePRWithNoChanges(context.Background(), pr, repo, "feature", 1, mergeBaseSHAStr)
	if err == nil {
		t.Fatal("expected error")
	}
	pullreqStore.AssertExpectations(t)
}

// TestClosePRWithNoChanges_ActivityWriteErrorIgnored verifies that an error
// writing activity is logged but does not cause the function to fail.
func TestClosePRWithNoChanges_ActivityWriteErrorIgnored(t *testing.T) {
	t.Parallel()

	pullreqStore := &mockstore.PullReqStore{}
	repoStore := &mockstore.RepoStore{}
	activityStore := &mockstore.PullReqActivityStore{}
	gitMock := &mockgit.Interface{}
	sseMock := &mocksse.Streamer{}

	repo := makeRepo("main")
	pr := makePR(42, "feature", sourceSHAStr, mergeBaseSHAStr)

	pullreqStore.On("UpdateOptLock", pr, mock.AnythingOfType("func(*types.PullReq) error")).
		Return(pr, nil).Once()

	repoStore.On("UpdateOptLock", repo, mock.AnythingOfType("func(*types.Repository) error")).
		Return(repo, nil).Once()

	// merge ref deletion
	gitMock.On("UpdateRef", mock.Anything, mock.AnythingOfType("git.UpdateRefParams")).
		Return(nil).Once()

	// Both activity writes fail — should be ignored
	activityStore.On("CreateWithPayload",
		mock.AnythingOfType("*types.PullReq"),
		int64(1),
		mock.Anything,
		mock.Anything,
	).Return((*types.PullReqActivity)(nil), errors.New("activity error")).Times(2)

	sseMock.On("Publish", int64(10), enum.SSETypePullReqUpdated, mock.Anything).Once()

	svc := newTestService(pullreqStore, repoStore, activityStore, gitMock, sseMock)

	err := svc.closePRWithNoChanges(context.Background(), pr, repo, "feature", 1, mergeBaseSHAStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pullreqStore.AssertExpectations(t)
	activityStore.AssertExpectations(t)
	sseMock.AssertExpectations(t)
}
