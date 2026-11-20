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
	"time"

	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

const QueueSize = 128

// shutdownFlushTimeout bounds the final flush, which runs detached from the cancelled service context.
const shutdownFlushTimeout = 30 * time.Second

var (
	days30 = time.Duration(30*24) * time.Hour
)

type Bandwidth struct {
	Out int64
	In  int64
}

type Metric struct {
	Time     time.Time
	SpaceRef string
	Bandwidth
	StorageTotal    int64
	LFSStorageTotal int64
	Pushes          int64
}

type SpaceFinder interface {
	FindByRef(ctx context.Context, spaceRef string) (*types.SpaceCore, error)
}

type MetricStore interface {
	Upsert(ctx context.Context, in []*types.UsageMetric) error
	UpsertStorage(ctx context.Context, in []*types.UsageMetric) error
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
	ctx   context.Context
	queue chan Metric

	spaceFinder  SpaceFinder
	metricsStore MetricStore

	config Config

	lastUpdated time.Time
	cache       map[string]*Metric
}

func NewMediator(
	ctx context.Context,
	spaceFinder SpaceFinder,
	usageMetricsStore MetricStore,
	config Config,
) *Mediator {
	config.Sanitize()

	m := &Mediator{
		ctx:          ctx,
		queue:        make(chan Metric, config.QueueSize),
		spaceFinder:  spaceFinder,
		metricsStore: usageMetricsStore,
		config:       config,
		cache:        make(map[string]*Metric),
	}

	go m.Start(ctx)

	return m
}

func (m *Mediator) Start(ctx context.Context) {
	flushFn := func(ctx context.Context) {
		usageMetrics := make([]*Metric, 0, len(m.cache))
		for _, metric := range m.cache {
			usageMetrics = append(usageMetrics, metric)
			delete(m.cache, metric.SpaceRef)
		}

		m.process(ctx, usageMetrics)
		m.lastUpdated = time.Now()
	}

	ticker := time.NewTicker(m.config.FlushInterval)
	defer ticker.Stop()

	log.Ctx(ctx).Info().Msg("starting usage metrics service")
	for {
		select {
		case <-ctx.Done():
			log.Ctx(ctx).Warn().Msg("context canceled")
			// The service context is already cancelled, so the final flush has to run detached from it,
			// otherwise every lookup and write below fails with context.Canceled.
			flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownFlushTimeout)
			flushFn(flushCtx)
			cancel()
			return
		case <-ticker.C:
			flushFn(ctx)
		case payload := <-m.queue:
			cache, ok := m.cache[payload.SpaceRef]
			if !ok {
				m.cache[payload.SpaceRef] = &payload
				continue
			}

			cache.Bandwidth.Out += payload.Bandwidth.Out
			cache.Bandwidth.In += payload.Bandwidth.In
			cache.StorageTotal += payload.StorageTotal
			cache.LFSStorageTotal += payload.LFSStorageTotal
			cache.Pushes += payload.Pushes
		}
	}
}

func (m *Mediator) Send(ctx context.Context, payload Metric) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case m.queue <- payload:
	default:
		log.Ctx(ctx).Warn().Msg("usage metric dropped: queue full")
	}
	return nil
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
		Out: metric.BandwidthOut,
		In:  metric.BandwidthIn,
	}, nil
}

func (m *Mediator) process(ctx context.Context, payload []*Metric) {
	if payload == nil {
		return
	}

	metrics := make([]*types.UsageMetric, len(payload))
	for i, metric := range payload {
		space, err := m.spaceFinder.FindByRef(ctx, metric.SpaceRef)
		if err != nil {
			log.Ctx(ctx).Err(err).Str("space", metric.SpaceRef).Msg("failed to find space")
			return
		}

		metrics[i] = &types.UsageMetric{
			Date:                metric.Time,
			RootSpaceID:         space.ID,
			RootSpaceIdentifier: space.Identifier,
			BandwidthOut:        metric.Out,
			BandwidthIn:         metric.In,
			StorageTotal:        metric.StorageTotal,
			LFSStorageTotal:     metric.LFSStorageTotal,
			Pushes:              metric.Pushes,
		}
	}

	if err := m.metricsStore.Upsert(ctx, metrics); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to upsert usage metrics")
	} else if len(metrics) > 0 {
		log.Ctx(ctx).Info().Msg("flush usage metrics data to db")
	}
}

type Noop struct{}

func (n *Noop) Send(
	context.Context,
	Metric,
) error {
	return nil
}
