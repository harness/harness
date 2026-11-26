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

package scheduler

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type infStageStore struct {
	store.StageStore

	stages []*types.Stage
	err    error
	calls  atomic.Int32
}

func (s *infStageStore) ListIncomplete(context.Context) ([]*types.Stage, error) {
	s.calls.Add(1)
	return s.stages, s.err
}

func infLockManager() lock.MutexManager {
	return lock.NewInMemory(lock.Config{
		App:        "inf",
		Namespace:  "sched",
		Expiry:     5 * time.Second,
		Tries:      3,
		RetryDelay: 10 * time.Millisecond,
	})
}

// infNewQueue builds a queue without launching its background loop.
func infNewQueue(t *testing.T, st store.StageStore) *queue {
	t.Helper()
	mx, err := infLockManager().NewMutex("build_queue")
	require.NoError(t, err)
	return &queue{
		store:    st,
		globMx:   mx,
		ready:    make(chan struct{}, 1),
		workers:  map[*worker]struct{}{},
		interval: time.Minute,
		ctx:      context.Background(),
	}
}

type infResult struct {
	stage *types.Stage
	err   error
}

func infRequest(ctx context.Context, q *queue, f Filter) <-chan infResult {
	ch := make(chan infResult, 1)
	go func() {
		s, err := q.Request(ctx, f)
		ch <- infResult{stage: s, err: err}
	}()
	return ch
}

func infWaitWorkers(t *testing.T, q *queue, n int) {
	t.Helper()
	require.Eventually(t, func() bool {
		q.Lock()
		defer q.Unlock()
		return len(q.workers) == n
	}, 5*time.Second, time.Millisecond)
}

func TestInfQueue_SignalMatchesWorkersToStages(t *testing.T) {
	tests := []struct {
		name    string
		filter  Filter
		stages  []*types.Stage
		wantID  int64 // 0 means no stage is expected to be handed out.
		wantMsg string
	}{
		{
			name:   "default kind and type",
			filter: Filter{},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending}},
			wantID: 1,
		},
		{
			name:   "running stage is skipped",
			filter: Filter{},
			stages: []*types.Stage{
				{ID: 1, Status: enum.CIStatusRunning},
				{ID: 2, Status: enum.CIStatusPending},
			},
			wantID: 2,
		},
		{
			name:   "stage already assigned to a machine is skipped",
			filter: Filter{},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, Machine: "m1"}},
		},
		{
			name:   "kind mismatch",
			filter: Filter{Kind: "pipeline", Type: "docker"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, Kind: "other"}},
		},
		{
			name:   "platform match",
			filter: Filter{OS: "linux", Arch: "amd64"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "linux", Arch: "amd64"}},
			wantID: 1,
		},
		{
			name:   "os mismatch",
			filter: Filter{OS: "linux", Arch: "amd64"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "windows", Arch: "amd64"}},
		},
		{
			name:   "arch mismatch",
			filter: Filter{OS: "linux", Arch: "amd64"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "linux", Arch: "arm64"}},
		},
		{
			name:   "variant mismatch",
			filter: Filter{OS: "linux", Arch: "arm", Variant: "v7"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "linux", Arch: "arm", Variant: "v6"}},
		},
		{
			name:   "variant match",
			filter: Filter{OS: "linux", Arch: "arm", Variant: "v7"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "linux", Arch: "arm", Variant: "v7"}},
			wantID: 1,
		},
		{
			name:   "kernel mismatch",
			filter: Filter{OS: "windows", Arch: "amd64", Kernel: "1709"},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, OS: "windows", Arch: "amd64", Kernel: "1803"}},
		},
		{
			name:   "labels match",
			filter: Filter{Labels: map[string]string{"gpu": "yes"}},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, Labels: map[string]string{"gpu": "yes"}}},
			wantID: 1,
		},
		{
			name:   "labels mismatch",
			filter: Filter{Labels: map[string]string{"gpu": "yes"}},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, Labels: map[string]string{"gpu": "no"}}},
		},
		{
			name:   "stage requires labels the worker lacks",
			filter: Filter{},
			stages: []*types.Stage{{ID: 1, Status: enum.CIStatusPending, Labels: map[string]string{"gpu": "yes"}}},
		},
		{
			name:   "concurrency limit blocks the stage",
			filter: Filter{},
			stages: []*types.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: enum.CIStatusPending},
				{ID: 2, RepoID: 1, Name: "build", Status: enum.CIStatusPending, Limit: 1},
			},
			wantID: 1,
		},
		{
			name:   "repo throttle blocks later stages",
			filter: Filter{},
			stages: []*types.Stage{
				{ID: 2, RepoID: 1, Name: "b", Status: enum.CIStatusPending, LimitRepo: 1},
				{ID: 1, RepoID: 1, Name: "a", Status: enum.CIStatusRunning},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := &infStageStore{stages: tt.stages}
			q := infNewQueue(t, st)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			res := infRequest(ctx, q, tt.filter)
			infWaitWorkers(t, q, 1)
			require.NoError(t, q.signal(context.Background()))

			if tt.wantID == 0 {
				select {
				case r := <-res:
					t.Fatalf("unexpected stage %+v", r)
				case <-time.After(30 * time.Millisecond):
				}
				cancel()
				r := <-res
				require.ErrorIs(t, r.err, context.Canceled)
				return
			}
			select {
			case r := <-res:
				require.NoError(t, r.err)
				require.Equal(t, tt.wantID, r.stage.ID)
			case <-time.After(5 * time.Second):
				t.Fatal("no stage handed out")
			}
			q.Lock()
			require.Empty(t, q.workers, "served worker must be removed")
			q.Unlock()
		})
	}
}

func TestInfQueue_SignalEarlyExits(t *testing.T) {
	ctx := context.Background()

	t.Run("no workers does not hit the store", func(t *testing.T) {
		st := &infStageStore{}
		q := infNewQueue(t, st)
		require.NoError(t, q.signal(ctx))
		require.Zero(t, st.calls.Load())
	})
	t.Run("paused queue", func(t *testing.T) {
		st := &infStageStore{}
		q := infNewQueue(t, st)
		q.workers[&worker{channel: make(chan *types.Stage)}] = struct{}{}
		require.NoError(t, q.Pause(ctx))
		require.NoError(t, q.signal(ctx))
		require.Zero(t, st.calls.Load())
	})
	t.Run("store error", func(t *testing.T) {
		st := &infStageStore{err: errors.New("db down")}
		q := infNewQueue(t, st)
		q.workers[&worker{channel: make(chan *types.Stage)}] = struct{}{}
		require.ErrorContains(t, q.signal(ctx), "db down")
	})
	t.Run("lock error", func(t *testing.T) {
		q := infNewQueue(t, &infStageStore{})
		q.globMx = infFailingMutex{}
		require.ErrorContains(t, q.signal(ctx), "lock failed")
	})
}

func TestInfQueue_ScheduleIsNonBlocking(t *testing.T) {
	q := infNewQueue(t, &infStageStore{})
	for i := 0; i < 5; i++ {
		require.NoError(t, q.Schedule(context.Background(), &types.Stage{}))
	}
	require.Len(t, q.ready, 1)
}

func TestInfQueue_RequestHonoursContext(t *testing.T) {
	q := infNewQueue(t, &infStageStore{})
	ctx, cancel := context.WithCancel(context.Background())
	res := infRequest(ctx, q, Filter{})
	infWaitWorkers(t, q, 1)
	cancel()
	r := <-res
	require.ErrorIs(t, r.err, context.Canceled)
	require.Nil(t, r.stage)
}

func TestInfQueue_StartLoopSignalsOnReady(t *testing.T) {
	st := &infStageStore{stages: []*types.Stage{{ID: 9, Status: enum.CIStatusPending}}}
	q := infNewQueue(t, st)
	ctx, cancel := context.WithCancel(context.Background())
	q.ctx = ctx

	done := make(chan error, 1)
	go func() { done <- q.start() }()

	res := infRequest(context.Background(), q, Filter{})
	select {
	case r := <-res:
		require.NoError(t, r.err)
		require.EqualValues(t, 9, r.stage.ID)
	case <-time.After(5 * time.Second):
		t.Fatal("start loop did not hand out the stage")
	}

	cancel()
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(5 * time.Second):
		t.Fatal("start did not stop")
	}
}

func TestInfQueue_StartLoopSignalsOnInterval(t *testing.T) {
	st := &infStageStore{err: errors.New("db down")}
	q := infNewQueue(t, st)
	q.interval = 5 * time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	q.ctx = ctx
	q.workers[&worker{channel: make(chan *types.Stage)}] = struct{}{}

	done := make(chan error, 1)
	go func() { done <- q.start() }()
	require.Eventually(t, func() bool { return st.calls.Load() >= 2 }, 5*time.Second, time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
}

func TestInfNewQueueAndScheduler(t *testing.T) {
	s, err := newScheduler(&infStageStore{}, infLockManager())
	require.NoError(t, err)
	require.NotNil(t, s)
	require.NoError(t, s.Schedule(context.Background(), &types.Stage{}))

	_, err = newQueue(&infStageStore{}, infFailingLockManager{})
	require.Error(t, err)
}

type infFailingMutex struct{}

func (infFailingMutex) Key() string                  { return "k" }
func (infFailingMutex) Lock(context.Context) error   { return errors.New("lock failed") }
func (infFailingMutex) Unlock(context.Context) error { return nil }

type infFailingLockManager struct{}

func (infFailingLockManager) NewMutex(string, ...lock.Option) (lock.Mutex, error) {
	return nil, errors.New("no mutex")
}

func TestInfCheckLabels(t *testing.T) {
	require.True(t, checkLabels(nil, nil))
	require.True(t, checkLabels(map[string]string{"a": "1"}, map[string]string{"a": "1"}))
	require.False(t, checkLabels(map[string]string{"a": "1"}, nil))
	require.False(t, checkLabels(map[string]string{"a": "1"}, map[string]string{"a": "2"}))
	require.False(t, checkLabels(map[string]string{"a": "1"}, map[string]string{"b": "1"}))
}

func TestInfMatchResource(t *testing.T) {
	require.True(t, matchResource("", "", "pipeline", "docker"))
	require.True(t, matchResource("pipeline", "docker", "", ""))
	require.True(t, matchResource("k", "t", "k", "t"))
	require.False(t, matchResource("k", "t", "k", "other"))
	require.False(t, matchResource("k", "t", "other", "t"))
}

func TestInfWithinLimits(t *testing.T) {
	running := &types.Stage{ID: 5, RepoID: 1, Name: "build", Status: enum.CIStatusRunning}
	older := &types.Stage{ID: 1, RepoID: 1, Name: "build", Status: enum.CIStatusPending}
	otherRepo := &types.Stage{ID: 2, RepoID: 2, Name: "build", Status: enum.CIStatusRunning}
	otherName := &types.Stage{ID: 3, RepoID: 1, Name: "test", Status: enum.CIStatusRunning}
	siblings := []*types.Stage{running, older, otherRepo, otherName}

	require.True(t, withinLimits(&types.Stage{ID: 9, RepoID: 1, Name: "build"}, siblings), "no limit set")
	require.False(t, withinLimits(&types.Stage{ID: 9, RepoID: 1, Name: "build", Limit: 2}, siblings))
	require.True(t, withinLimits(&types.Stage{ID: 9, RepoID: 1, Name: "build", Limit: 3}, siblings))
	// the stage itself and unrelated stages are not counted.
	require.True(t, withinLimits(&types.Stage{ID: 1, RepoID: 1, Name: "build", Limit: 1},
		[]*types.Stage{older, otherRepo, otherName}))
}

func TestInfShouldThrottle(t *testing.T) {
	siblings := []*types.Stage{
		{ID: 1, RepoID: 1}, {ID: 2, RepoID: 1}, {ID: 3, RepoID: 2}, {ID: 10, RepoID: 1},
	}
	stage := &types.Stage{ID: 5, RepoID: 1, Status: enum.CIStatusPending}

	require.False(t, shouldThrottle(stage, siblings, 0), "no limit")
	require.False(t, shouldThrottle(&types.Stage{ID: 5, RepoID: 1, Status: enum.CIStatusRunning}, siblings, 1),
		"running stages are never throttled")
	require.True(t, shouldThrottle(stage, siblings, 2), "two older stages of the same repo")
	require.False(t, shouldThrottle(stage, siblings, 3))
}

func TestInfCanceler_CancelNotifiesSubscribers(t *testing.T) {
	c := newCanceler()
	ctx := context.Background()

	type result struct {
		cancelled bool
		err       error
	}
	res := make(chan result, 2)
	for _, id := range []int64{7, 8} {
		go func() {
			ok, err := c.Cancelled(ctx, id)
			res <- result{cancelled: ok, err: err}
		}()
	}
	require.Eventually(t, func() bool {
		c.Lock()
		defer c.Unlock()
		return len(c.subscribers) == 2
	}, 5*time.Second, time.Millisecond)

	require.NoError(t, c.Cancel(ctx, 7))
	select {
	case r := <-res:
		require.NoError(t, r.err)
		require.True(t, r.cancelled)
	case <-time.After(5 * time.Second):
		t.Fatal("subscriber was not notified")
	}

	// the other subscriber is still waiting, stop it through its context.
	c.Lock()
	require.Len(t, c.subscribers, 1)
	require.Contains(t, c.cancelled, int64(7))
	c.Unlock()
	require.NoError(t, c.Cancel(ctx, 8))
	require.True(t, (<-res).cancelled)
}

func TestInfCanceler_CancelledHonoursContext(t *testing.T) {
	c := newCanceler()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := c.Cancelled(ctx, 1)
		done <- err
	}()
	require.Eventually(t, func() bool {
		c.Lock()
		defer c.Unlock()
		return len(c.subscribers) == 1
	}, 5*time.Second, time.Millisecond)
	cancel()
	require.ErrorIs(t, <-done, context.Canceled)
	c.Lock()
	require.Empty(t, c.subscribers)
	c.Unlock()
}

func TestInfCanceler_CollectRemovesExpiredEntries(t *testing.T) {
	c := newCanceler()
	c.cancelled[1] = time.Now().Add(-time.Minute)
	c.cancelled[2] = time.Now().Add(time.Minute)
	c.collect()
	require.NotContains(t, c.cancelled, int64(1))
	require.Contains(t, c.cancelled, int64(2))
}
