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

package quotaresume

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/insights/triage"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeStore struct {
	tasks []*types.Task
	limit int
}

func (f *fakeStore) ListWaitingQuota(_ context.Context, limit int) ([]*types.Task, error) {
	f.limit = limit
	return f.tasks[:min(limit, len(f.tasks))], nil
}

type fakeTasks struct {
	resumed []int64
	back    []int64
}

func (f *fakeTasks) Resume(_ context.Context, _, id int64, _ string) (*types.Task, error) {
	f.resumed = append(f.resumed, id)
	return nil, nil
}

func (f *fakeTasks) Transition(_ context.Context, _, id int64, to enum.TaskState, _ string) (*types.Task, error) {
	if to != enum.TaskStateWaitingQuota {
		return nil, errors.New("unexpected state")
	}
	f.back = append(f.back, id)
	return nil, nil
}

type fakeQuota struct{ n int }

func (q *fakeQuota) Remaining(aigateway.Class) int { return q.n }

type fakeLinks struct{ links []*types.TaskLink }

func (f fakeLinks) ListByTask(context.Context, int64) ([]*types.TaskLink, error) { return f.links, nil }

type fakeRetriager struct {
	issues []int64
	err    error
}

func (f *fakeRetriager) Retriage(_ context.Context, id int64) error {
	f.issues = append(f.issues, id)
	return f.err
}

func newMutexes() lock.MutexManager {
	return lock.NewInMemory(lock.Config{
		App: "t", Expiry: time.Minute, Tries: 1, RetryDelay: time.Millisecond,
	})
}

func tk(id int64, origin enum.TaskOrigin) *types.Task {
	return &types.Task{ID: id, Origin: origin, State: enum.TaskStateWaitingQuota}
}

func TestRunKeepsStoreOrderAndLimitsByQuota(t *testing.T) {
	st := &fakeStore{tasks: []*types.Task{tk(3, "manual"), tk(1, "manual"), tk(2, "manual")}}
	ts := &fakeTasks{}
	s := newSvc(st, ts, &fakeQuota{n: 2}, newMutexes(), nil)

	res, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if st.limit != 2 || !slices.Equal(ts.resumed, []int64{3, 1}) || res.Resumed != 2 {
		t.Fatalf("limit=%d resumed=%v res=%v", st.limit, ts.resumed, res)
	}
}

func TestRunNoQuotaDoesNothing(t *testing.T) {
	st := &fakeStore{tasks: []*types.Task{tk(1, "manual")}}
	ts := &fakeTasks{}
	res, err := newSvc(st, ts, &fakeQuota{}, newMutexes(), nil).Run(context.Background())
	if err != nil || !res.Skipped || len(ts.resumed) != 0 {
		t.Fatalf("res=%v err=%v resumed=%v", res, err, ts.resumed)
	}
}

func TestRunDispatchesByOrigin(t *testing.T) {
	var calls []string
	mk := func(name string) Resumer {
		return ResumerFunc(func(_ context.Context, t *types.Task) error {
			calls = append(calls, fmt.Sprintf("%s:%d", name, t.ID))
			return nil
		})
	}
	st := &fakeStore{tasks: []*types.Task{tk(1, enum.TaskOriginInsights), tk(2, enum.TaskOriginShield), tk(3, enum.TaskOriginManual)}}
	ts := &fakeTasks{}
	s := newSvc(st, ts, &fakeQuota{n: 10}, newMutexes(), map[enum.TaskOrigin]Resumer{
		enum.TaskOriginInsights: mk("insights"), enum.TaskOriginShield: mk("shield"),
	})
	res, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(calls, []string{"insights:1", "shield:2"}) || !slices.Equal(ts.resumed, []int64{1, 2, 3}) || res.Resumed != 3 {
		t.Fatalf("calls=%v resumed=%v res=%v", calls, ts.resumed, res)
	}
}

func TestRunStopsWhenResumerHitsQuota(t *testing.T) {
	var ran []int64
	r := ResumerFunc(func(_ context.Context, t *types.Task) error {
		ran = append(ran, t.ID)
		if t.ID == 2 {
			return fmt.Errorf("wrapped: %w", triage.ErrQuotaExhausted)
		}
		return nil
	})
	st := &fakeStore{tasks: []*types.Task{tk(1, "insights"), tk(2, "insights"), tk(3, "insights")}}
	ts := &fakeTasks{}
	s := newSvc(st, ts, &fakeQuota{n: 10}, newMutexes(), map[enum.TaskOrigin]Resumer{"insights": r})

	res, err := s.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !res.Stopped || !slices.Equal(ran, []int64{1, 2}) || !slices.Equal(ts.back, []int64{2}) {
		t.Fatalf("ran=%v back=%v res=%v", ran, ts.back, res)
	}
}

func TestRunOtherErrorContinues(t *testing.T) {
	r := ResumerFunc(func(_ context.Context, t *types.Task) error {
		if t.ID == 1 {
			return errors.New("boom")
		}
		return nil
	})
	st := &fakeStore{tasks: []*types.Task{tk(1, "insights"), tk(2, "insights")}}
	ts := &fakeTasks{}
	res, _ := newSvc(st, ts, &fakeQuota{n: 10}, newMutexes(), map[enum.TaskOrigin]Resumer{"insights": r}).
		Run(context.Background())
	if res.Failed != 1 || res.Resumed != 2 || len(ts.back) != 0 {
		t.Fatalf("res=%v back=%v", res, ts.back)
	}
}

func TestRunSkipsWhenLockHeld(t *testing.T) {
	mutexes := newMutexes()
	mx, err := mutexes.NewMutex(lockKey)
	if err != nil {
		t.Fatal(err)
	}
	if err = mx.Lock(context.Background()); err != nil {
		t.Fatal(err)
	}
	st := &fakeStore{tasks: []*types.Task{tk(1, "manual")}}
	ts := &fakeTasks{}
	res, err := newSvc(st, ts, &fakeQuota{n: 10}, mutexes, nil).Run(context.Background())
	if err != nil || !res.Skipped || len(ts.resumed) != 0 {
		t.Fatalf("res=%v err=%v resumed=%v", res, err, ts.resumed)
	}
	_ = mx.Unlock(context.Background())
	res, _ = newSvc(st, ts, &fakeQuota{n: 10}, mutexes, nil).Run(context.Background())
	if res.Skipped || res.Resumed != 1 {
		t.Fatalf("lock not released: %v", res)
	}
}

func TestInsightsResumer(t *testing.T) {
	ret := &fakeRetriager{}
	links := fakeLinks{links: []*types.TaskLink{
		{Kind: enum.TaskLinkKindPullReq, RefID: "9"}, {Kind: enum.TaskLinkKindInsightsIssue, RefID: "42"},
	}}
	if err := InsightsResumer(links, ret).Resume(context.Background(), tk(1, "insights")); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(ret.issues, []int64{42}) {
		t.Fatalf("issues=%v", ret.issues)
	}
	if err := InsightsResumer(fakeLinks{}, ret).Resume(context.Background(), tk(1, "insights")); err == nil {
		t.Fatal("expected error without link")
	}
	ret.err = triage.ErrQuotaExhausted
	err := InsightsResumer(links, ret).Resume(context.Background(), tk(1, "insights"))
	if !IsQuotaError(err) {
		t.Fatalf("quota error not detected: %v", err)
	}
}

func newSvc(st TaskStore, ts TaskService, q Quota, m lock.MutexManager, r map[enum.TaskOrigin]Resumer) *Service {
	s := New(st, ts, q, m, r)
	s.principal = func() int64 { return 1 }
	return s
}
