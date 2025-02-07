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

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

func TestMediator_basic(t *testing.T) {
	space := &types.Space{
		ID:         1,
		Identifier: "space",
	}
	spaceMock := &SpaceStoreMock{
		FindByRefFn: func(context.Context, string) (*types.Space, error) {
			return space, nil
		},
		FindByIDsFn: func(context.Context, ...int64) ([]*types.Space, error) {
			return []*types.Space{space}, nil
		},
	}

	out := atomic.Int64{}
	in := atomic.Int64{}
	counter := atomic.Int64{}

	usageMock := &MetricsMock{
		UpsertOptimisticFn: func(_ context.Context, metric *types.UsageMetric) error {
			if metric.RootSpaceID != space.ID {
				return fmt.Errorf("expected root space id to be %d, got %d", space.ID, metric.RootSpaceID)
			}
			out.Add(metric.Bandwidth)
			in.Add(metric.Storage)
			counter.Add(1)
			return nil
		},
		GetMetricsFn: func(
			context.Context,
			int64, // spaceID
			int64, // startDate
			int64, // endDate
		) (*types.UsageMetric, error) {
			return &types.UsageMetric{
				Bandwidth: out.Load(),
				Storage:   in.Load(),
			}, nil
		},
		ListFn: func(context.Context, int64, int64) ([]types.UsageMetric, error) {
			return []types.UsageMetric{}, nil
		},
	}

	numRoutines := 10
	defaultSize := 512
	mediator := NewMediator(
		context.Background(),
		spaceMock,
		usageMock,
		Config{
			MaxWorkers: 5,
		},
	)
	wg := sync.WaitGroup{}
	for range numRoutines {
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
	mediator.Wait()

	require.Equal(t, int64(numRoutines*defaultSize), out.Load())
	require.Equal(t, int64(numRoutines*defaultSize), in.Load())
}
