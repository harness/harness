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

	"github.com/stretchr/testify/require"
)

func TestMediator_basic(t *testing.T) {
	space := &types.SpaceCore{
		ID:         1,
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
		UpsertOptimisticFn: func(_ context.Context, metric *types.UsageMetric) error {
			if metric.RootSpaceID != space.ID {
				return fmt.Errorf("expected root space id to be %d, got %d", space.ID, metric.RootSpaceID)
			}
			out.Add(metric.BandwidthOut)
			in.Add(metric.BandwidthIn)
			pushes.Add(metric.Pushes)
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
		Config{
			MaxWorkers: 5,
		},
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

	// todo: add ability to wait for event system to complete
	time.Sleep(200 * time.Millisecond)

	mediator.Wait()

	require.Equal(t, int64(numBandwidthRoutines*defaultSize), out.Load())
	require.Equal(t, int64(numBandwidthRoutines*defaultSize), in.Load())
	require.Equal(t, int64(numEventsCreated+numEventsPushed), pushes.Load())
}
