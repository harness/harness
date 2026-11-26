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
	"sync"
	"testing"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/codecomments"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	mockpullreq "github.com/harness/gitness/mocks/pullreq"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

// svaPubSub records published topics and payloads.
type svaPubSub struct {
	pubsub.PubSub

	mu       sync.Mutex
	topics   []string
	payloads []string
	err      error
}

func (p *svaPubSub) Publish(_ context.Context, topic string, payload []byte, _ ...pubsub.PublishOption) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.topics = append(p.topics, topic)
	p.payloads = append(p.payloads, string(payload))
	return p.err
}

func svaBase(number int64) pullreqevents.Base {
	return pullreqevents.Base{PullReqID: 100 + number, TargetRepoID: 1, Number: number, PrincipalID: 3}
}

func TestSvaMergeCheckEventsPassExpectedSHAs(t *testing.T) {
	t.Parallel()

	const newSHA = "2222222222222222222222222222222222222222"
	const oldSHA = "1111111111111111111111111111111111111111"

	tests := []struct {
		name    string
		run     func(s *Service) error
		wantOld string
	}{
		{
			name: "created",
			run: func(s *Service) error {
				return s.mergeCheckOnCreated(context.Background(), &events.Event[*pullreqevents.CreatedPayload]{
					Payload: &pullreqevents.CreatedPayload{Base: svaBase(4), SourceSHA: newSHA},
				})
			},
			wantOld: sha.Nil.String(),
		},
		{
			name: "branch updated",
			run: func(s *Service) error {
				return s.mergeCheckOnBranchUpdate(context.Background(),
					&events.Event[*pullreqevents.BranchUpdatedPayload]{
						Payload: &pullreqevents.BranchUpdatedPayload{Base: svaBase(4), OldSHA: oldSHA, NewSHA: newSHA},
					})
			},
			wantOld: oldSHA,
		},
		{
			name: "target branch changed",
			run: func(s *Service) error {
				return s.mergeCheckOnTargetBranchChange(context.Background(),
					&events.Event[*pullreqevents.TargetBranchChangedPayload]{
						Payload: &pullreqevents.TargetBranchChangedPayload{Base: svaBase(4), SourceSHA: newSHA},
					})
			},
			wantOld: sha.None.String(),
		},
		{
			name: "reopened",
			run: func(s *Service) error {
				return s.mergeCheckOnReopen(context.Background(), &events.Event[*pullreqevents.ReopenedPayload]{
					Payload: &pullreqevents.ReopenedPayload{Base: svaBase(4), SourceSHA: newSHA},
				})
			},
			wantOld: sha.None.String(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			bus := &svaPubSub{err: errSvaBoom}
			var gotRepoID, gotNumber int64
			prs := &svaPullReqStore{findByNumber: func(_ context.Context, repoID, number int64) (*types.PullReq, error) {
				gotRepoID, gotNumber = repoID, number
				return &types.PullReq{ID: 1, Number: number, State: enum.PullReqStateOpen}, nil
			}}
			svc := &Service{pullreqStore: prs, pubsub: bus, cancelMergeability: map[string]context.CancelFunc{}}

			err := test.run(svc)
			require.ErrorIs(t, err, errSvaBoom)
			require.Equal(t, int64(1), gotRepoID)
			require.Equal(t, int64(4), gotNumber)
			require.Equal(t, []string{cancelMergeCheckKey}, bus.topics)
			require.Equal(t, []string{test.wantOld}, bus.payloads)
		})
	}
}

func TestSvaUpdateMergeDataEarlyExits(t *testing.T) {
	t.Parallel()

	const newSHA = "2222222222222222222222222222222222222222"

	t.Run("pull request lookup fails", func(t *testing.T) {
		t.Parallel()

		prs := &svaPullReqStore{findByNumber: func(context.Context, int64, int64) (*types.PullReq, error) {
			return nil, errSvaBoom
		}}
		svc := &Service{pullreqStore: prs}

		err := svc.updateMergeData(context.Background(), 1, 2, "", newSHA, 3)
		require.ErrorIs(t, err, errSvaBoom)
	})

	t.Run("closed pull request is rejected", func(t *testing.T) {
		t.Parallel()

		prs := &svaPullReqStore{findByNumber: func(context.Context, int64, int64) (*types.PullReq, error) {
			return &types.PullReq{Number: 2, State: enum.PullReqStateMerged}, nil
		}}
		svc := &Service{pullreqStore: prs, pubsub: &svaPubSub{}}

		err := svc.updateMergeData(context.Background(), 1, 2, "", newSHA, 3)
		require.ErrorContains(t, err, "closed PR 2")
	})

	t.Run("duplicate check on the same SHA is skipped", func(t *testing.T) {
		t.Parallel()

		prs := &svaPullReqStore{findByNumber: func(context.Context, int64, int64) (*types.PullReq, error) {
			return &types.PullReq{Number: 2, State: enum.PullReqStateOpen}, nil
		}}
		cancelled := false
		svc := &Service{
			pullreqStore: prs,
			pubsub:       &svaPubSub{},
			cancelMergeability: map[string]context.CancelFunc{
				newSHA: func() { cancelled = true },
			},
		}

		require.NoError(t, svc.updateMergeData(context.Background(), 1, 2, "", newSHA, 3))
		require.False(t, cancelled, "the running check must not be cancelled by the duplicate")
		require.Contains(t, svc.cancelMergeability, newSHA)
	})

	t.Run("repository lookup failure releases the SHA", func(t *testing.T) {
		t.Parallel()

		prs := &svaPullReqStore{findByNumber: func(context.Context, int64, int64) (*types.PullReq, error) {
			return &types.PullReq{Number: 2, State: enum.PullReqStateOpen, TargetRepoID: 1}, nil
		}}
		svc := &Service{
			pullreqStore: prs,
			pubsub:       &svaPubSub{},
			repoFinder: svaNewRepoFinder(func(context.Context, int64) (*types.RepositoryCore, error) {
				return nil, errSvaBoom
			}),
			cancelMergeability: map[string]context.CancelFunc{},
		}

		err := svc.updateMergeData(context.Background(), 1, 2, "", newSHA, 3)
		require.ErrorIs(t, err, errSvaBoom)
		require.Empty(t, svc.cancelMergeability)
	})
}

type svaFileViewStore struct {
	store.PullReqFileViewStore

	prID  int64
	paths []string
	err   error
	calls int
}

func (s *svaFileViewStore) MarkObsolete(_ context.Context, prID int64, paths []string) error {
	s.calls++
	s.prID = prID
	s.paths = paths
	return s.err
}

type svaDiffGit struct {
	git.Interface

	diffs   []*git.FileDiff
	diffErr error
	params  *git.DiffParams
}

func (g *svaDiffGit) Diff(
	_ context.Context,
	params *git.DiffParams,
	_ ...api.FileDiffRequest,
) (<-chan *git.FileDiff, <-chan error) {
	g.params = params
	if g.diffErr != nil {
		errCh := make(chan error, 1)
		errCh <- g.diffErr
		return nil, errCh
	}

	ch := make(chan *git.FileDiff, len(g.diffs))
	for _, diff := range g.diffs {
		ch <- diff
	}
	close(ch)

	return ch, nil
}

func TestSvaHandleFileViewedOnBranchUpdate(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, GitUID: "git-uid"}
	event := &events.Event[*pullreqevents.BranchUpdatedPayload]{
		Payload: &pullreqevents.BranchUpdatedPayload{
			Base:   pullreqevents.Base{PullReqID: 55, TargetRepoID: 1},
			OldSHA: "old",
			NewSHA: "new",
		},
	}

	tests := []struct {
		name      string
		diffs     []*git.FileDiff
		diffErr   error
		storeErr  error
		wantErr   bool
		wantPaths []string
		wantMark  bool
	}{
		{
			name: "all change kinds",
			diffs: []*git.FileDiff{
				{Status: gitenum.FileDiffStatusAdded, Path: "added.go"},
				{Status: gitenum.FileDiffStatusDeleted, OldPath: "deleted.go", Path: "deleted.go"},
				{Status: gitenum.FileDiffStatusRenamed, OldPath: "old.go", Path: "new.go"},
				{Status: gitenum.FileDiffStatusModified, Path: "modified.go"},
				{Status: gitenum.FileDiffStatusCopied, Path: "copied.go"},
				{Status: gitenum.FileDiffStatusUndefined, Path: "undefined.go"},
			},
			wantMark:  true,
			wantPaths: []string{"added.go", "deleted.go", "old.go", "new.go", "modified.go"},
		},
		{name: "no changes does not touch the store"},
		{
			name:     "only copied files does not touch the store",
			diffs:    []*git.FileDiff{{Status: gitenum.FileDiffStatusCopied, Path: "copied.go"}},
			wantMark: false,
		},
		{name: "diff error", diffErr: errSvaBoom, wantErr: true},
		{
			name:      "store error",
			diffs:     []*git.FileDiff{{Status: gitenum.FileDiffStatusAdded, Path: "a"}},
			storeErr:  errSvaBoom,
			wantErr:   true,
			wantMark:  true,
			wantPaths: []string{"a"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			gitSvc := &svaDiffGit{diffs: test.diffs, diffErr: test.diffErr}
			views := &svaFileViewStore{err: test.storeErr}
			svc := &Service{git: gitSvc, fileViewStore: views, repoFinder: svaFixedRepoFinder(repo)}

			err := svc.handleFileViewedOnBranchUpdate(context.Background(), event)
			if test.wantErr {
				require.ErrorIs(t, err, errSvaBoom)
			} else {
				require.NoError(t, err)
			}

			if test.wantMark {
				require.Equal(t, 1, views.calls)
				require.Equal(t, int64(55), views.prID)
				require.Equal(t, test.wantPaths, views.paths)
			} else {
				require.Zero(t, views.calls)
			}

			if test.diffErr == nil {
				require.Equal(t, "git-uid", gitSvc.params.RepoUID)
				require.Equal(t, "old", gitSvc.params.BaseRef)
				require.Equal(t, "new", gitSvc.params.HeadRef)
				require.False(t, gitSvc.params.IncludePatch)
			}
		})
	}

	t.Run("repository lookup error", func(t *testing.T) {
		t.Parallel()

		svc := &Service{repoFinder: svaNewRepoFinder(func(context.Context, int64) (*types.RepositoryCore, error) {
			return nil, errSvaBoom
		})}
		require.ErrorIs(t, svc.handleFileViewedOnBranchUpdate(context.Background(), event), errSvaBoom)
	})
}

type svaCodeCommentView struct {
	store.CodeCommentView

	atMergeBase []*types.CodeComment
	atSource    []*types.CodeComment
	mergeErr    error
	sourceErr   error
	updateErr   error
	updateCalls [][]*types.CodeComment
	updateFails int // The 1-based UpdateAll call that fails (0 means never).
}

func (v *svaCodeCommentView) ListNotAtMergeBaseSHA(context.Context, int64, string) ([]*types.CodeComment, error) {
	return v.atMergeBase, v.mergeErr
}

func (v *svaCodeCommentView) ListNotAtSourceSHA(context.Context, int64, string) ([]*types.CodeComment, error) {
	return v.atSource, v.sourceErr
}

func (v *svaCodeCommentView) UpdateAll(_ context.Context, comments []*types.CodeComment) error {
	v.updateCalls = append(v.updateCalls, comments)
	if v.updateFails == len(v.updateCalls) {
		return v.updateErr
	}
	return nil
}

// svaHunkGit returns an empty diff, meaning no changes happened between the compared commits.
type svaHunkGit struct {
	git.Interface
}

func (svaHunkGit) GetDiffHunkHeaders(
	context.Context, git.GetDiffHunkHeadersParams,
) (git.GetDiffHunkHeadersOutput, error) {
	return git.GetDiffHunkHeadersOutput{}, nil
}

func TestSvaUpdateCodeComments(t *testing.T) {
	t.Parallel()

	repo := &types.RepositoryCore{ID: 1, GitUID: "git-uid"}

	newService := func(view *svaCodeCommentView, repoErr error) *Service {
		return &Service{
			repoFinder: svaNewRepoFinder(func(context.Context, int64) (*types.RepositoryCore, error) {
				if repoErr != nil {
					return nil, repoErr
				}
				return repo, nil
			}),
			codeCommentView:     view,
			codeCommentMigrator: codecomments.ProvideMigrator(svaHunkGit{}),
		}
	}

	t.Run("migrates both merge base and source comments", func(t *testing.T) {
		t.Parallel()

		oldBase := &types.CodeComment{ID: 1, CodeCommentFields: types.CodeCommentFields{MergeBaseSHA: "oldbase", Path: "a"}}
		oldSrc := &types.CodeComment{ID: 2, CodeCommentFields: types.CodeCommentFields{SourceSHA: "oldsrc", Path: "b"}}
		view := &svaCodeCommentView{atMergeBase: []*types.CodeComment{oldBase}, atSource: []*types.CodeComment{oldSrc}}

		err := newService(view, nil).updateCodeComments(context.Background(), 1, 9, "newsrc", "newbase")
		require.NoError(t, err)
		require.Len(t, view.updateCalls, 2)
		require.Equal(t, "newbase", oldBase.MergeBaseSHA)
		require.Equal(t, "newsrc", oldSrc.SourceSHA)
	})

	t.Run("branch update event", func(t *testing.T) {
		t.Parallel()

		comment := &types.CodeComment{ID: 2, CodeCommentFields: types.CodeCommentFields{SourceSHA: "oldsrc", Path: "b"}}
		view := &svaCodeCommentView{atSource: []*types.CodeComment{comment}}

		err := newService(view, nil).updateCodeCommentsOnBranchUpdate(context.Background(),
			&events.Event[*pullreqevents.BranchUpdatedPayload]{
				Payload: &pullreqevents.BranchUpdatedPayload{
					Base: svaBase(1), NewSHA: "newsrc", NewMergeBaseSHA: "newbase",
				},
			})
		require.NoError(t, err)
		require.Equal(t, "newsrc", comment.SourceSHA)
	})

	t.Run("reopen event", func(t *testing.T) {
		t.Parallel()

		comment := &types.CodeComment{ID: 2, CodeCommentFields: types.CodeCommentFields{MergeBaseSHA: "oldbase", Path: "b"}}
		view := &svaCodeCommentView{atMergeBase: []*types.CodeComment{comment}}

		err := newService(view, nil).updateCodeCommentsOnReopen(context.Background(),
			&events.Event[*pullreqevents.ReopenedPayload]{
				Payload: &pullreqevents.ReopenedPayload{Base: svaBase(1), SourceSHA: "newsrc", MergeBaseSHA: "newbase"},
			})
		require.NoError(t, err)
		require.Equal(t, "newbase", comment.MergeBaseSHA)
	})

	errTests := []struct {
		name    string
		view    *svaCodeCommentView
		repoErr error
	}{
		{name: "repository", view: &svaCodeCommentView{}, repoErr: errSvaBoom},
		{name: "list at merge base", view: &svaCodeCommentView{mergeErr: errSvaBoom}},
		{name: "update after merge base", view: &svaCodeCommentView{updateErr: errSvaBoom, updateFails: 1}},
		{name: "list at source", view: &svaCodeCommentView{sourceErr: errSvaBoom}},
		{name: "update after source", view: &svaCodeCommentView{updateErr: errSvaBoom, updateFails: 2}},
	}
	for _, test := range errTests {
		t.Run("error "+test.name, func(t *testing.T) {
			t.Parallel()

			err := newService(test.view, test.repoErr).updateCodeComments(context.Background(), 1, 9, "s", "b")
			require.ErrorIs(t, err, errSvaBoom)
		})
	}
}

func TestSvaGetBranchFromRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ref     string
		want    string
		wantErr bool
	}{
		{ref: "refs/heads/main", want: "main"},
		{ref: "refs/heads/feature/x", want: "feature/x"},
		{ref: "refs/heads/", wantErr: true},
		{ref: "refs/tags/v1", wantErr: true},
		{ref: "", wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.ref, func(t *testing.T) {
			t.Parallel()

			got, err := getBranchFromRef(test.ref)
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.want, got)
		})
	}
}

func TestSvaForEveryOpenPR(t *testing.T) {
	t.Parallel()

	t.Run("filters by source branch and visits all open pull requests", func(t *testing.T) {
		t.Parallel()

		var gotFilter *types.PullReqFilter
		prs := &svaPullReqStore{list: func(_ context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
			gotFilter = opts
			return []*types.PullReq{{Number: 1}, {Number: 2}, {Number: 3}}, nil
		}}
		svc := &Service{pullreqStore: prs}

		var visited []int64
		svc.forEveryOpenPR(context.Background(), 7, "refs/heads/feature", func(pr *types.PullReq) error {
			visited = append(visited, pr.Number)
			if pr.Number == 2 {
				return errSvaBoom // failures must not stop the iteration
			}
			return nil
		})

		require.Equal(t, []int64{1, 2, 3}, visited)
		require.Equal(t, int64(7), gotFilter.SourceRepoID)
		require.Equal(t, "feature", gotFilter.SourceBranch)
		require.Equal(t, []enum.PullReqState{enum.PullReqStateOpen}, gotFilter.States)
	})

	t.Run("invalid ref does not query the store", func(t *testing.T) {
		t.Parallel()

		svc := &Service{pullreqStore: &svaPullReqStore{}}
		svc.forEveryOpenPR(context.Background(), 7, "refs/tags/x", func(*types.PullReq) error {
			t.Fatal("must not be called")
			return nil
		})
	})

	t.Run("list error", func(t *testing.T) {
		t.Parallel()

		prs := &svaPullReqStore{list: func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
			return nil, errSvaBoom
		}}
		svc := &Service{pullreqStore: prs}
		svc.forEveryOpenPR(context.Background(), 7, "refs/heads/x", func(*types.PullReq) error {
			t.Fatal("must not be called")
			return nil
		})
	})
}

type svaBranchDeleteFixture struct {
	svc      *Service
	activity *svaActivityStore
	sse      *svaSSE
	repo     *types.Repository
	pr       *types.PullReq
}

func svaNewBranchDeleteFixture(t *testing.T, prState enum.PullReqState) *svaBranchDeleteFixture {
	t.Helper()

	srcRepoID := int64(1)
	fixture := &svaBranchDeleteFixture{
		activity: &svaActivityStore{},
		sse:      &svaSSE{},
		repo:     &types.Repository{ID: 1, ParentID: 10, NumOpenPulls: 1, DefaultBranch: "main"},
		pr: &types.PullReq{
			ID: 99, Number: 5, State: prState, SourceRepoID: &srcRepoID, TargetRepoID: 1,
			SourceBranch: "feature", TargetBranch: "main", SourceSHA: "src", ActivitySeq: 5,
			MergeSHA: new(string),
		},
	}

	prs := &svaPullReqStore{
		list: func(_ context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
			if opts.SourceBranch != "" {
				return []*types.PullReq{fixture.pr}, nil
			}
			return nil, nil
		},
		updateOptLck: svaApplyUpdate,
	}
	repos := &svaRepoStore{
		find: func(context.Context, int64) (*types.Repository, error) { return fixture.repo, nil },
		updateOpt: func(_ context.Context, repo *types.Repository, fn func(*types.Repository) error,
		) (*types.Repository, error) {
			return repo, fn(repo)
		},
	}

	fixture.svc = &Service{
		pullreqStore:      prs,
		repoStore:         repos,
		activityStore:     fixture.activity,
		sseStreamer:       fixture.sse,
		pullreqEvReporter: mockpullreq.NewStubReporter(t),
	}

	return fixture
}

func svaBranchDeletedEvent(ref string) *events.Event[*gitevents.BranchDeletedPayload] {
	return &events.Event[*gitevents.BranchDeletedPayload]{
		Payload: &gitevents.BranchDeletedPayload{RepoID: 1, PrincipalID: 8, Ref: ref},
	}
}

func TestSvaHandleBranchDeletedClosesOpenPullRequest(t *testing.T) {
	t.Parallel()

	fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateOpen)

	require.NoError(t, fixture.svc.handleBranchDeleted(context.Background(), svaBranchDeletedEvent("refs/heads/feature")))

	require.Equal(t, enum.PullReqStateClosed, fixture.pr.State)
	require.Equal(t, enum.PullReqSubStateNone, fixture.pr.SubState)
	require.Nil(t, fixture.pr.MergeSHA)
	require.Equal(t, 1, fixture.repo.NumClosedPulls)
	require.Equal(t, 0, fixture.repo.NumOpenPulls)

	require.Len(t, fixture.activity.created, 2)
	branchDelete, ok := fixture.activity.created[0].(*types.PullRequestActivityPayloadBranchDelete)
	require.True(t, ok)
	require.Equal(t, "src", branchDelete.SHA)
	stateChange, ok := fixture.activity.created[1].(*types.PullRequestActivityPayloadStateChange)
	require.True(t, ok)
	require.Equal(t, enum.PullReqStateOpen, stateChange.Old)
	require.Equal(t, enum.PullReqStateClosed, stateChange.New)
	// The branch deletion is recorded before the closure.
	require.Equal(t, []int64{6, 7}, fixture.activity.seqs)

	require.Equal(t, []int64{10}, fixture.sse.spaceIDs)
}

func TestSvaHandleBranchDeletedSkipsNonOpenPullRequest(t *testing.T) {
	t.Parallel()

	fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateMerged)

	require.NoError(t, fixture.svc.handleBranchDeleted(context.Background(), svaBranchDeletedEvent("refs/heads/feature")))

	require.Equal(t, enum.PullReqStateMerged, fixture.pr.State)
	require.Empty(t, fixture.activity.created)
	require.Empty(t, fixture.sse.published)
	require.Equal(t, 1, fixture.repo.NumOpenPulls)
}

func TestSvaHandleBranchDeletedToleratesBestEffortFailures(t *testing.T) {
	t.Parallel()

	fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateOpen)
	fixture.activity.err = errSvaBoom
	repos, ok := fixture.svc.repoStore.(*svaRepoStore)
	require.True(t, ok)
	repos.updateOpt = func(context.Context, *types.Repository, func(*types.Repository) error) (*types.Repository, error) {
		return nil, errSvaBoom
	}

	require.NoError(t, fixture.svc.handleBranchDeleted(context.Background(), svaBranchDeletedEvent("refs/heads/feature")))

	require.Equal(t, enum.PullReqStateClosed, fixture.pr.State)
	require.Len(t, fixture.activity.created, 2)
	require.Len(t, fixture.sse.published, 1)
}

func TestSvaHandleBranchDeletedPullRequestFailures(t *testing.T) {
	t.Parallel()

	t.Run("repository lookup failure leaves the pull request open", func(t *testing.T) {
		t.Parallel()

		fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateOpen)
		repos, ok := fixture.svc.repoStore.(*svaRepoStore)
		require.True(t, ok)
		repos.find = func(context.Context, int64) (*types.Repository, error) { return nil, errSvaBoom }

		require.NoError(t, fixture.svc.closePullReqOnBranchDelete(
			context.Background(), svaBranchDeletedEvent("refs/heads/feature")))
		require.Equal(t, enum.PullReqStateOpen, fixture.pr.State)
		require.Empty(t, fixture.activity.created)
	})

	t.Run("update failure leaves no activity", func(t *testing.T) {
		t.Parallel()

		fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateOpen)
		prs, ok := fixture.svc.pullreqStore.(*svaPullReqStore)
		require.True(t, ok)
		prs.updateOptLck = func(context.Context, *types.PullReq, func(*types.PullReq) error) (*types.PullReq, error) {
			return nil, errSvaBoom
		}

		require.NoError(t, fixture.svc.closePullReqOnBranchDelete(
			context.Background(), svaBranchDeletedEvent("refs/heads/feature")))
		require.Empty(t, fixture.activity.created)
		require.Empty(t, fixture.sse.published)
	})

	t.Run("invalid ref is discarded by the target branch handler", func(t *testing.T) {
		t.Parallel()

		fixture := svaNewBranchDeleteFixture(t, enum.PullReqStateOpen)

		err := fixture.svc.handleBranchDeleted(context.Background(), svaBranchDeletedEvent("refs/tags/v1"))
		require.Error(t, err)
		require.Equal(t, enum.PullReqStateOpen, fixture.pr.State)
	})
}

func TestSvaUpdatePullReqTargetOnBranchDeleteGuards(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		list    func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error)
		repo    *types.Repository
		repoErr error
		wantErr bool
	}{
		{
			name:    "list failure",
			list:    func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) { return nil, errSvaBoom },
			wantErr: true,
		},
		{
			name: "no pull requests target the branch",
			list: func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) { return nil, nil },
		},
		{
			name: "repository lookup failure",
			list: func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
				return []*types.PullReq{{}}, nil
			},
			repoErr: errSvaBoom,
			wantErr: true,
		},
		{
			name: "repository without default branch",
			list: func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
				return []*types.PullReq{{}}, nil
			},
			repo: &types.Repository{},
		},
		{
			name: "default branch was deleted",
			list: func(context.Context, *types.PullReqFilter) ([]*types.PullReq, error) {
				return []*types.PullReq{{}}, nil
			},
			repo: &types.Repository{DefaultBranch: "feature"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			svc := &Service{
				pullreqStore: &svaPullReqStore{list: test.list},
				repoStore: &svaRepoStore{find: func(context.Context, int64) (*types.Repository, error) {
					return test.repo, test.repoErr
				}},
			}

			err := svc.updatePullReqTargetOnBranchDelete(context.Background(), svaBranchDeletedEvent("refs/heads/feature"))
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
