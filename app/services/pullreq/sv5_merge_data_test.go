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

package pullreq

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnessurl "github.com/harness/gitness/app/url"
	gitnesserrors "github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/pubsub"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSv5Boom = errors.New("boom")

const (
	sv5SHAa = "1111111111111111111111111111111111111111"
	sv5SHAb = "2222222222222222222222222222222222222222"
	sv5SHAc = "3333333333333333333333333333333333333333"
)

type sv5PrincipalStore struct{ store.PrincipalStore }

func (sv5PrincipalStore) FindServiceByUID(context.Context, string) (*types.Service, error) {
	return &types.Service{ID: 77, UID: "sys", DisplayName: "System", Email: "sys@example.com", Admin: true}, nil
}

// sv5Booted is set once at package initialization: the system service principal is package-level state.
var sv5Booted = sv5InitBootstrap()

func sv5InitBootstrap() bool {
	cfg := &types.Config{}
	cfg.Principal.System.UID = "sys"
	ctrl := service.NewController(check.PrincipalUIDDefault, nil, sv5PrincipalStore{})
	if err := bootstrap.SystemService(context.Background(), cfg, ctrl); err != nil {
		panic(err)
	}
	return true
}

type sv5PubSub struct {
	pubsub.PubSub

	err    error
	topics []string
	data   []string
}

func (p *sv5PubSub) Publish(_ context.Context, topic string, payload []byte, _ ...pubsub.PublishOption) error {
	p.topics = append(p.topics, topic)
	p.data = append(p.data, string(payload))
	return p.err
}

type sv5PullReqStore struct {
	store.PullReqStore

	pr        *types.PullReq
	findErr   error
	updateErr error
	updated   *types.PullReq
	closed    *types.PullReq
}

func (s *sv5PullReqStore) FindByNumber(context.Context, int64, int64) (*types.PullReq, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	cp := *s.pr
	return &cp, nil
}

func (s *sv5PullReqStore) UpdateMergeCheckMetadataOptLock(
	_ context.Context, pr *types.PullReq, fn func(*types.PullReq) error,
) (*types.PullReq, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	cp := *pr
	if err := fn(&cp); err != nil {
		return nil, err
	}
	s.updated = &cp
	return &cp, nil
}

func (s *sv5PullReqStore) UpdateOptLock(
	_ context.Context, pr *types.PullReq, fn func(*types.PullReq) error,
) (*types.PullReq, error) {
	if s.updateErr != nil {
		return nil, s.updateErr
	}
	cp := *pr
	if err := fn(&cp); err != nil {
		return nil, err
	}
	s.closed = &cp
	return &cp, nil
}

type sv5RepoStore struct {
	store.RepoStore

	err     error
	updated *types.Repository
}

func (s *sv5RepoStore) Find(context.Context, int64) (*types.Repository, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &types.Repository{ID: 10, NumOpenPulls: 2}, nil
}

func (s *sv5RepoStore) UpdateOptLock(
	_ context.Context, r *types.Repository, fn func(*types.Repository) error,
) (*types.Repository, error) {
	cp := *r
	if err := fn(&cp); err != nil {
		return nil, err
	}
	s.updated = &cp
	return &cp, nil
}

type sv5ActivityStore struct {
	store.PullReqActivityStore

	err      error
	payloads []types.PullReqActivityPayload
}

func (s *sv5ActivityStore) CreateWithPayload(
	_ context.Context, _ *types.PullReq, _ int64, p types.PullReqActivityPayload, _ *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	s.payloads = append(s.payloads, p)
	return &types.PullReqActivity{}, s.err
}

type sv5RepoIDCache struct{ store.RepoIDCache }

func (sv5RepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if id == 404 {
		return nil, errSv5Boom
	}
	if id == 405 {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return &types.RepositoryCore{ID: id, GitUID: "git-uid", ParentID: 3}, nil
}

type sv5Git struct {
	git.Interface

	out    git.MergeOutput
	err    error
	params *git.MergeParams
}

func (g *sv5Git) Merge(_ context.Context, p *git.MergeParams) (git.MergeOutput, error) {
	g.params = p
	return g.out, g.err
}

type sv5SSE struct {
	sse.Streamer

	published []enum.SSEType
}

func (s *sv5SSE) Publish(_ context.Context, _ int64, t enum.SSEType, _ any) {
	s.published = append(s.published, t)
}

type sv5URLProvider struct{ gitnessurl.Provider }

func (sv5URLProvider) GetInternalAPIURL(context.Context) string { return "http://internal.test" }

type sv5Producer struct {
	mu      sync.Mutex
	streams []string
}

func (p *sv5Producer) Send(_ context.Context, streamID string, _ map[string]any) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.streams = append(p.streams, streamID)
	return "1", nil
}

func (p *sv5Producer) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.streams)
}

type sv5Env struct {
	svc      *Service
	prs      *sv5PullReqStore
	repos    *sv5RepoStore
	activity *sv5ActivityStore
	git      *sv5Git
	bus      *sv5PubSub
	sse      *sv5SSE
	producer *sv5Producer
}

func sv5NewEnv(t *testing.T) *sv5Env {
	t.Helper()
	require.True(t, sv5Booted)

	srcRepoID := int64(10)
	env := &sv5Env{
		prs: &sv5PullReqStore{pr: &types.PullReq{
			ID: 7, Number: 3, TargetRepoID: 10, SourceRepoID: &srcRepoID, State: enum.PullReqStateOpen,
			SourceBranch: "feature", TargetBranch: "main", SourceSHA: sv5SHAa, ActivitySeq: 4,
		}},
		repos:    &sv5RepoStore{},
		activity: &sv5ActivityStore{},
		git: &sv5Git{out: git.MergeOutput{
			BaseSHA: sha.Must(sv5SHAb), HeadSHA: sha.Must(sv5SHAa), MergeSHA: sha.Must(sv5SHAc),
			MergeBaseSHA: sha.Must(sv5SHAb), CommitCount: 2, ChangedFileCount: 3, Additions: 4, Deletions: 5,
		}},
		bus:      &sv5PubSub{},
		sse:      &sv5SSE{},
		producer: &sv5Producer{},
	}

	system, err := events.NewSystem(
		func(string, string) (events.StreamConsumer, error) { return nil, errSv5Boom },
		env.producer,
		events.NewNoopCollector(),
	)
	require.NoError(t, err)
	reporter, err := pullreqevents.NewReporter(system)
	require.NoError(t, err)

	env.svc = &Service{
		pullreqEvReporter: reporter,
		git:               env.git,
		repoFinder: refcache.NewRepoFinder(
			nil, nil, sv5RepoIDCache{}, nil, cache.Evictor[*types.RepositoryCore]{}),
		repoStore:          env.repos,
		pullreqStore:       env.prs,
		activityStore:      env.activity,
		sseStreamer:        env.sse,
		urlProvider:        sv5URLProvider{},
		cancelMergeability: map[string]context.CancelFunc{},
		pubsub:             env.bus,
	}
	return env
}

func TestSv5UpdateMergeData(t *testing.T) {
	ctx := context.Background()

	t.Run("stores the merge outcome", func(t *testing.T) {
		env := sv5NewEnv(t)
		require.NoError(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9))

		require.NotNil(t, env.git.params)
		require.Equal(t, "main", env.git.params.BaseBranch)
		require.Equal(t, sha.Must(sv5SHAa), env.git.params.HeadSHA)
		require.True(t, env.git.params.Force)
		require.Len(t, env.git.params.Refs, 1)
		require.Equal(t, "refs/pullreq/3/merge", env.git.params.Refs[0].Name)

		pr := env.prs.updated
		require.NotNil(t, pr)
		require.Equal(t, sv5SHAb, pr.MergeBaseSHA)
		require.Equal(t, sv5SHAb, *pr.MergeTargetSHA)
		require.Equal(t, sv5SHAc, *pr.MergeSHA)
		require.EqualValues(t, 2, *pr.Stats.DiffStats.Commits)
		require.Equal(t, enum.MergeCheckStatusMergeable, pr.MergeCheckStatus)

		require.Equal(t, []string{cancelMergeCheckKey}, env.bus.topics)
		require.Equal(t, []string{sv5SHAb}, env.bus.data)
		require.Equal(t, []enum.SSEType{enum.SSETypePullReqUpdated}, env.sse.published)
		require.Equal(t, 1, env.producer.count(), "merge check succeeded event")
		require.Empty(t, env.svc.cancelMergeability, "the cancel func is removed afterwards")
	})

	t.Run("conflicting merge clears the merge commit", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.out.MergeSHA = sha.None
		env.git.out.ConflictFiles = []string{"a.txt"}
		require.NoError(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9))
		require.Nil(t, env.prs.updated.MergeSHA)
		require.Equal(t, enum.MergeCheckStatusConflict, env.prs.updated.MergeCheckStatus)
	})

	t.Run("pull request lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.findErr = errSv5Boom
		require.ErrorIs(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9), errSv5Boom)
	})

	t.Run("closed pull request", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.pr.State = enum.PullReqStateClosed
		require.ErrorContains(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9), "closed PR 3")
	})

	t.Run("cancel notification failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.bus.err = errSv5Boom
		require.ErrorIs(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9), errSv5Boom)
	})

	t.Run("same commit already being checked on this machine", func(t *testing.T) {
		env := sv5NewEnv(t)
		called := false
		env.svc.cancelMergeability[sv5SHAa] = func() { called = true }
		require.NoError(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9))
		require.Nil(t, env.git.params, "no merge is attempted")
		require.False(t, called, "the running check is left alone")
	})

	t.Run("target repository lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.pr.TargetRepoID = 404
		require.ErrorIs(t, env.svc.updateMergeData(ctx, 404, 3, sv5SHAb, sv5SHAa, 9), errSv5Boom)
	})

	t.Run("source branch moved on", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.err = gitnesserrors.PreconditionFailed("moved")
		err := env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9)
		require.ErrorContains(t, err, "discarding requested")
		require.ErrorContains(t, err, "is not on SHA")
	})

	t.Run("git failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.git.err = errSv5Boom
		err := env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9)
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "failed to run git merge")
	})

	t.Run("newer commit on the pull request is discarded", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.pr.SourceSHA = sv5SHAc
		err := env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9)
		require.ErrorContains(t, err, "discarding requested")
		require.ErrorContains(t, err, "is newer than")
	})

	t.Run("database failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.updateErr = errSv5Boom
		require.ErrorContains(t, env.svc.updateMergeData(ctx, 10, 3, sv5SHAb, sv5SHAa, 9), "update PR merge ref in db")
	})
}

func TestSv5MergeCheckEventHandlers(t *testing.T) {
	ctx := context.Background()
	base := pullreqevents.Base{PullReqID: 7, TargetRepoID: 10, Number: 3, PrincipalID: 9}

	tests := []struct {
		name    string
		run     func(s *Service) error
		oldSHA  string
		wantSHA string
	}{
		{"created", func(s *Service) error {
			return s.mergeCheckOnCreated(ctx, &events.Event[*pullreqevents.CreatedPayload]{
				Payload: &pullreqevents.CreatedPayload{Base: base, SourceSHA: sv5SHAa}})
		}, sha.Nil.String(), sv5SHAa},
		{"branch updated", func(s *Service) error {
			return s.mergeCheckOnBranchUpdate(ctx, &events.Event[*pullreqevents.BranchUpdatedPayload]{
				Payload: &pullreqevents.BranchUpdatedPayload{Base: base, OldSHA: sv5SHAb, NewSHA: sv5SHAa}})
		}, sv5SHAb, sv5SHAa},
		{"target branch changed", func(s *Service) error {
			return s.mergeCheckOnTargetBranchChange(ctx, &events.Event[*pullreqevents.TargetBranchChangedPayload]{
				Payload: &pullreqevents.TargetBranchChangedPayload{Base: base, SourceSHA: sv5SHAa}})
		}, "", sv5SHAa},
		{"reopened", func(s *Service) error {
			return s.mergeCheckOnReopen(ctx, &events.Event[*pullreqevents.ReopenedPayload]{
				Payload: &pullreqevents.ReopenedPayload{Base: base, SourceSHA: sv5SHAa}})
		}, "", sv5SHAa},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			env := sv5NewEnv(t)
			require.NoError(t, tc.run(env.svc))
			require.Equal(t, []string{tc.oldSHA}, env.bus.data)
			require.Equal(t, tc.wantSHA, env.git.params.HeadSHA.String())
		})
	}
}

func TestSv5CloseBecauseNonUniqueMergeBase(t *testing.T) {
	ctx := context.Background()
	target := sha.Must(sv5SHAb)
	source := sha.Must(sv5SHAc)

	t.Run("closes the pull request", func(t *testing.T) {
		env := sv5NewEnv(t)
		pr := env.prs.pr
		pr.IsDraft = true

		require.NoError(t, env.svc.CloseBecauseNonUniqueMergeBase(ctx, target, source, pr))

		closed := env.prs.closed
		require.NotNil(t, closed)
		require.Equal(t, enum.PullReqStateClosed, closed.State)
		require.Equal(t, enum.PullReqSubStateNone, closed.SubState)
		require.Equal(t, sv5SHAc, closed.SourceSHA)
		require.Equal(t, sv5SHAb, *closed.MergeTargetSHA)
		require.Nil(t, closed.MergeSHA)
		require.Equal(t, enum.MergeCheckStatusUnchecked, closed.MergeCheckStatus)
		require.Equal(t, int64(6), closed.ActivitySeq)

		require.Equal(t, 1, env.repos.updated.NumClosedPulls)
		require.Equal(t, 1, env.repos.updated.NumOpenPulls)

		require.Len(t, env.activity.payloads, 2)
		require.Equal(t,
			&types.PullRequestActivityPayloadNonUniqueMergeBase{TargetSHA: target, SourceSHA: source},
			env.activity.payloads[0])
		require.Equal(t, &types.PullRequestActivityPayloadStateChange{
			Old: enum.PullReqStateOpen, New: enum.PullReqStateClosed, OldDraft: true, NewDraft: true,
		}, env.activity.payloads[1])

		require.Equal(t, 1, env.producer.count(), "closed event")
		require.Equal(t, []enum.SSEType{enum.SSETypePullReqUpdated}, env.sse.published)
	})

	t.Run("activity failures are tolerated", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.activity.err = errSv5Boom
		require.NoError(t, env.svc.CloseBecauseNonUniqueMergeBase(ctx, target, source, env.prs.pr))
		require.Len(t, env.activity.payloads, 2)
	})

	t.Run("repository lookup failure", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.repos.err = errSv5Boom
		err := env.svc.CloseBecauseNonUniqueMergeBase(ctx, target, source, env.prs.pr)
		require.ErrorIs(t, err, errSv5Boom)
		require.ErrorContains(t, err, "find repository")
	})

	t.Run("pull request is not open anymore", func(t *testing.T) {
		env := sv5NewEnv(t)
		env.prs.pr.State = enum.PullReqStateMerged
		err := env.svc.CloseBecauseNonUniqueMergeBase(ctx, target, source, env.prs.pr)
		require.ErrorIs(t, err, ErrPullReqNotOpen)
		require.Nil(t, env.repos.updated)
	})
}
