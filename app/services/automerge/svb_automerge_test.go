// Copyright 2026 Harness, Inc.
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

package automerge

import (
	"context"
	"errors"
	"testing"

	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbTx struct{ err error }

func (t svbTx) WithTx(ctx context.Context, fn func(context.Context) error, _ ...any) error {
	if t.err != nil {
		return t.err
	}
	return fn(ctx)
}

var _ dbtx.Transactor = svbTx{}

type svbPullReqStore struct {
	store.PullReqStore

	pr        *types.PullReq
	findErr   error
	updateErr error
	list      []*types.PullReq
	listErr   error
	updated   *types.PullReq
	filter    *types.PullReqFilter
}

func (f *svbPullReqStore) Find(context.Context, int64) (*types.PullReq, error) {
	return f.pr, f.findErr
}

func (f *svbPullReqStore) FindByNumber(context.Context, int64, int64) (*types.PullReq, error) {
	return f.pr, f.findErr
}

func (f *svbPullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	f.updated = pr
	return f.updateErr
}

func (f *svbPullReqStore) List(_ context.Context, filter *types.PullReqFilter) ([]*types.PullReq, error) {
	f.filter = filter
	return f.list, f.listErr
}

type svbAutoMergeStore struct {
	store.AutoMergeStore

	am        *types.AutoMerge
	findErr   error
	deleteErr error
	deleted   bool
}

func (f *svbAutoMergeStore) Find(context.Context, int64) (*types.AutoMerge, error) {
	return f.am, f.findErr
}

func (f *svbAutoMergeStore) Delete(context.Context, int64) (bool, error) {
	f.deleted = true
	return true, f.deleteErr
}

type svbPrincipalStore struct {
	store.PrincipalStore

	err error
}

func (f *svbPrincipalStore) Find(_ context.Context, id int64) (*types.Principal, error) {
	return &types.Principal{ID: id}, f.err
}

type svbRepoIDCache struct {
	store.RepoIDCache

	err error
}

func (f *svbRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.RepositoryCore{ID: id}, nil
}

func svbOpenPR() *types.PullReq {
	repoID := int64(1)
	return &types.PullReq{
		ID: 5, Number: 7, State: enum.PullReqStateOpen, SubState: enum.PullReqSubStateAutoMerge,
		MergeCheckStatus: enum.MergeCheckStatusMergeable, SourceRepoID: &repoID, TargetRepoID: 1,
	}
}

func TestSvbIsEligibleForAutoMerge(t *testing.T) {
	tests := []struct {
		name string
		mod  func(pr *types.PullReq)
		want bool
	}{
		{"eligible", func(*types.PullReq) {}, true},
		{"merged", func(pr *types.PullReq) { pr.State = enum.PullReqStateMerged }, false},
		{"wrong substate", func(pr *types.PullReq) { pr.SubState = enum.PullReqSubStateNone }, false},
		{"draft", func(pr *types.PullReq) { pr.IsDraft = true }, false},
		{"conflicts", func(pr *types.PullReq) { pr.MergeCheckStatus = enum.MergeCheckStatusConflict }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pr := svbOpenPR()
			tc.mod(pr)
			require.Equal(t, tc.want, isEligibleForAutoMerge(pr))
		})
	}
}

func TestSvbDisable(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: 1}

	newSvc := func(pr *types.PullReq) (*Service, *svbPullReqStore, *svbAutoMergeStore) {
		ps := &svbPullReqStore{pr: pr}
		as := &svbAutoMergeStore{}
		return &Service{tx: svbTx{}, pullreqStore: ps, autoMergeStore: as}, ps, as
	}

	t.Run("disables", func(t *testing.T) {
		svc, ps, as := newSvc(svbOpenPR())
		pr, err := svc.Disable(ctx, repo, 7)
		require.NoError(t, err)
		require.Equal(t, enum.PullReqSubStateNone, pr.SubState)
		require.Same(t, ps.updated, pr)
		require.True(t, as.deleted)
	})

	t.Run("not open", func(t *testing.T) {
		pr := svbOpenPR()
		pr.State = enum.PullReqStateClosed
		svc, _, _ := newSvc(pr)
		_, err := svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, ErrPullReqNotOpen)
	})

	t.Run("draft", func(t *testing.T) {
		pr := svbOpenPR()
		pr.IsDraft = true
		svc, _, _ := newSvc(pr)
		_, err := svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, ErrPullReqDraft)
	})

	t.Run("not in auto merge", func(t *testing.T) {
		pr := svbOpenPR()
		pr.SubState = enum.PullReqSubStateNone
		svc, _, _ := newSvc(pr)
		_, err := svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, ErrNotInAutoMerge)
	})

	t.Run("store failures", func(t *testing.T) {
		svc, ps, _ := newSvc(svbOpenPR())
		ps.findErr = errSvbBoom
		_, err := svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, errSvbBoom)

		svc, ps, _ = newSvc(svbOpenPR())
		ps.updateErr = errSvbBoom
		_, err = svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, errSvbBoom)

		svc, _, as := newSvc(svbOpenPR())
		as.deleteErr = errSvbBoom
		_, err = svc.Disable(ctx, repo, 7)
		require.ErrorIs(t, err, errSvbBoom)
	})
}

func TestSvbAutoMerge(t *testing.T) {
	ctx := context.Background()

	newSvc := func() (*Service, *svbAutoMergeStore, *svbPrincipalStore, *svbRepoIDCache) {
		as := &svbAutoMergeStore{am: &types.AutoMerge{PullReqID: 5, RequestedBy: 3}}
		ps := &svbPrincipalStore{}
		rc := &svbRepoIDCache{}
		return &Service{
			autoMergeStore: as,
			principalStore: ps,
			repoFinder:     refcache.NewRepoFinder(nil, nil, rc, nil, cache.Evictor[*types.RepositoryCore]{}),
		}, as, ps, rc
	}

	t.Run("not eligible is a no-op", func(t *testing.T) {
		svc, _, _, _ := newSvc()
		pr := svbOpenPR()
		pr.IsDraft = true
		require.NoError(t, svc.autoMerge(ctx, pr))
	})

	t.Run("missing auto merge entry discards the event", func(t *testing.T) {
		svc, as, _, _ := newSvc()
		as.findErr = gitnessstore.ErrResourceNotFound
		err := svc.autoMerge(ctx, svbOpenPR())
		require.Error(t, err)
		var discard interface{ Unwrap() error }
		require.ErrorAs(t, err, &discard)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("auto merge lookup error", func(t *testing.T) {
		svc, as, _, _ := newSvc()
		as.findErr = errSvbBoom
		require.ErrorIs(t, svc.autoMerge(ctx, svbOpenPR()), errSvbBoom)
	})

	t.Run("principal lookup error", func(t *testing.T) {
		svc, _, ps, _ := newSvc()
		ps.err = errSvbBoom
		require.ErrorIs(t, svc.autoMerge(ctx, svbOpenPR()), errSvbBoom)
	})

	t.Run("pull request without source repo is not eligible and is discarded", func(t *testing.T) {
		svc, _, _, _ := newSvc()
		pr := svbOpenPR()
		pr.SourceRepoID = nil
		err := svc.autoMerge(ctx, pr)
		require.ErrorIs(t, err, ErrNotEligible)
	})

	t.Run("merge failure is returned for retry", func(t *testing.T) {
		svc, _, _, rc := newSvc()
		rc.err = errSvbBoom
		require.ErrorIs(t, svc.autoMerge(ctx, svbOpenPR()), errSvbBoom)
	})
}

func TestSvbMergeGuards(t *testing.T) {
	ctx := context.Background()
	failingCache := &svbRepoIDCache{err: errSvbBoom}
	svc := &Service{
		repoFinder: refcache.NewRepoFinder(nil, nil, failingCache, nil, cache.Evictor[*types.RepositoryCore]{}),
	}

	pr := svbOpenPR()
	pr.State = enum.PullReqStateClosed
	_, _, err := svc.Merge(ctx, pr, types.AutoMergeInput{})
	require.ErrorIs(t, err, ErrNotEligible)

	pr = svbOpenPR()
	pr.IsDraft = true
	_, _, err = svc.Merge(ctx, pr, types.AutoMergeInput{})
	require.ErrorIs(t, err, ErrNotEligible)

	pr = svbOpenPR()
	pr.SourceRepoID = nil
	_, _, err = svc.Merge(ctx, pr, types.AutoMergeInput{})
	require.ErrorIs(t, err, ErrNotEligible)

	_, _, err = svc.Merge(ctx, svbOpenPR(), types.AutoMergeInput{})
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbHandlers(t *testing.T) {
	ctx := context.Background()

	// A closed pull request is never eligible, so the handlers only exercise lookups and filters.
	closedPR := func() *types.PullReq {
		pr := svbOpenPR()
		pr.State = enum.PullReqStateClosed
		return pr
	}
	newSvc := func(ps *svbPullReqStore) *Service { return &Service{pullreqStore: ps} }

	t.Run("merge check succeeded", func(t *testing.T) {
		ev := &events.Event[*pullreqevents.MergeCheckSucceededPayload]{
			Payload: &pullreqevents.MergeCheckSucceededPayload{Base: pullreqevents.Base{PullReqID: 5}},
		}
		require.NoError(t, newSvc(&svbPullReqStore{pr: closedPR()}).mergePRsOnMergeCheckSucceeded(ctx, ev))
		require.ErrorIs(t, newSvc(&svbPullReqStore{findErr: errSvbBoom}).mergePRsOnMergeCheckSucceeded(ctx, ev),
			errSvbBoom)
	})

	t.Run("approval", func(t *testing.T) {
		ev := &events.Event[*pullreqevents.ReviewSubmittedPayload]{
			Payload: &pullreqevents.ReviewSubmittedPayload{Decision: enum.PullReqReviewDecisionChangeReq},
		}
		// Non-approvals are ignored without touching the store.
		require.NoError(t, newSvc(&svbPullReqStore{findErr: errSvbBoom}).mergePRsOnApproval(ctx, ev))

		ev.Payload.Decision = enum.PullReqReviewDecisionApproved
		require.NoError(t, newSvc(&svbPullReqStore{pr: closedPR()}).mergePRsOnApproval(ctx, ev))
		require.ErrorIs(t, newSvc(&svbPullReqStore{findErr: errSvbBoom}).mergePRsOnApproval(ctx, ev), errSvbBoom)
	})

	t.Run("comment resolved", func(t *testing.T) {
		ev := &events.Event[*pullreqevents.CommentStatusUpdatedPayload]{
			Payload: &pullreqevents.CommentStatusUpdatedPayload{
				OldStatus: enum.PullReqCommentStatusResolved, NewStatus: enum.PullReqCommentStatusActive,
			},
		}
		require.NoError(t, newSvc(&svbPullReqStore{findErr: errSvbBoom}).mergePRsOnCommentResolve(ctx, ev))

		ev.Payload.OldStatus = enum.PullReqCommentStatusActive
		ev.Payload.NewStatus = enum.PullReqCommentStatusResolved
		require.NoError(t, newSvc(&svbPullReqStore{pr: closedPR()}).mergePRsOnCommentResolve(ctx, ev))
		require.ErrorIs(t, newSvc(&svbPullReqStore{findErr: errSvbBoom}).mergePRsOnCommentResolve(ctx, ev),
			errSvbBoom)
	})

	t.Run("check reported", func(t *testing.T) {
		const validSHA = "1111111111111111111111111111111111111111"
		mk := func(status enum.CheckStatus, bypassed bool, commit string) *events.Event[*checkevents.ReportedPayload] {
			return &events.Event[*checkevents.ReportedPayload]{Payload: &checkevents.ReportedPayload{
				Base: checkevents.Base{RepoID: 1, SHA: commit}, Status: status, Bypassed: bypassed,
			}}
		}

		// Failed, non-bypassed checks are ignored.
		ps := &svbPullReqStore{}
		require.NoError(t, newSvc(ps).mergePRsOnCheckSucceeded(ctx, mk(enum.CheckStatusFailure, false, validSHA)))
		require.Nil(t, ps.filter)

		// Invalid SHA is a discarded event.
		err := newSvc(ps).mergePRsOnCheckSucceeded(ctx, mk(enum.CheckStatusSuccess, false, "zz"))
		require.Error(t, err)

		// Successful and bypassed checks look up the open pull requests of the commit.
		for _, ev := range []*events.Event[*checkevents.ReportedPayload]{
			mk(enum.CheckStatusSuccess, false, validSHA),
			mk(enum.CheckStatusFailure, true, validSHA),
		} {
			ps = &svbPullReqStore{list: []*types.PullReq{closedPR()}}
			require.NoError(t, newSvc(ps).mergePRsOnCheckSucceeded(ctx, ev))
			require.NotNil(t, ps.filter)
			require.Equal(t, int64(1), ps.filter.SourceRepoID)
			require.Equal(t, sha.Must(validSHA), ps.filter.SourceSHA)
		}

		// A failing list does not fail the event.
		ps = &svbPullReqStore{listErr: errSvbBoom}
		require.NoError(t, newSvc(ps).mergePRsOnCheckSucceeded(ctx, mk(enum.CheckStatusSuccess, false, validSHA)))
	})
}

func TestSvbForEveryOpenPullReqWithSourceSHA(t *testing.T) {
	ctx := context.Background()
	ps := &svbPullReqStore{list: []*types.PullReq{{Number: 1}, {Number: 2}, {Number: 3}}}
	svc := &Service{pullreqStore: ps}

	var seen []int64
	err := svc.forEveryOpenPullReqWithSourceSHA(ctx, 1, sha.Must("1111111111111111111111111111111111111111"),
		func(pr *types.PullReq) error {
			seen = append(seen, pr.Number)
			if pr.Number == 2 {
				return errSvbBoom // a failing callback does not stop the iteration
			}
			return nil
		})
	require.NoError(t, err)
	require.Equal(t, []int64{1, 2, 3}, seen)
	require.Equal(t, enum.PullReqStateOpen, ps.filter.States[0])
	require.False(t, *ps.filter.IsDraft)

	ps.listErr = errSvbBoom
	err = svc.forEveryOpenPullReqWithSourceSHA(ctx, 1, sha.SHA{}, nil)
	require.ErrorIs(t, err, errSvbBoom)
}

type svbConsumer struct {
	registered []string
	startErr   error
}

func (c *svbConsumer) Register(streamID string, _ stream.HandlerFunc, _ ...stream.HandlerOption) error {
	c.registered = append(c.registered, streamID)
	return nil
}

func (c *svbConsumer) Configure(...stream.ConsumerOption) {}

func (c *svbConsumer) Start(context.Context) error { return c.startErr }

func (c *svbConsumer) Errors() <-chan error { return make(chan error) }

func (c *svbConsumer) Infos() <-chan string { return make(chan string) }

type svbProducer struct{}

func (svbProducer) Send(context.Context, string, map[string]any) (string, error) { return "1", nil }

func svbEventSystem(t *testing.T, consumer *svbConsumer, factoryErr error) *events.System {
	t.Helper()
	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) {
			if factoryErr != nil {
				return nil, factoryErr
			}
			return consumer, nil
		},
		svbProducer{},
		events.NewNoopCollector(),
		false,
	)
	require.NoError(t, err)
	return system
}

func TestSvbNewService(t *testing.T) {
	ctx := context.Background()
	cfg := &types.Config{InstanceID: "inst"}
	cfg.AutoMerge.CheckEventsConcurrency = 1
	cfg.AutoMerge.PullReqEventsConcurrency = 1

	t.Run("registers handlers", func(t *testing.T) {
		consumer := &svbConsumer{}
		system := svbEventSystem(t, consumer, nil)
		checkFactory, err := checkevents.NewReaderFactory(system)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(system)
		require.NoError(t, err)

		svc, err := NewService(
			ctx, cfg, nil, nil, nil, nil, checkFactory, prFactory,
			refcache.RepoFinder{}, nil, nil, nil, nil, nil, nil, nil,
		)
		require.NoError(t, err)
		require.NotNil(t, svc)
		// 1 check event + 3 pull request events.
		require.Len(t, consumer.registered, 4)
	})

	t.Run("reader launch failure", func(t *testing.T) {
		system := svbEventSystem(t, nil, errSvbBoom)
		checkFactory, err := checkevents.NewReaderFactory(system)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(system)
		require.NoError(t, err)

		_, err = NewService(
			ctx, cfg, nil, nil, nil, nil, checkFactory, prFactory,
			refcache.RepoFinder{}, nil, nil, nil, nil, nil, nil, nil,
		)
		require.ErrorIs(t, err, errSvbBoom)
	})

	t.Run("pull request reader launch failure", func(t *testing.T) {
		goodSystem := svbEventSystem(t, &svbConsumer{}, nil)
		badSystem := svbEventSystem(t, nil, errSvbBoom)
		checkFactory, err := checkevents.NewReaderFactory(goodSystem)
		require.NoError(t, err)
		prFactory, err := pullreqevents.NewReaderFactory(badSystem)
		require.NoError(t, err)

		_, err = NewService(
			ctx, cfg, nil, nil, nil, nil, checkFactory, prFactory,
			refcache.RepoFinder{}, nil, nil, nil, nil, nil, nil, nil,
		)
		require.ErrorIs(t, err, errSvbBoom)
	})
}
