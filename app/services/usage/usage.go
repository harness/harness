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
	"time"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

var (
	days30 = time.Duration(30*24) * time.Hour
)

type Bandwidth struct {
	Out int64
	In  int64
}

type Metric struct {
	SpaceRef string
	Bandwidth
	Pushes int64
}

type SpaceFinder interface {
	FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
}

type MetricStore interface {
	UpsertOptimistic(ctx context.Context, in *types.UsageMetric) error
	GetMetrics(
		ctx context.Context,
		rootSpaceID int64,
		startDate int64,
		endDate int64,
	) (*types.UsageMetric, error)
	List(
		ctx context.Context,
		start int64,
		end int64,
	) ([]types.UsageMetric, error)
}

type Mediator struct {
	queue *queue

	workers []*worker

	spaceFinder  SpaceFinder
	metricsStore MetricStore

	wg sync.WaitGroup

	config Config
}

func NewMediator(
	ctx context.Context,
	spaceFinder SpaceFinder,
	usageMetricsStore MetricStore,
	config Config,
) *Mediator {
	m := &Mediator{
		queue:        newQueue(),
		spaceFinder:  spaceFinder,
		metricsStore: usageMetricsStore,
		workers:      make([]*worker, config.MaxWorkers),
		config:       config,
	}

	m.Start(ctx)

	return m
}

func (m *Mediator) Start(ctx context.Context) {
	for i := range m.workers {
		w := newWorker(i, m.queue)
		go w.start(ctx, m.process)
		m.workers[i] = w
	}
}

func (m *Mediator) Stop() {
	for i := range m.workers {
		m.workers[i].stop()
	}
}

func (m *Mediator) Send(ctx context.Context, payload Metric) error {
	m.wg.Add(1)
	m.queue.Add(ctx, payload)
	return nil
}

func (m *Mediator) Wait() {
	m.wg.Wait()
}

func (m *Mediator) Size(ctx context.Context, spaceRef string) (Bandwidth, error) {
	space, err := m.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return Bandwidth{}, fmt.Errorf("could not find space: %w", err)
	}
	now := time.Now()
	metric, err := m.metricsStore.GetMetrics(ctx, space.ID, now.Add(-days30).UnixMilli(), now.UnixMilli())
	if err != nil {
		return Bandwidth{}, err
	}
	return Bandwidth{
		Out: metric.Bandwidth,
		In:  metric.Storage,
	}, nil
}

func (m *Mediator) process(ctx context.Context, payload *Metric) {
	defer m.wg.Done()

	space, err := m.spaceFinder.FindByRef(ctx, payload.SpaceRef)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to find space")
		return
	}

	if err = m.metricsStore.UpsertOptimistic(ctx, &types.UsageMetric{
		RootSpaceID: space.ID,
		Bandwidth:   payload.Out,
		Storage:     payload.In,
		Pushes:      payload.Pushes,
	}); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to upsert usage metrics")
	}
}

type worker struct {
	id     int
	queue  *queue
	stopCh chan struct{}
}

func newWorker(id int, queue *queue) *worker {
	return &worker{
		id:     id,
		queue:  queue,
		stopCh: make(chan struct{}),
	}
}

func (w *worker) start(ctx context.Context, fn func(context.Context, *Metric)) {
	log.Ctx(ctx).Info().Int("usage-worker", w.id).Msg("usage metrics starting worker")
	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Err(ctx.Err()).Msg("context canceled")
			return
		case <-w.stopCh:
			log.Ctx(ctx).Warn().Int("usage-worker", w.id).Msg("worker is stopped")
			return
		default:
			payload, err := w.queue.Pop(ctx)
			if err != nil {
				log.Ctx(ctx).Err(err).Int("usage-worker", w.id).Msg("failed to consume the queue")
				return
			}
			fn(ctx, payload)
		}
	}
}

func (w *worker) stop() {
	defer close(w.stopCh)
	w.stopCh <- struct{}{}
}

type Noop struct{}

func (n *Noop) Send(
	context.Context,
	Metric,
) error {
	return nil
}
