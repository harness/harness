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
	"testing"
	"time"

	"github.com/harness/gitness/app/auth/authz/authztest"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	mockgit "github.com/harness/gitness/mocks/git"
	mocksse "github.com/harness/gitness/mocks/sse"
	mockstore "github.com/harness/gitness/mocks/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	stateTestRepoID   int64 = 1
	stateTestParentID int64 = 10
	stateTestPRID     int64 = 55
	stateTestPRNum    int64 = 7

	// stateTestSentinelPRID marks readiness probes sent through the event system.
	stateTestSentinelPRID int64 = -1
)

// stateEventRecorder captures the pull request events published by State.
type stateEventRecorder struct {
	ready    chan struct{}
	draft    chan *pullreqevents.DraftChangedPayload
	closed   chan *pullreqevents.ClosedPayload
	reopened chan *pullreqevents.ReopenedPayload
}

// newStateEventRecorder returns a reporter backed by an in-memory event system
// and a recorder subscribed to the draft/closed/reopened events.
//
// The in-memory broker drops messages for streams that have no subscribed
// group yet, and subscriptions happen asynchronously after Launch, so every
// stream is probed with a sentinel event until it is confirmed live.
func newStateEventRecorder(t *testing.T) (*pullreqevents.Reporter, *stateEventRecorder) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	system, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil, events.NewNoopCollector())
	require.NoError(t, err)

	readerFactory, err := pullreqevents.NewReaderFactory(system)
	require.NoError(t, err)

	rec := &stateEventRecorder{
		ready:    make(chan struct{}, 100),
		draft:    make(chan *pullreqevents.DraftChangedPayload, 10),
		closed:   make(chan *pullreqevents.ClosedPayload, 10),
		reopened: make(chan *pullreqevents.ReopenedPayload, 10),
	}

	_, err = readerFactory.Launch(ctx, "test-group", "test-reader", func(r *pullreqevents.Reader) error {
		if err := r.RegisterDraftChanged(
			func(_ context.Context, e *events.Event[*pullreqevents.DraftChangedPayload]) error {
				if e.Payload.PullReqID == stateTestSentinelPRID {
					rec.ready <- struct{}{}
					return nil
				}
				rec.draft <- e.Payload
				return nil
			}); err != nil {
			return err
		}
		if err := r.RegisterClosed(
			func(_ context.Context, e *events.Event[*pullreqevents.ClosedPayload]) error {
				if e.Payload.PullReqID == stateTestSentinelPRID {
					rec.ready <- struct{}{}
					return nil
				}
				rec.closed <- e.Payload
				return nil
			}); err != nil {
			return err
		}
		return r.RegisterReopened(
			func(_ context.Context, e *events.Event[*pullreqevents.ReopenedPayload]) error {
				if e.Payload.PullReqID == stateTestSentinelPRID {
					rec.ready <- struct{}{}
					return nil
				}
				rec.reopened <- e.Payload
				return nil
			})
	})
	require.NoError(t, err)

	reporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	sentinel := pullreqevents.Base{PullReqID: stateTestSentinelPRID}
	rec.waitReady(t, func() { reporter.DraftChanged(ctx, &pullreqevents.DraftChangedPayload{Base: sentinel}) })
	rec.waitReady(t, func() { reporter.Closed(ctx, &pullreqevents.ClosedPayload{Base: sentinel}) })
	rec.waitReady(t, func() { reporter.Reopened(ctx, &pullreqevents.ReopenedPayload{Base: sentinel}) })

	return reporter, rec
}

func (rec *stateEventRecorder) waitReady(t *testing.T, send func()) {
	t.Helper()

	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	timeout := time.After(5 * time.Second)

	send()
	for {
		select {
		case <-rec.ready:
			// drain probes that were already in flight.
			time.Sleep(50 * time.Millisecond)
			for len(rec.ready) > 0 {
				<-rec.ready
			}
			return
		case <-ticker.C:
			send()
		case <-timeout:
			t.Fatal("timed out waiting for event reader to subscribe")
		}
	}
}

func (rec *stateEventRecorder) expectDraft(t *testing.T) *pullreqevents.DraftChangedPayload {
	t.Helper()
	select {
	case p := <-rec.draft:
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for draft changed event")
		return nil
	}
}

func (rec *stateEventRecorder) expectClosed(t *testing.T) *pullreqevents.ClosedPayload {
	t.Helper()
	select {
	case p := <-rec.closed:
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for closed event")
		return nil
	}
}

func (rec *stateEventRecorder) expectReopened(t *testing.T) *pullreqevents.ReopenedPayload {
	t.Helper()
	select {
	case p := <-rec.reopened:
		return p
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for reopened event")
		return nil
	}
}

// expectNoMore asserts that no further events arrive within a short window.
func (rec *stateEventRecorder) expectNoMore(t *testing.T) {
	t.Helper()
	select {
	case p := <-rec.draft:
		t.Fatalf("unexpected draft changed event: %+v", p)
	case p := <-rec.closed:
		t.Fatalf("unexpected closed event: %+v", p)
	case p := <-rec.reopened:
		t.Fatalf("unexpected reopened event: %+v", p)
	case <-time.After(200 * time.Millisecond):
	}
}

type stateFixture struct {
	ctrl           *Controller
	tx             *txAssertStub
	rec            *stateEventRecorder
	pullreqStore   *mockstore.PullReqStore
	repoStore      *mockstore.RepoStore
	autoMergeStore *mockstore.AutoMergeStore
	activityStore  *mockstore.PullReqActivityStore
	gitClient      *mockgit.Interface
	sse            *mocksse.Streamer
}

func (f *stateFixture) assertExpectations(t *testing.T) {
	t.Helper()
	f.pullreqStore.AssertExpectations(t)
	f.repoStore.AssertExpectations(t)
	f.autoMergeStore.AssertExpectations(t)
	f.activityStore.AssertExpectations(t)
	f.gitClient.AssertExpectations(t)
	f.sse.AssertExpectations(t)
}

func stateTestPR(state enum.PullReqState, isDraft bool) *types.PullReq {
	repoID := stateTestRepoID
	return &types.PullReq{
		ID:           stateTestPRID,
		Number:       stateTestPRNum,
		State:        state,
		IsDraft:      isDraft,
		SourceRepoID: &repoID,
		SourceBranch: "feature",
		SourceSHA:    "1111111111111111111111111111111111111111",
		TargetRepoID: stateTestRepoID,
		TargetBranch: "main",
	}
}

func newStateFixture(t *testing.T, pr *types.PullReq) *stateFixture {
	t.Helper()

	repo := &types.RepositoryCore{
		ID:       stateTestRepoID,
		ParentID: stateTestParentID,
		Path:     "space/repo",
		GitUID:   "git-uid",
		State:    enum.RepoStateActive,
	}

	reporter, rec := newStateEventRecorder(t)

	f := &stateFixture{
		tx:             &txAssertStub{},
		rec:            rec,
		pullreqStore:   &mockstore.PullReqStore{},
		repoStore:      &mockstore.RepoStore{},
		autoMergeStore: &mockstore.AutoMergeStore{},
		activityStore:  &mockstore.PullReqActivityStore{},
		gitClient:      &mockgit.Interface{},
		sse:            &mocksse.Streamer{},
	}

	f.pullreqStore.On("FindByNumber", stateTestRepoID, stateTestPRNum).Return(pr, nil).Once()
	f.repoStore.On("Find", stateTestRepoID).
		Return(&types.Repository{ID: stateTestRepoID, ParentID: stateTestParentID, Path: "space/repo"}, nil).Once()
	f.activityStore.On("CreateWithPayload", mock.Anything, testSession().Principal.ID, mock.Anything, mock.Anything).
		Return(&types.PullReqActivity{}, nil).Once()
	f.sse.On("Publish", stateTestParentID, enum.SSETypePullReqUpdated, mock.Anything).Once()

	f.ctrl = &Controller{
		tx:             f.tx,
		authorizer:     authztest.AllowAuthorizer{},
		repoFinder:     testRepoFinder(repo),
		urlProvider:    &revertURLProvider{},
		pullreqStore:   f.pullreqStore,
		repoStore:      f.repoStore,
		autoMergeStore: f.autoMergeStore,
		activityStore:  f.activityStore,
		git:            f.gitClient,
		eventReporter:  reporter,
		sseStreamer:    f.sse,
		// mergeQueueService is left nil: IsEnqueued only inspects the pull request.
	}

	return f
}

func TestState_MarkAsDraft_EmitsDraftChanged(t *testing.T) {
	f := newStateFixture(t, stateTestPR(enum.PullReqStateOpen, false))

	// marking an open PR as draft must cancel auto merge inside the transaction.
	f.autoMergeStore.On("Delete", stateTestPRID).Run(func(mock.Arguments) {
		require.True(t, f.tx.inTx, "auto merge delete must run inside transaction")
	}).Return(true, nil).Once()
	f.pullreqStore.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		require.True(t, f.tx.inTx, "pull request update must run inside transaction")
		pr, _ := args.Get(0).(*types.PullReq)
		require.True(t, pr.IsDraft)
		require.Equal(t, enum.PullReqStateOpen, pr.State)
	}).Return(nil).Once()

	out, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
		&StateInput{State: enum.PullReqStateOpen, IsDraft: true})
	require.NoError(t, err)
	require.True(t, out.IsDraft)

	p := f.rec.expectDraft(t)
	require.True(t, p.IsDraft)
	require.Equal(t, stateTestPRID, p.PullReqID)
	require.Equal(t, stateTestPRNum, p.Number)
	require.Equal(t, stateTestRepoID, p.TargetRepoID)
	require.Equal(t, testSession().Principal.ID, p.PrincipalID)

	f.rec.expectNoMore(t)
	f.assertExpectations(t)
}

func TestState_MarkReadyForReview_EmitsDraftChanged(t *testing.T) {
	f := newStateFixture(t, stateTestPR(enum.PullReqStateOpen, true))

	// leaving draft must not touch auto merge.
	f.pullreqStore.On("Update", mock.Anything).Return(nil).Once()

	out, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
		&StateInput{State: enum.PullReqStateOpen, IsDraft: false})
	require.NoError(t, err)
	require.False(t, out.IsDraft)

	p := f.rec.expectDraft(t)
	require.False(t, p.IsDraft)

	f.rec.expectNoMore(t)
	f.autoMergeStore.AssertNotCalled(t, "Delete", mock.Anything)
	f.assertExpectations(t)
}

func TestState_DraftChangeOnClosedPR_EmitsDraftChanged(t *testing.T) {
	f := newStateFixture(t, stateTestPR(enum.PullReqStateClosed, false))

	// auto merge is only canceled for open PRs.
	f.pullreqStore.On("Update", mock.Anything).Return(nil).Once()

	_, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
		&StateInput{State: enum.PullReqStateClosed, IsDraft: true})
	require.NoError(t, err)

	p := f.rec.expectDraft(t)
	require.True(t, p.IsDraft)

	f.rec.expectNoMore(t)
	f.autoMergeStore.AssertNotCalled(t, "Delete", mock.Anything)
	f.assertExpectations(t)
}

// TestState_CloseAndMarkDraft_EmitsOnlyClosed documents current behavior:
// when a single request closes the PR and flips is_draft, only the Closed event
// is published even though the draft flag changes too.
func TestState_CloseAndMarkDraft_EmitsOnlyClosed(t *testing.T) {
	f := newStateFixture(t, stateTestPR(enum.PullReqStateOpen, false))

	f.autoMergeStore.On("Delete", stateTestPRID).Return(false, nil).Once()
	f.pullreqStore.On("Update", mock.Anything).Run(func(args mock.Arguments) {
		pr, _ := args.Get(0).(*types.PullReq)
		require.Equal(t, enum.PullReqStateClosed, pr.State)
		require.True(t, pr.IsDraft)
	}).Return(nil).Once()
	f.gitClient.On("UpdateRef", mock.Anything, mock.Anything).Return(nil).Once()
	f.repoStore.On("UpdateOptLock", mock.Anything, mock.Anything).
		Return(&types.Repository{ID: stateTestRepoID}, nil).Once()

	_, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
		&StateInput{State: enum.PullReqStateClosed, IsDraft: true})
	require.NoError(t, err)

	f.rec.expectClosed(t)
	f.rec.expectNoMore(t)
	f.assertExpectations(t)
}

func TestState_NoChange_EmitsNothing(t *testing.T) {
	f := newStateFixture(t, stateTestPR(enum.PullReqStateOpen, true))

	// early return: none of the post-change side effects happen.
	f.activityStore.ExpectedCalls = nil
	f.sse.ExpectedCalls = nil

	out, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
		&StateInput{State: enum.PullReqStateOpen, IsDraft: true})
	require.NoError(t, err)
	require.True(t, out.IsDraft)

	f.rec.expectNoMore(t)
	f.pullreqStore.AssertNotCalled(t, "Update", mock.Anything)
	f.activityStore.AssertNotCalled(t, "CreateWithPayload", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	f.pullreqStore.AssertExpectations(t)
	f.repoStore.AssertExpectations(t)
}

func TestState_Reopen_SourceSHAPushedBy(t *testing.T) {
	const (
		oldSHA    = "1111111111111111111111111111111111111111"
		newSHA    = "2222222222222222222222222222222222222222"
		targetSHA = "3333333333333333333333333333333333333333"
		baseSHA   = "4444444444444444444444444444444444444444"

		originalPusher int64 = 42
	)
	reopener := testSession().Principal.ID

	tests := []struct {
		name       string
		branchSHA  string
		pushedBy   *int64
		wantPusher int64
	}{
		{name: "sha changed while closed", branchSHA: newSHA, pushedBy: ptr.Int64(originalPusher), wantPusher: reopener},
		{name: "pusher unknown", branchSHA: oldSHA, pushedBy: nil, wantPusher: reopener},
		{name: "sha unchanged keeps pusher", branchSHA: oldSHA, pushedBy: ptr.Int64(originalPusher),
			wantPusher: originalPusher},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := stateTestPR(enum.PullReqStateClosed, false)
			pr.SourceSHA = oldSHA
			pr.SourceSHAPushedBy = tt.pushedBy
			f := newStateFixture(t, pr)

			branchRef := func(name string) any {
				return mock.MatchedBy(func(p git.GetRefParams) bool { return p.Name == name })
			}
			f.gitClient.On("GetRef", mock.Anything, branchRef("feature")).
				Return(git.GetRefResponse{SHA: sha.Must(tt.branchSHA)}, nil).Once()
			f.gitClient.On("GetRef", mock.Anything, branchRef("main")).
				Return(git.GetRefResponse{SHA: sha.Must(targetSHA)}, nil).Twice()
			f.pullreqStore.On("List", mock.Anything).Return([]*types.PullReq{}, nil).Once()
			f.gitClient.On("MergeBase", mock.Anything, mock.Anything).
				Return(git.MergeBaseOutput{MergeBaseSHA: sha.Must(baseSHA)}, nil).Once()

			var updated *types.PullReq
			f.pullreqStore.On("Update", mock.Anything).Run(func(args mock.Arguments) {
				updated, _ = args.Get(0).(*types.PullReq)
			}).Return(nil).Once()
			f.gitClient.On("UpdateRef", mock.Anything, mock.Anything).Return(nil).Once()
			f.repoStore.On("UpdateOptLock", mock.Anything, mock.Anything).
				Return(&types.Repository{ID: stateTestRepoID}, nil).Once()

			_, err := f.ctrl.State(context.Background(), testSession(), "1", stateTestPRNum,
				&StateInput{State: enum.PullReqStateOpen})
			require.NoError(t, err)

			require.NotNil(t, updated)
			require.Equal(t, enum.PullReqStateOpen, updated.State)
			require.Equal(t, tt.branchSHA, updated.SourceSHA)
			require.NotNil(t, updated.SourceSHAPushedBy)
			require.Equal(t, tt.wantPusher, *updated.SourceSHAPushedBy)

			f.rec.expectReopened(t)
			f.rec.expectNoMore(t)
			f.assertExpectations(t)
		})
	}
}
