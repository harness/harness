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

package aigateway

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	aievents "github.com/harness/gitness/app/events/ai"
)

// fakeEventReporter records every event the gateway emits.
type fakeEventReporter struct {
	decisions []*aievents.DecisionMadePayload
	quotas    []*aievents.QuotaExhaustedPayload
	paid      []*aievents.PaidModelDetectedPayload
}

func (r *fakeEventReporter) EmitDecisionMade(_ context.Context, p *aievents.DecisionMadePayload) error {
	r.decisions = append(r.decisions, p)
	return nil
}

func (r *fakeEventReporter) EmitQuotaExhausted(_ context.Context, p *aievents.QuotaExhaustedPayload) error {
	r.quotas = append(r.quotas, p)
	return nil
}

func (r *fakeEventReporter) EmitPaidModelDetected(_ context.Context, p *aievents.PaidModelDetectedPayload) error {
	r.paid = append(r.paid, p)
	return nil
}

func TestEventsDecisionMadeOnceAndNotForCache(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: answer(DefaultDecisionModel, 0),
	}}
	g := newTestGateway(t, f, 100)
	rec := &fakeEventReporter{}
	g.SetEventReporter(rec)

	if _, err := g.Decide(context.Background(), "task-1", "state", fixQuestion); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.decisions) != 1 {
		t.Fatalf("expected one decision_made event, got %d", len(rec.decisions))
	}
	got := rec.decisions[0]
	if got.SessionID != "task-1" || got.Model != DefaultDecisionModel+"-build" || got.Fallback || got.Cached {
		t.Errorf("unexpected decision_made payload: %+v", got)
	}

	// the second identical call is served from cache and must not emit again
	if res, err := g.Decide(context.Background(), "task-1", "state", fixQuestion); err != nil || !res.Cached {
		t.Fatalf("expected cached result, got %+v %v", res, err)
	}
	if len(rec.decisions) != 1 {
		t.Errorf("cached decisions must not emit events, got %d", len(rec.decisions))
	}
	if len(rec.quotas) != 0 || len(rec.paid) != 0 {
		t.Errorf("unexpected events: quotas=%d paid=%d", len(rec.quotas), len(rec.paid))
	}
}

func TestEventsDecisionMadeReportsFallback(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         status(http.StatusNotFound),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 100)
	rec := &fakeEventReporter{}
	g.SetEventReporter(rec)

	if _, err := g.Decide(context.Background(), "", "state", fixQuestion); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.decisions) != 1 || !rec.decisions[0].Fallback {
		t.Fatalf("expected one fallback decision_made event, got %+v", rec.decisions)
	}
}

func TestEventsQuotaExhaustedOnce(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: answer(DefaultDecisionModel, 0),
	}}
	// same setup as TestDecideStopsWhenQuotaIsExhausted: one decision request a day
	g := newTestGateway(t, f, 4)
	g.Planner().now = fixedClock("2026-10-25T08:00:00Z")
	rec := &fakeEventReporter{}
	g.SetEventReporter(rec)

	if _, err := g.Decide(context.Background(), "", "a", fixQuestion); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := g.Decide(context.Background(), "", "b", fixQuestion); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("expected quota error, got %v", err)
	}
	if len(rec.quotas) != 1 {
		t.Fatalf("expected one quota_exhausted event, got %d", len(rec.quotas))
	}
	got := rec.quotas[0]
	nextDay := time.Date(2026, 10, 26, 0, 0, 0, 0, time.UTC).UnixMilli()
	if got.Class != string(ClassDecision) || !got.Daily || got.RetryAt != nextDay {
		t.Errorf("unexpected quota_exhausted payload: %+v", got)
	}
	if len(rec.decisions) != 1 {
		t.Errorf("refused request must not emit decision_made, got %d", len(rec.decisions))
	}
}

func TestEventsPaidModelDetectedOnDecide(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel:         answer(DefaultDecisionModel, 0.0001),
		DefaultDecisionFallbackModel: answer(DefaultDecisionFallbackModel, 0),
	}}
	g := newTestGateway(t, f, 100)
	rec := &fakeEventReporter{}
	g.SetEventReporter(rec)

	if _, err := g.Decide(context.Background(), "", "state", fixQuestion); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.paid) != 1 {
		t.Fatalf("expected one paid_model_detected event, got %d", len(rec.paid))
	}
	if rec.paid[0].Model != DefaultDecisionModel || rec.paid[0].CostUSD != 0.0001 {
		t.Errorf("unexpected paid_model_detected payload: %+v", rec.paid[0])
	}
	if len(rec.decisions) != 1 {
		t.Errorf("expected decision_made for the fallback answer, got %d", len(rec.decisions))
	}
}

func TestEventsPaidModelDetectedOnChat(t *testing.T) {
	f := &fakeOpenRouter{chatCost: 0.01}
	g := newTestGateway(t, f, 100)
	rec := &fakeEventReporter{}
	g.SetEventReporter(rec)

	req := ChatRequest{Model: "vendor/coder:free", Messages: []ChatMessage{{Role: "user", Content: "hi"}}}
	if _, err := g.Chat(context.Background(), ClassTriage, req); !errors.Is(err, ErrPaidUsage) {
		t.Fatalf("expected ErrPaidUsage, got %v", err)
	}
	if len(rec.paid) != 1 || rec.paid[0].Model != "vendor/coder:free" || rec.paid[0].CostUSD != 0.01 {
		t.Fatalf("expected one paid_model_detected event, got %+v", rec.paid)
	}
}

func TestEventsNilReporterEmitsNothing(t *testing.T) {
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: answer(DefaultDecisionModel, 0),
	}}
	g := newTestGateway(t, f, 100)
	g.SetEventReporter(nil)

	if _, err := g.Decide(context.Background(), "", "state", fixQuestion); err != nil {
		t.Fatalf("unexpected error without reporter: %v", err)
	}
}
