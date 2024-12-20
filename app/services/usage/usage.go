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
	"sync"
	"time"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

type Size struct {
	Bandwidth int64
	Storage   int64
}

type Metric struct {
	SpaceRef string
	Size
}

type SpaceStore interface {
	FindByRef(ctx context.Context, spaceRef string) (*types.Space, error)
	FindByIDs(ctx context.Context, spaceIDs ...int64) ([]*types.Space, error)
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

type LicenseFetcher interface {
	Fetch(ctx context.Context, spaceID int64) (*Size, error)
}

type Mediator struct {
	queue *queue

	mux     sync.RWMutex
	chunks  map[string]Size
	spaces  map[string]Size
	workers []*worker

	spaceStore   SpaceStore
	metricsStore MetricStore

	wg sync.WaitGroup

	config Config
}

func NewMediator(
	ctx context.Context,
	spaceStore SpaceStore,
	usageMetricsStore MetricStore,
	config Config,
) *Mediator {
	m := &Mediator{
		queue:        newQueue(),
		chunks:       make(map[string]Size),
		spaces:       make(map[string]Size),
		spaceStore:   spaceStore,
		metricsStore: usageMetricsStore,
		workers:      make([]*worker, config.MaxWorkers),
		config:       config,
	}

	m.initialize(ctx)
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

func (m *Mediator) Size(space string) Size {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.spaces[space]
}

// initialize will load when app is started all metrics for last 30 days.
func (m *Mediator) initialize(ctx context.Context) {
	m.mux.Lock()
	defer m.mux.Unlock()

	now := time.Now()

	metrics, err := m.metricsStore.List(ctx, now.Add(-m.days30()).UnixMilli(), now.UnixMilli())
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to list usage metrics")
		return
	}

	ids := make([]int64, len(metrics))
	values := make(map[int64]Size, len(metrics))
	for i, metric := range metrics {
		ids[i] = metric.RootSpaceID
		values[metric.RootSpaceID] = Size{
			Bandwidth: metric.Bandwidth,
			Storage:   metric.Storage,
		}
	}

	spaces, err := m.spaceStore.FindByIDs(ctx, ids...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to find spaces by id")
	}

	for _, space := range spaces {
		m.spaces[space.Identifier] = values[space.ID]
	}
}

func (m *Mediator) days30() time.Duration {
	return time.Duration(30*24) * time.Hour
}

func (m *Mediator) process(ctx context.Context, payload *Metric) {
	defer m.wg.Done()

	m.mux.Lock()
	defer m.mux.Unlock()

	size := m.chunks[payload.SpaceRef]
	m.chunks[payload.SpaceRef] = Size{
		Bandwidth: size.Bandwidth + payload.Size.Bandwidth,
		Storage:   size.Storage + payload.Size.Storage,
	}

	newSize := m.chunks[payload.SpaceRef]

	if newSize.Bandwidth < m.config.ChunkSize && newSize.Storage < m.config.ChunkSize {
		return
	}

	space, err := m.spaceStore.FindByRef(ctx, payload.SpaceRef)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to find space")
		return
	}

	if err = m.metricsStore.UpsertOptimistic(ctx, &types.UsageMetric{
		RootSpaceID: space.ID,
		Bandwidth:   newSize.Bandwidth,
		Storage:     newSize.Storage,
	}); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to upsert usage metrics")
	}

	m.chunks[payload.SpaceRef] = Size{
		Bandwidth: 0,
		Storage:   0,
	}

	now := time.Now()

	metric, err := m.metricsStore.GetMetrics(ctx, space.ID, now.Add(-m.days30()).UnixMilli(), now.UnixMilli())
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get usage metrics")
		return
	}

	m.spaces[space.Identifier] = Size{
		Bandwidth: metric.Bandwidth,
		Storage:   metric.Storage,
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
	log.Ctx(ctx).Info().Int("worker", w.id).Msg("starting worker")
	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Err(ctx.Err()).Msg("context canceled")
			return
		case <-w.stopCh:
			log.Ctx(ctx).Warn().Int("worker", w.id).Msg("worker is stopped")
			return
		default:
			payload, err := w.queue.Pop(ctx)
			if err != nil {
				log.Ctx(ctx).Err(err).Msg("failed to consume the queue")
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
