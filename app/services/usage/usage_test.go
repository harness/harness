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

package usage

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	repoevents "github.com/harness/gitness/app/events/repo"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/assert"
)

func TestMediator_basic(t *testing.T) {
	t.Parallel()

	MinFlushInterval = 500 * time.Millisecond
	space := &types.SpaceCore{
		//ID:         1,
		Identifier: "space",
	}
	spaceFinderMock := &SpaceFinderMock{
		FindByRefFn: func(context.Context, string) (*types.SpaceCore, error) {
			return space, nil
		},
	}
	repo := &types.RepositoryCore{
		ID:   2,
		Path: "space/repo",
	}
	repoFinderMock := &RepoFinderMock{
		FindByIDFn: func(_ context.Context, id int64) (*types.RepositoryCore, error) {
			if id != repo.ID {
				return nil, fmt.Errorf("expected id to be %d, got %d", repo.ID, id)
			}
			return repo, nil
		},
	}

	eventSystem, err := events.ProvideSystem(events.Config{
		Mode:            events.ModeInMemory,
		MaxStreamLength: 100,
	}, nil)
	if err != nil {
		t.Fatalf("failed to create event system: %v", err)
	}
	repoEvReaderFactory, err := repoevents.NewReaderFactory(eventSystem)
	if err != nil {
		t.Fatalf("failed to create repo event reader factory: %v", err)
	}
	repoEvReporter, err := repoevents.NewReporter(eventSystem)
	if err != nil {
		t.Fatalf("failed to create repo event reporter: %v", err)
	}

	out := atomic.Int64{}
	in := atomic.Int64{}
	pushes := atomic.Int64{}

	usageMock := &MetricsMock{
		UpsertFn: func(_ context.Context, metrics []*types.UsageMetric) error {
			for _, metric := range metrics {
				if metric.RootSpaceID != space.ID {
					return fmt.Errorf("expected root space id to be %d, got %d", space.ID, metric.RootSpaceID)
				}
				out.Add(metric.BandwidthOut)
				in.Add(metric.BandwidthIn)
				pushes.Add(metric.Pushes)
			}
			return nil
		},
		GetMetricsFn: func(
			context.Context,
			int64, // spaceID
			int64, // startDate
			int64, // endDate
		) (*types.UsageMetric, error) {
			return &types.UsageMetric{
				BandwidthOut: out.Load(),
				BandwidthIn:  in.Load(),
			}, nil
		},
		ListFn: func(context.Context, int64, int64) ([]types.UsageMetric, error) {
			return []types.UsageMetric{}, nil
		},
	}

	numBandwidthRoutines := 10
	numEventsCreated := 4
	numEventsPushed := 5
	defaultSize := 512
	mediator := NewMediator(
		context.Background(),
		spaceFinderMock,
		usageMock,
		Config{},
	)
	err = RegisterEventListeners(context.Background(), "test", mediator, repoEvReaderFactory, repoFinderMock)
	if err != nil {
		t.Fatalf("failed to register event listeners: %v", err)
	}

	wg := sync.WaitGroup{}
	for range numBandwidthRoutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = mediator.Send(context.Background(), Metric{
				SpaceRef: space.Identifier,
				Bandwidth: Bandwidth{
					Out: int64(defaultSize),
					In:  int64(defaultSize),
				},
			})
		}()
	}

	wg.Wait()

	for range numEventsCreated {
		repoEvReporter.Created(context.Background(), &repoevents.CreatedPayload{
			Base: repoevents.Base{
				RepoID: repo.ID,
			},
		})
	}

	for range numEventsPushed {
		repoEvReporter.Pushed(context.Background(), &repoevents.PushedPayload{
			Base: repoevents.Base{
				RepoID: repo.ID,
			},
		})
	}

	time.Sleep(mediator.config.FlushInterval + 1*time.Second)

	assert.Equal(t, int64(numBandwidthRoutines*defaultSize), out.Load())
	assert.Equal(t, int64(numBandwidthRoutines*defaultSize), in.Load())
	assert.Equal(t, int64(numEventsCreated+numEventsPushed), pushes.Load())
}

func BenchmarkMediator_MillionRequests(b *testing.B) {
	MinFlushInterval = 500 * time.Millisecond
	space := &types.SpaceCore{
		ID:         1,
		Identifier: "space",
	}
	spaceFinderMock := &SpaceFinderMock{
		FindByRefFn: func(context.Context, string) (*types.SpaceCore, error) {
			return space, nil
		},
	}

	usageMock := &MetricsMock{
		UpsertFn: func(_ context.Context, _ []*types.UsageMetric) error {
			time.Sleep(50 * time.Millisecond)
			return nil
		},
		GetMetricsFn: func(context.Context, int64, int64, int64) (*types.UsageMetric, error) {
			return &types.UsageMetric{}, nil
		},
		ListFn: func(context.Context, int64, int64) ([]types.UsageMetric, error) {
			return []types.UsageMetric{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mediator := NewMediator(
		ctx,
		spaceFinderMock,
		usageMock,
		Config{},
	)

	numRequests := 1_000_000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for j := 0; j < numRequests; j++ {
			go func() {
				defer wg.Done()
				_ = mediator.Send(ctx, Metric{
					SpaceRef: space.Identifier,
					Bandwidth: Bandwidth{
						Out: 512,
						In:  512,
					},
				})
			}()
		}

		wg.Wait()
	}
	b.StopTimer()
}

func BenchmarkMediator_MillionRequests_MultiSpace(b *testing.B) {
	MinFlushInterval = 500 * time.Millisecond
	spaces := make(map[string]*types.SpaceCore)
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("space-%d", i)
		spaces[id] = &types.SpaceCore{
			ID:         int64(i),
			Identifier: id,
		}
	}

	spaceFinderMock := &SpaceFinderMock{
		FindByRefFn: func(_ context.Context, ref string) (*types.SpaceCore, error) {
			return spaces[ref], nil
		},
	}

	usageMock := &MetricsMock{
		UpsertFn: func(_ context.Context, _ []*types.UsageMetric) error {
			time.Sleep(500 * time.Millisecond)
			return nil
		},
		GetMetricsFn: func(context.Context, int64, int64, int64) (*types.UsageMetric, error) {
			return &types.UsageMetric{}, nil
		},
		ListFn: func(context.Context, int64, int64) ([]types.UsageMetric, error) {
			return []types.UsageMetric{}, nil
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mediator := NewMediator(
		ctx,
		spaceFinderMock,
		usageMock,
		Config{},
	)

	numRequests := 1_000_000
	spaceIDs := make([]string, 0, len(spaces))
	for id := range spaces {
		spaceIDs = append(spaceIDs, id)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		wg.Add(numRequests)

		for j := 0; j < numRequests; j++ {
			spaceRef := spaceIDs[j%len(spaceIDs)]
			go func(ref string) {
				defer wg.Done()
				_ = mediator.Send(ctx, Metric{
					SpaceRef: ref,
					Bandwidth: Bandwidth{
						Out: 512,
						In:  512,
					},
				})
			}(spaceRef)
		}

		wg.Wait()
	}
	b.StopTimer()
}
