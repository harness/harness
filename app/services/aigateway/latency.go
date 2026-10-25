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

package aigateway

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	gitnessstore "github.com/harness/gitness/store"

	"github.com/rs/zerolog/log"
)

const (
	// latencySamples is the number of most recent call durations kept per model.
	latencySamples = 200
	// latencyFlushEvery is the number of new samples of a model after which its p95 is persisted.
	latencyFlushEvery = 20
)

// latencyTracker keeps the durations of the last successful calls of every model.
type latencyTracker struct {
	mu     sync.Mutex
	models map[string]*latencyRing
}

type latencyRing struct {
	samples []time.Duration
	next    int
	dirty   int // samples added since the last flush
}

func newLatencyTracker() *latencyTracker {
	return &latencyTracker{models: map[string]*latencyRing{}}
}

// observe adds a sample and reports whether enough new samples accumulated to persist the p95.
func (t *latencyTracker) observe(model string, d time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	r := t.models[model]
	if r == nil {
		r = &latencyRing{}
		t.models[model] = r
	}
	if len(r.samples) < latencySamples {
		r.samples = append(r.samples, d)
	} else {
		r.samples[r.next] = d
		r.next = (r.next + 1) % latencySamples
	}
	r.dirty++
	return r.dirty >= latencyFlushEvery
}

// p95 returns the 95th percentile in milliseconds (nearest rank, at least 1 ms).
func (t *latencyTracker) p95(model string) (int, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	r := t.models[model]
	if r == nil || len(r.samples) == 0 {
		return 0, false
	}
	return percentile95(r.samples), true
}

// takeDirty returns the p95 of the models that got new samples and marks them clean.
func (t *latencyTracker) takeDirty() map[string]int {
	t.mu.Lock()
	defer t.mu.Unlock()

	out := map[string]int{}
	for id, r := range t.models {
		if r.dirty > 0 && len(r.samples) > 0 {
			out[id] = percentile95(r.samples)
			r.dirty = 0
		}
	}
	return out
}

func percentile95(samples []time.Duration) int {
	sorted := append([]time.Duration(nil), samples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	idx := (len(sorted)*95+99)/100 - 1
	ms := int(sorted[idx].Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return ms
}

// LatencyP95 returns the p95 latency in milliseconds of the model's recent successful calls on this replica.
func (g *Gateway) LatencyP95(model string) (int, bool) {
	return g.latency.p95(model)
}

// timedCall runs one attempt of a model call. Only the duration of a successful attempt is
// recorded, so retry sleeps and failures don't distort the latency.
func (g *Gateway) timedCall(ctx context.Context, model string, call func() error) error {
	start := g.now()
	if err := call(); err != nil {
		return err
	}
	if g.latency.observe(model, g.now().Sub(start)) {
		g.flushLatency(ctx)
	}
	return nil
}

// flushLatency persists the p95 of the models with new samples. Failures are logged only.
func (g *Gateway) flushLatency(ctx context.Context) {
	if g.modelStore == nil {
		return
	}
	dirty := g.latency.takeDirty()
	if len(dirty) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), storeTimeout)
	defer cancel()

	for id, p95 := range dirty {
		err := g.modelStore.UpdateLatency(ctx, id, p95)
		if err != nil && !errors.Is(err, gitnessstore.ErrResourceNotFound) {
			log.Ctx(ctx).Warn().Err(err).Str("model", id).Msg("ai gateway: failed to persist model latency")
		}
	}
}
