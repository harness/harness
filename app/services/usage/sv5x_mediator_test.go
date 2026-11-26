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

package usage

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/require"
)

var errSv5xBoom = errors.New("boom")

type sv5xSpaceFinder struct {
	SpaceFinder

	err error
}

func (f sv5xSpaceFinder) FindByRef(_ context.Context, ref string) (*types.SpaceCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.SpaceCore{ID: 5, Identifier: ref}, nil
}

type sv5xMetricStore struct {
	MetricStore

	err      error
	startMs  int64
	endMs    int64
	rootID   int64
	response *types.UsageMetric
}

func (s *sv5xMetricStore) GetMetrics(_ context.Context, rootID, start, end int64) (*types.UsageMetric, error) {
	s.rootID, s.startMs, s.endMs = rootID, start, end
	return s.response, s.err
}

func TestSv5xMediatorSend(t *testing.T) {
	t.Run("queues the metric", func(t *testing.T) {
		m := &Mediator{queue: make(chan Metric, 1)}
		require.NoError(t, m.Send(context.Background(), Metric{SpaceRef: "root", Pushes: 1}))
		require.Equal(t, Metric{SpaceRef: "root", Pushes: 1}, <-m.queue)
	})

	t.Run("drops the metric when the queue is full", func(t *testing.T) {
		m := &Mediator{queue: make(chan Metric)}
		require.NoError(t, m.Send(context.Background(), Metric{SpaceRef: "root"}))
		require.Empty(t, m.queue)
	})

	t.Run("canceled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		// Nobody receives from the queue, so only the context case is ready.
		m := &Mediator{queue: make(chan Metric)}
		require.ErrorIs(t, m.Send(ctx, Metric{}), context.Canceled)
	})
}

func TestSv5xMediatorSize(t *testing.T) {
	ctx := context.Background()

	t.Run("sums the bandwidth of the last 30 days", func(t *testing.T) {
		metrics := &sv5xMetricStore{response: &types.UsageMetric{BandwidthOut: 11, BandwidthIn: 7}}
		m := &Mediator{spaceFinder: sv5xSpaceFinder{}, metricsStore: metrics}

		got, err := m.Size(ctx, "root")
		require.NoError(t, err)
		require.Equal(t, Bandwidth{Out: 11, In: 7}, got)
		require.Equal(t, int64(5), metrics.rootID)
		require.Positive(t, metrics.endMs-metrics.startMs)
	})

	t.Run("space not found", func(t *testing.T) {
		m := &Mediator{spaceFinder: sv5xSpaceFinder{err: errSv5xBoom}, metricsStore: &sv5xMetricStore{}}
		_, err := m.Size(ctx, "root")
		require.ErrorIs(t, err, errSv5xBoom)
		require.ErrorContains(t, err, "could not find space")
	})

	t.Run("store failure", func(t *testing.T) {
		m := &Mediator{spaceFinder: sv5xSpaceFinder{}, metricsStore: &sv5xMetricStore{err: errSv5xBoom}}
		_, err := m.Size(ctx, "root")
		require.ErrorIs(t, err, errSv5xBoom)
	})
}

func TestSv5xNoopSender(t *testing.T) {
	var sender Sender = &Noop{}
	require.NoError(t, sender.Send(context.Background(), Metric{}))
}
