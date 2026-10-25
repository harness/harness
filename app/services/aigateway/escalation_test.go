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
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
)

// sequencedAnswers answers each decision request with the next noul probability.
func sequencedAnswers(model string, probabilities ...float64) func(w http.ResponseWriter) {
	var mu sync.Mutex
	i := 0
	return func(w http.ResponseWriter) {
		mu.Lock()
		p := probabilities[min(i, len(probabilities)-1)]
		i++
		mu.Unlock()

		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": model + "-build",
			"answers": map[string]any{
				"fix": map[string]any{"type": "noul", "noul": p},
			},
			"usage": map[string]any{"cost": 0},
		})
	}
}

func escalationGateway(t *testing.T, daily int, probabilities ...float64) (*Gateway, *fakeOpenRouter) {
	t.Helper()
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: sequencedAnswers(DefaultDecisionModel, probabilities...),
	}}
	return newTestGateway(t, f, daily), f
}

func escalationRequest() EscalationRequest {
	return EscalationRequest{
		SessionID:   "task-1",
		State:       map[string]any{"error": "panic: nil pointer"},
		Questions:   fixQuestion,
		KeyQuestion: "fix",
	}
}

func TestThresholdsValidate(t *testing.T) {
	for _, th := range []Thresholds{{High: 0.8, Low: 0.5}, {High: 1, Low: 0}, {High: 0.6, Low: 0.6}} {
		if err := th.Validate(); err != nil {
			t.Errorf("%+v should be valid: %v", th, err)
		}
	}
	for _, th := range []Thresholds{{High: 0.4, Low: 0.5}, {High: 1.1, Low: 0.5}, {High: 0.8, Low: -0.1}} {
		if err := th.Validate(); !errors.Is(err, ErrInvalidThresholds) {
			t.Errorf("%+v should be invalid, got %v", th, err)
		}
	}
	if d := DefaultThresholds(); d.High != 0.8 || d.Low != 0.5 {
		t.Errorf("unexpected defaults: %+v", d)
	}
}

func TestEscalationActsOnHighConfidence(t *testing.T) {
	g, f := escalationGateway(t, 100, 0.95)

	res, err := g.DecideWithEscalation(context.Background(), escalationRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeAct || res.Escalated || res.Second != nil || res.SecondOpinion != "" {
		t.Errorf("unexpected result: %+v", res)
	}
	if n := f.count("/api/v1/chat/completions"); n != 0 {
		t.Errorf("System 2 must not be consulted, got %d chats", n)
	}
}

func TestEscalationGoesToHumanOnLowConfidence(t *testing.T) {
	// P(true)=0.45 gives confidence 0.55, so use 0.5 thresholds shifted up
	g, f := escalationGateway(t, 100, 0.6)

	req := escalationRequest()
	req.Thresholds = Thresholds{High: 0.9, Low: 0.7}
	res, err := g.DecideWithEscalation(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeHuman || res.Escalated || res.Reason != "" {
		t.Errorf("unexpected result: %+v", res)
	}
	if n := f.count("/api/v1/chat/completions"); n != 0 {
		t.Errorf("System 2 must not be consulted, got %d chats", n)
	}
}

func TestEscalationSecondOpinionTurnsIntoAct(t *testing.T) {
	g, f := escalationGateway(t, 100, 0.65, 0.92)

	res, err := g.DecideWithEscalation(context.Background(), escalationRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeAct || !res.Escalated {
		t.Fatalf("unexpected outcome: %+v", res)
	}
	if res.First == nil || res.Second == nil || res.First.Decisions["fix"].Confidence != 0.65 ||
		res.Decision.Confidence != 0.92 {
		t.Errorf("both decisions must be recorded: %+v", res)
	}
	if res.SecondOpinion != "ok" || res.SecondOpinionModel != "vendor/coder:free" {
		t.Errorf("unexpected second opinion: %q by %q", res.SecondOpinion, res.SecondOpinionModel)
	}

	// the second decision carries the analysis in the state
	state := f.last()["state"].(map[string]any)
	if state["second_opinion"] != "ok" || state["error"] != "panic: nil pointer" {
		t.Errorf("second opinion not added to the state: %v", state)
	}
	if n := f.count("/api/alpha/decisions"); n != 2 {
		t.Errorf("expected two decision requests, got %d", n)
	}
}

func TestEscalationSecondOpinionStillUncertainGoesToHuman(t *testing.T) {
	g, _ := escalationGateway(t, 100, 0.65, 0.7)

	res, err := g.DecideWithEscalation(context.Background(), escalationRequest())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Outcome != OutcomeHuman || res.Second == nil || res.SecondOpinion == "" {
		t.Errorf("unexpected result: %+v", res)
	}
}

func TestEscalationWrapsNonObjectState(t *testing.T) {
	g, f := escalationGateway(t, 100, 0.65, 0.9)

	req := escalationRequest()
	req.State = "plain text state"
	if _, err := g.DecideWithEscalation(context.Background(), req); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	state := f.last()["state"].(map[string]any)
	if state["original"] != "plain text state" || state["second_opinion"] != "ok" {
		t.Errorf("state not wrapped: %v", state)
	}
}

func TestEscalationUsesRequestedModelAndRejectsPaid(t *testing.T) {
	g, f := escalationGateway(t, 100, 0.65, 0.9)

	req := escalationRequest()
	req.SecondOpinionModel = "vendor/small-free"
	res, err := g.DecideWithEscalation(context.Background(), req)
	if err != nil || res.SecondOpinionModel != "vendor/small-free" {
		t.Fatalf("unexpected result: %+v %v", res, err)
	}

	// a paid model never reaches OpenRouter: the System 2 failure sends the task to a human
	g2, f2 := escalationGateway(t, 100, 0.65)
	req.SecondOpinionModel = "vendor/paid"
	res, err = g2.DecideWithEscalation(context.Background(), req)
	if err != nil || res.Outcome != OutcomeHuman || !strings.Contains(res.Reason, "second opinion failed") {
		t.Fatalf("unexpected result: %+v %v", res, err)
	}
	if n := f2.count("/api/v1/chat/completions"); n != 0 {
		t.Errorf("paid model must not be called, got %d", n)
	}
	_ = f
}

func TestEscalationSystem2FailureGoesToHuman(t *testing.T) {
	g, f := escalationGateway(t, 100, 0.65)
	f.chatCost = 0.01 // a paid answer fails the free-only guard

	res, err := g.DecideWithEscalation(context.Background(), escalationRequest())
	if err != nil {
		t.Fatalf("a System 2 failure must not be an error: %v", err)
	}
	if res.Outcome != OutcomeHuman || res.Second != nil || res.Reason == "" || !res.Escalated {
		t.Errorf("unexpected result: %+v", res)
	}
	if n := f.count("/api/alpha/decisions"); n != 1 {
		t.Errorf("decision must not be asked again, got %d", n)
	}
}

func TestEscalationPropagatesQuotaErrors(t *testing.T) {
	// 4 requests a day: the decision class gets a single request, used by the first decision
	g, _ := escalationGateway(t, 4, 0.65)
	g.Planner().now = fixedClock("2026-10-25T08:00:00Z")

	res, err := g.DecideWithEscalation(context.Background(), escalationRequest())
	if !errors.Is(err, ErrQuotaExhausted) || res != nil {
		t.Fatalf("expected quota error from the System 2 call, got %+v %v", res, err)
	}

	// quota left for System 2 but not for the second decision
	g, _ = escalationGateway(t, 4, 0.65)
	g.Planner().now = fixedClock("2026-10-25T08:00:00Z")
	req := escalationRequest()
	req.Class = ClassTriage
	if _, err := g.DecideWithEscalation(context.Background(), req); !errors.Is(err, ErrQuotaExhausted) {
		t.Fatalf("expected quota error from the second decision, got %v", err)
	}
}

func TestEscalationDisabledAndInvalidRequests(t *testing.T) {
	var g *Gateway
	if _, err := g.DecideWithEscalation(context.Background(), escalationRequest()); !errors.Is(err, ErrDisabled) {
		t.Errorf("expected ErrDisabled, got %v", err)
	}

	g, f := escalationGateway(t, 100, 0.9)
	req := escalationRequest()
	req.KeyQuestion = "missing"
	if _, err := g.DecideWithEscalation(context.Background(), req); !errors.Is(err, ErrInvalidEscalation) {
		t.Errorf("expected ErrInvalidEscalation, got %v", err)
	}
	req = escalationRequest()
	req.Thresholds = Thresholds{High: 0.2, Low: 0.9}
	if _, err := g.DecideWithEscalation(context.Background(), req); !errors.Is(err, ErrInvalidThresholds) {
		t.Errorf("expected ErrInvalidThresholds, got %v", err)
	}
	if n := f.count("/api/alpha/decisions"); n != 0 {
		t.Errorf("invalid requests must not reach OpenRouter")
	}
}
