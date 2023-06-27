// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

package queue

import (
	"context"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/drone/drone-go/drone"
	"github.com/drone/drone/core"
	"github.com/drone/drone/mock"

	"github.com/golang/mock/gomock"
)

func TestQueue(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	items := []*core.Stage{
		{ID: 3, OS: "linux", Arch: "amd64"},
		{ID: 2, OS: "linux", Arch: "amd64"},
		{ID: 1, OS: "linux", Arch: "amd64"},
	}

	ctx := context.Background()
	store := mock.NewMockStageStore(controller)
	store.EXPECT().ListIncomplete(ctx).Return(items, nil).Times(1)
	store.EXPECT().ListIncomplete(ctx).Return(items[1:], nil).Times(1)
	store.EXPECT().ListIncomplete(ctx).Return(items[2:], nil).Times(1)

	q := newQueue(ctx, store)
	for _, item := range items {
		next, err := q.Request(ctx, core.Filter{OS: "linux", Arch: "amd64"})
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := next, item; got != want {
			t.Errorf("Want build %d, got %d", want.ID, got.ID)
		}
	}
}

func TestQueueCancel(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	ctx, cancel := context.WithCancel(context.Background())
	store := mock.NewMockStageStore(controller)
	store.EXPECT().ListIncomplete(ctx).Return(nil, nil)

	q := newQueue(ctx, store)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		build, err := q.Request(ctx, core.Filter{OS: "linux/amd64", Arch: "amd64"})
		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got %s", err)
		}
		if build != nil {
			t.Errorf("Expect nil build when subscribe canceled")
		}
		wg.Done()
	}()
	<-time.After(10 * time.Millisecond)

	q.Lock()
	count := len(q.workers)
	q.Unlock()

	if got, want := count, 1; got != want {
		t.Errorf("Want %d listener, got %d", want, got)
	}

	cancel()
	wg.Wait()
}

func TestQueuePush(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	item1 := &core.Stage{
		ID:   1,
		OS:   "linux",
		Arch: "amd64",
	}
	item2 := &core.Stage{
		ID:   2,
		OS:   "linux",
		Arch: "amd64",
	}

	ctx := context.Background()
	store := mock.NewMockStageStore(controller)

	q := &queue{
		store: store,
		ready: make(chan struct{}, 1),
	}
	q.Schedule(ctx, item1)
	q.Schedule(ctx, item2)
	select {
	case <-q.ready:
	case <-time.After(time.Millisecond):
		t.Errorf("Expect queue signaled on push")
	}
}

func TestMatchResource(t *testing.T) {
	tests := []struct {
		kinda, typea, kindb, typeb string
		want                       bool
	}{
		// unspecified in yaml, unspecified by agent
		{"", "", "", "", true},

		// unspecified in yaml, specified by agent
		{"pipeline", "docker", "", "", true},
		{"pipeline", "", "", "", true},
		{"", "docker", "", "", true},

		// specified in yaml, unspecified by agent
		{"", "", "pipeline", "docker", true},
		{"", "", "pipeline", "", true},
		{"", "", "", "docker", true},

		// specified in yaml, specified by agent
		{"pipeline", "docker", "pipeline", "docker", true},
		{"pipeline", "exec", "pipeline", "docker", false},
		{"approval", "slack", "pipeline", "docker", false},

		// misc
		{"", "docker", "pipeline", "docker", true},
		{"pipeline", "", "pipeline", "docker", true},
		{"pipeline", "docker", "", "docker", true},
		{"pipeline", "docker", "pipeline", "", true},
	}

	for i, test := range tests {
		got, want := matchResource(test.kinda, test.typea, test.kindb, test.typeb), test.want
		if got != want {
			t.Errorf("Unexpected results at index %d", i)
		}
	}
}

func TestShouldThrottle(t *testing.T) {
	tests := []struct {
		ID     int64
		RepoID int64
		Status string
		Limit  int
		Want   bool
	}{
		// repo 1: 2 running, 1 pending
		{Want: false, ID: 1, RepoID: 1, Status: drone.StatusRunning, Limit: 2},
		{Want: false, ID: 2, RepoID: 1, Status: drone.StatusRunning, Limit: 2},
		{Want: true, ID: 3, RepoID: 1, Status: drone.StatusPending, Limit: 2},

		// repo 2: 1 running, 1 pending
		{Want: false, ID: 4, RepoID: 2, Status: drone.StatusRunning, Limit: 2},
		{Want: false, ID: 5, RepoID: 2, Status: drone.StatusPending, Limit: 2},

		// repo 3: 3 running, 1 pending
		{Want: false, ID: 6, RepoID: 3, Status: drone.StatusRunning, Limit: 2},
		{Want: false, ID: 7, RepoID: 3, Status: drone.StatusRunning, Limit: 2},
		{Want: false, ID: 8, RepoID: 3, Status: drone.StatusRunning, Limit: 2},
		{Want: true, ID: 9, RepoID: 3, Status: drone.StatusPending, Limit: 2},

		// repo 4: 2 running, 1 pending, no limit
		{Want: false, ID: 10, RepoID: 4, Status: drone.StatusRunning, Limit: 0},
		{Want: false, ID: 11, RepoID: 4, Status: drone.StatusRunning, Limit: 0},
		{Want: false, ID: 12, RepoID: 4, Status: drone.StatusPending, Limit: 0},
	}
	var stages []*core.Stage
	for _, test := range tests {
		stages = append(stages, &core.Stage{
			ID:        test.ID,
			RepoID:    test.RepoID,
			Status:    test.Status,
			LimitRepo: test.Limit,
		})
	}
	for i, test := range tests {
		stage := stages[i]
		if got, want := shouldThrottle(stage, stages, stage.LimitRepo), test.Want; got != want {
			t.Errorf("Unexpected results at index %d", i)
		}
	}
}

func TestWithinLimits(t *testing.T) {
	tests := []struct {
		result bool
		stage  *core.Stage
		stages []*core.Stage
	}{
		// multiple stages executing for same repository and with same
		// name, but no concurrency limits exist. expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 3, RepoID: 1, Name: "build", Limit: 0,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "running"},
				{ID: 2, RepoID: 1, Name: "build", Status: "running"},
				{ID: 3, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 1, no existing stages
		// exist for same repository id. expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 3, RepoID: 2, Name: "build", Limit: 0,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "running"},
				{ID: 2, RepoID: 1, Name: "build", Status: "running"},
				{ID: 3, RepoID: 2, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 1, no existing stages
		// exist for same stage name. expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 3, RepoID: 1, Name: "build", Limit: 0,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "test", Status: "running"},
				{ID: 2, RepoID: 1, Name: "test", Status: "running"},
				{ID: 3, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// single stage with concurrency 1, no existing stages
		// exist. expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 1, RepoID: 1, Name: "build", Limit: 1,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 1, other named stages
		// exist in the queue, but they come after this stage.
		// expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 1, RepoID: 1, Name: "build", Limit: 1,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "pending"},
				{ID: 2, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 1, however, stage with same
		// repository and name is already executing. expect false.
		{
			result: false,
			stage: &core.Stage{
				ID: 2, RepoID: 1, Name: "build", Limit: 1,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "running"},
				{ID: 2, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 2. one existing stage in the
		// queue before this stage. expect true.
		{
			result: true,
			stage: &core.Stage{
				ID: 2, RepoID: 1, Name: "build", Limit: 2,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "running"},
				{ID: 2, RepoID: 1, Name: "build", Status: "pending"},
				{ID: 3, RepoID: 1, Name: "build", Status: "pending"},
			},
		},

		// stage with concurrency 1. stages start out of order, and the
		// second named stage starts before its predecessor.  Its predecessor
		// should not execute. expect false.
		{
			result: false,
			stage: &core.Stage{
				ID: 1, RepoID: 1, Name: "build", Limit: 1,
			},
			stages: []*core.Stage{
				{ID: 1, RepoID: 1, Name: "build", Status: "pending"},
				{ID: 2, RepoID: 1, Name: "build", Status: "running"},
			},
		},
	}

	for i, test := range tests {
		if got, want := withinLimits(test.stage, test.stages), test.result; got != want {
			t.Errorf("Unexpected results at index %d", i)
		}
	}
}

func TestWithinLimits_Old(t *testing.T) {
	tests := []struct {
		ID     int64
		RepoID int64
		Name   string
		Limit  int
		Want   bool
	}{
		{Want: true, ID: 1, RepoID: 1, Name: "foo"},
		{Want: true, ID: 2, RepoID: 2, Name: "bar", Limit: 1},
		{Want: true, ID: 3, RepoID: 1, Name: "bar", Limit: 1},
		{Want: false, ID: 4, RepoID: 1, Name: "bar", Limit: 1},
		{Want: false, ID: 5, RepoID: 1, Name: "bar", Limit: 1},
		{Want: true, ID: 6, RepoID: 1, Name: "baz", Limit: 2},
		{Want: true, ID: 7, RepoID: 1, Name: "baz", Limit: 2},
		{Want: false, ID: 8, RepoID: 1, Name: "baz", Limit: 2},
		{Want: false, ID: 9, RepoID: 1, Name: "baz", Limit: 2},
		{Want: true, ID: 10, RepoID: 1, Name: "baz", Limit: 0},
	}
	var stages []*core.Stage
	for _, test := range tests {
		stages = append(stages, &core.Stage{
			ID:     test.ID,
			RepoID: test.RepoID,
			Name:   test.Name,
			Limit:  test.Limit,
		})
	}
	for i, test := range tests {
		stage := stages[i]
		if got, want := withinLimits(stage, stages), test.Want; got != want {
			t.Errorf("Unexpected results at index %d", i)
		}
	}
}

func incomplete(n int) ([]*core.Stage, error) {
	ret := make([]*core.Stage, n)
	for i := range ret {
		ret[i] = &core.Stage{
			OS:   "linux/amd64",
			Arch: "amd64",
		}
	}
	return ret, nil
}

func TestQueueDeadlock(t *testing.T) {
	controller := gomock.NewController(t)
	defer controller.Finish()

	n := 10
	donechan := make(chan struct{}, n)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	store := mock.NewMockStageStore(controller)
	store.EXPECT().ListIncomplete(ctx).Return(incomplete(n)).AnyTimes()

	q := newQueue(ctx, store)
	doWork := func(i int) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		ctx, cancel := context.WithTimeout(ctx,
			time.Duration(i+rand.Intn(1000/n))*time.Millisecond)
		defer cancel()
		if i%3 == 0 {
			// Randomly cancel some contexts to simulate timeouts
			cancel()
		}
		_, err := q.Request(ctx, core.Filter{OS: "linux/amd64", Arch: "amd64"})
		if err != nil && err != context.Canceled && err !=
			context.DeadlineExceeded {
			t.Errorf("Expected context.Canceled or context.DeadlineExceeded error, got %s", err)
		}
		select {
		case donechan <- struct{}{}:
		case <-ctx.Done():
		}
		return true
	}
	for i := 0; i < n; i++ {
		go func(i int) {
			// Spawn n workers, doing work until the parent context is canceled
			for doWork(i) {
			}
		}(i)
	}
	// Wait for n * 10 tasks to complete, then exit and cancel all the workers.
	for seen := 0; seen < n*10; seen++ {
		<-donechan
	}
}
