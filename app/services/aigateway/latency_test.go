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
	"net/http"
	"testing"
	"time"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestLatencyTrackerP95AndRing(t *testing.T) {
	tr := newLatencyTracker()
	if _, ok := tr.p95("m"); ok {
		t.Fatal("no samples, no p95")
	}
	for i := 1; i <= 100; i++ {
		tr.observe("m", time.Duration(i)*time.Millisecond)
	}
	if got, _ := tr.p95("m"); got != 95 {
		t.Fatalf("expected p95 95ms, got %d", got)
	}

	// the ring keeps only the last latencySamples: slow old samples are forgotten
	for i := 0; i < latencySamples; i++ {
		tr.observe("m", 10*time.Millisecond)
	}
	if got, _ := tr.p95("m"); got != 10 {
		t.Fatalf("expected old samples to be dropped, got %d", got)
	}
	if n := len(tr.models["m"].samples); n != latencySamples {
		t.Fatalf("ring grew to %d", n)
	}

	tr.observe("fast", 0)
	if got, _ := tr.p95("fast"); got != 1 {
		t.Fatalf("expected minimum of 1ms, got %d", got)
	}
}

func TestLatencyTrackerFlushThreshold(t *testing.T) {
	tr := newLatencyTracker()
	for i := 1; i < latencyFlushEvery; i++ {
		if tr.observe("m", time.Second) {
			t.Fatalf("flush requested after %d samples", i)
		}
	}
	if !tr.observe("m", time.Second) {
		t.Fatal("flush expected")
	}
	if got := tr.takeDirty(); got["m"] != 1000 {
		t.Fatalf("unexpected dirty set %v", got)
	}
	if got := tr.takeDirty(); len(got) != 0 {
		t.Fatalf("dirty must be cleared, got %v", got)
	}
}

// steppingClock advances by step on every call, so each timed call lasts exactly step.
func steppingClock(step time.Duration) func() time.Time {
	now := time.Unix(0, 0)
	return func() time.Time {
		now = now.Add(step)
		return now
	}
}

func TestChatRecordsSuccessfulCallLatencyAndPersistsIt(t *testing.T) {
	g := newTestGateway(t, &fakeOpenRouter{}, 1000)
	g.planner = NewPlanner(1000, 1000, nil) // more calls than the per-minute default
	g.now = steppingClock(300 * time.Millisecond)
	ms := newFakeModelStore()
	ms.models["vendor/coder:free"] = &types.AIModel{ID: "vendor/coder:free", Status: enum.AIModelStatusActive}
	g.modelStore = ms

	for i := 0; i < latencyFlushEvery; i++ {
		if _, err := g.Chat(context.Background(), ClassManual, chatRequest()); err != nil {
			t.Fatal(err)
		}
	}
	if got, ok := g.LatencyP95("vendor/coder:free"); !ok || got != 300 {
		t.Fatalf("expected p95 300ms, got %d (%v)", got, ok)
	}
	if got := ms.models["vendor/coder:free"].P95LatencyMS; got != 300 {
		t.Fatalf("expected p95 persisted after %d samples, got %d", latencyFlushEvery, got)
	}
}

func TestDecideRecordsLatencyAndIgnoresFailures(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         status(http.StatusBadRequest),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 1000)
	g.now = steppingClock(120 * time.Millisecond)

	if _, err := g.Decide(context.Background(), "s", map[string]any{"a": 1}, fixQuestion); err != nil {
		t.Fatal(err)
	}
	if _, ok := g.LatencyP95(DefaultDecisionModel); ok {
		t.Fatal("a failed call must not be recorded")
	}
	if got, ok := g.LatencyP95(DefaultDecisionFallbackModel); !ok || got != 120 {
		t.Fatalf("expected 120ms for the model that answered, got %d (%v)", got, ok)
	}
}

func TestFlushLatencyPersistsPartialBatch(t *testing.T) {
	g := newTestGateway(t, &fakeOpenRouter{}, 1000)
	ms := newFakeModelStore()
	ms.models["a/b:free"] = &types.AIModel{ID: "a/b:free"}
	g.modelStore = ms

	g.latency.observe("a/b:free", 2*time.Second)
	g.latency.observe("gone/model:free", time.Second) // not stored: must not fail
	g.flushLatency(context.Background())

	if got := ms.models["a/b:free"].P95LatencyMS; got != 2000 {
		t.Fatalf("expected 2000, got %d", got)
	}
}
