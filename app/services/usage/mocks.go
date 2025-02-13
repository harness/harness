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

	"github.com/harness/gitness/types"
)

const (
	sampleText   = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789 "
	sampleLength = len(sampleText)
	spaceRef     = "space1%2fspace2%2fspace3"
)

type mockInterface struct {
	SendFunc func(
		ctx context.Context,
		payload Metric,
	) error
}

func (i *mockInterface) Send(
	ctx context.Context,
	payload Metric,
) error {
	return i.SendFunc(ctx, payload)
}

type SpaceFinderMock struct {
	FindByRefFn func(
		ctx context.Context,
		spaceRef string,
	) (*types.SpaceCore, error)
}

func (s *SpaceFinderMock) FindByRef(
	ctx context.Context,
	spaceRef string,
) (*types.SpaceCore, error) {
	return s.FindByRefFn(ctx, spaceRef)
}

type MetricsMock struct {
	UpsertOptimisticFn func(ctx context.Context, in *types.UsageMetric) error
	GetMetricsFn       func(
		ctx context.Context,
		rootSpaceID int64,
		startDate int64,
		endDate int64,
	) (*types.UsageMetric, error)
	ListFn func(
		ctx context.Context,
		start int64,
		end int64,
	) ([]types.UsageMetric, error)
}

func (m *MetricsMock) GetMetrics(
	ctx context.Context,
	rootSpaceID int64,
	startDate int64,
	endDate int64,
) (*types.UsageMetric, error) {
	return m.GetMetricsFn(ctx, rootSpaceID, startDate, endDate)
}

func (m *MetricsMock) UpsertOptimistic(
	ctx context.Context,
	in *types.UsageMetric,
) error {
	return m.UpsertOptimisticFn(ctx, in)
}

func (m *MetricsMock) List(
	ctx context.Context,
	start int64,
	end int64,
) ([]types.UsageMetric, error) {
	return m.ListFn(ctx, start, end)
}
