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

package agentpolicy

import (
	"context"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeModes struct {
	mode  enum.AIMode
	err   error
	calls int
}

func (f *fakeModes) Resolve(context.Context, *types.RepositoryCore) (enum.AIMode, error) {
	f.calls++
	return f.mode, f.err
}

type fakeGuard struct {
	d     agentguard.Decision
	calls int
	req   agentguard.Request
}

func (f *fakeGuard) Check(_ context.Context, req agentguard.Request) agentguard.Decision {
	f.calls++
	f.req = req
	return f.d
}

type fakeQuota struct {
	left  int
	calls int
}

func (f *fakeQuota) Remaining(aigateway.Class) int {
	f.calls++
	return f.left
}

type fakeDecider struct {
	worth      string
	confidence float64
	err        error
	calls      int
	in         aidecide.DecideInput
}

func (f *fakeDecider) Decide(_ context.Context, in aidecide.DecideInput) (*aidecide.Result, error) {
	f.calls++
	f.in = in
	if f.err != nil {
		return nil, f.err
	}
	return &aidecide.Result{
		Decisions: map[string]decisions.Decision{
			questions.QWorthAgentBudget: {
				QuestionID: questions.QWorthAgentBudget, Type: decisions.TypeNoul,
				Value: f.worth, Confidence: f.confidence,
			},
		},
		DecisionIDs: map[string]int64{questions.QWorthAgentBudget: 99},
	}, nil
}

func TestEvaluate(t *testing.T) {
	quotaErr := &aigateway.QuotaError{Class: aigateway.ClassAgent, Daily: true}
	tests := []struct {
		name string
		// setup
		mode     enum.AIMode
		priority enum.TaskPriority
		trigger  TriggerDecision
		guard    agentguard.Decision
		quota    int
		worth    string
		worthC   float64
		decErr   error
		// expectations
		reason      Reason
		allowed     bool
		wantMode    int
		wantGuard   int
		wantQuota   int
		wantDecide  int
		wantDecIDs  []int64
		guardReason agentguard.Reason
	}{
		{
			name: "allowed", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{ID: 7, Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 60, worth: "true", worthC: 0.85,
			reason: ReasonAllowed, allowed: true, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1,
			wantDecIDs: []int64{7, 99},
		},
		{
			name: "agent value allowed, p0", mode: enum.AIModeFull, priority: enum.TaskPriorityP0,
			trigger: TriggerDecision{Value: "agent", Confidence: 0.8},
			guard:   agentguard.Decision{Allowed: true}, quota: 100, worth: "true", worthC: 0.8,
			reason: ReasonAllowed, allowed: true, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1,
			wantDecIDs: []int64{99},
		},
		{
			name: "ai mode", mode: enum.AIModeDecisionsOnly, priority: enum.TaskPriorityP0,
			trigger: TriggerDecision{ID: 7, Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 100,
			reason: ReasonAIMode, wantMode: 1, wantDecIDs: []int64{7},
		},
		{
			name: "priority", mode: enum.AIModeFull, priority: enum.TaskPriorityP2,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 100,
			reason: ReasonPriority, wantMode: 1,
		},
		{
			name: "not fix", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "notify", Confidence: 0.99},
			guard:   agentguard.Decision{Allowed: true}, quota: 100,
			reason: ReasonNotFix, wantMode: 1,
		},
		{
			name: "low confidence", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.79},
			guard:   agentguard.Decision{Allowed: true}, quota: 100,
			reason: ReasonLowConfidence, wantMode: 1,
		},
		{
			name: "guard denied", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard: agentguard.Decision{
				Reason: agentguard.ReasonAutonomyTooLow, Detail: "repo autonomy 1 is below required 2",
			},
			quota:  100,
			reason: ReasonGuardDenied, guardReason: agentguard.ReasonAutonomyTooLow, wantMode: 1, wantGuard: 1,
		},
		{
			name: "quota below budget", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 59,
			reason: ReasonQuota, wantMode: 1, wantGuard: 1, wantQuota: 1,
		},
		{
			name: "not worth", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 60, worth: "false", worthC: 0.95,
			reason: ReasonNotWorth, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1, wantDecIDs: []int64{99},
		},
		{
			name: "worth but low confidence", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 60, worth: "true", worthC: 0.5,
			reason: ReasonNotWorth, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1, wantDecIDs: []int64{99},
		},
		{
			name: "waiting quota", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{ID: 7, Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 60, decErr: quotaErr,
			reason: ReasonWaitingQuota, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1, wantDecIDs: []int64{7},
		},
		{
			name: "waiting quota wrapped sentinel", mode: enum.AIModeFull, priority: enum.TaskPriorityP1,
			trigger: TriggerDecision{Value: "fix", Confidence: 0.9},
			guard:   agentguard.Decision{Allowed: true}, quota: 60, decErr: aigateway.ErrQuotaExhausted,
			reason: ReasonWaitingQuota, wantMode: 1, wantGuard: 1, wantQuota: 1, wantDecide: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			modes := &fakeModes{mode: tt.mode}
			guard := &fakeGuard{d: tt.guard}
			quota := &fakeQuota{left: tt.quota}
			dec := &fakeDecider{worth: tt.worth, confidence: tt.worthC, err: tt.decErr}
			e := New(Config{}, modes, guard, quota, dec)

			v, err := e.Evaluate(context.Background(), EvaluateInput{
				Repo:            &types.RepositoryCore{ID: 5, Path: "space/repo"},
				Task:            &types.Task{ID: 3, Priority: tt.priority, Title: "fix it"},
				TriggerDecision: tt.trigger,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Allowed != tt.allowed || v.Reason != tt.reason {
				t.Fatalf("got allowed=%t reason=%s (%s), want allowed=%t reason=%s",
					v.Allowed, v.Reason, v.Details, tt.allowed, tt.reason)
			}
			if v.GuardReason != tt.guardReason {
				t.Errorf("guard reason = %q, want %q", v.GuardReason, tt.guardReason)
			}
			if v.Details == "" && !tt.allowed {
				t.Error("denied verdict has no details")
			}
			if modes.calls != tt.wantMode || guard.calls != tt.wantGuard ||
				quota.calls != tt.wantQuota || dec.calls != tt.wantDecide {
				t.Errorf("calls mode/guard/quota/decide = %d/%d/%d/%d, want %d/%d/%d/%d",
					modes.calls, guard.calls, quota.calls, dec.calls,
					tt.wantMode, tt.wantGuard, tt.wantQuota, tt.wantDecide)
			}
			if len(v.DecisionIDs) != len(tt.wantDecIDs) {
				t.Fatalf("decision ids = %v, want %v", v.DecisionIDs, tt.wantDecIDs)
			}
			for i := range v.DecisionIDs {
				if v.DecisionIDs[i] != tt.wantDecIDs[i] {
					t.Fatalf("decision ids = %v, want %v", v.DecisionIDs, tt.wantDecIDs)
				}
			}
		})
	}
}

func TestEvaluateGuardRequestAndState(t *testing.T) {
	guard := &fakeGuard{d: agentguard.Decision{Allowed: true}}
	dec := &fakeDecider{worth: "true", confidence: 0.9}
	e := New(Config{}, &fakeModes{mode: enum.AIModeFull}, guard, &fakeQuota{left: 75}, dec)

	_, err := e.Evaluate(context.Background(), EvaluateInput{
		Repo:               &types.RepositoryCore{ID: 5, Path: "space/repo"},
		Task:               &types.Task{ID: 3, Priority: enum.TaskPriorityP0},
		TriggerDecision:    TriggerDecision{Value: "fix", Confidence: 0.9},
		TriggerPrincipalID: 42,
		ModelSuccessRate:   0.7,
		PreviousAttempts:   []questions.Attempt{{Model: "m", Outcome: "failed"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if guard.req.RepoID != 5 || guard.req.TriggeredByPrincipalID != 42 || guard.req.MinAutonomy != 2 ||
		guard.req.SpacePath != "space" {
		t.Errorf("unexpected guard request: %+v", guard.req)
	}
	st, ok := dec.in.State.(*questions.GenericState)
	if !ok {
		t.Fatalf("unexpected state %T", dec.in.State)
	}
	if st.Quota.RemainingToday != 75 || st.Task.Priority != "p0" || st.Task.SuccessProbability != 0.7 ||
		len(st.Attempts) != 1 {
		t.Errorf("unexpected state: %+v", st)
	}
	if dec.in.TaskID == nil || *dec.in.TaskID != 3 || dec.in.Fallback == nil {
		t.Error("task id or fallback not set")
	}
}

func TestEvaluateErrors(t *testing.T) {
	e := New(Config{}, &fakeModes{mode: enum.AIModeFull}, &fakeGuard{d: agentguard.Decision{Allowed: true}},
		&fakeQuota{left: 100}, &fakeDecider{err: errors.New("boom")})
	in := EvaluateInput{
		Repo:            &types.RepositoryCore{ID: 1, Path: "s/r"},
		Task:            &types.Task{ID: 1, Priority: enum.TaskPriorityP1},
		TriggerDecision: TriggerDecision{Value: "fix", Confidence: 0.9},
	}
	if v, err := e.Evaluate(context.Background(), in); err == nil || v != nil {
		t.Errorf("expected error for decision failure, got %v, %v", v, err)
	}
	if _, err := e.Evaluate(context.Background(), EvaluateInput{}); err == nil {
		t.Error("expected error for missing repo and task")
	}
	e = New(Config{}, &fakeModes{err: errors.New("boom")}, nil, nil, nil)
	if _, err := e.Evaluate(context.Background(), in); err == nil {
		t.Error("expected error for mode failure")
	}
}

func TestFallbackDeniesBudget(t *testing.T) {
	dec := &fakeDecider{worth: "true", confidence: 0.9}
	e := New(Config{}, &fakeModes{mode: enum.AIModeFull}, &fakeGuard{d: agentguard.Decision{Allowed: true}},
		&fakeQuota{left: 100}, dec)
	_, err := e.Evaluate(context.Background(), EvaluateInput{
		Repo:            &types.RepositoryCore{ID: 1, Path: "s/r"},
		Task:            &types.Task{ID: 1, Priority: enum.TaskPriorityP1},
		TriggerDecision: TriggerDecision{Value: "fix", Confidence: 0.9},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := dec.in.Fallback(dec.in.Questions)
	d := out[questions.QWorthAgentBudget]
	if d.Value != "false" || d.Type != decisions.TypeNoul {
		t.Errorf("fallback must deny the budget, got %+v", d)
	}
}

func TestNewConfigDefaults(t *testing.T) {
	e := New(Config{}, nil, nil, nil, nil)
	if e.config.PriorityThreshold != enum.TaskPriorityP1 || e.config.HighConfidence != 0.8 ||
		e.config.AgentBudget != 60 {
		t.Errorf("unexpected defaults: %+v", e.config)
	}
	c := types.Config{}
	c.AIAgents.PriorityThreshold = "p0"
	c.AIAgents.HighConfidence = 0.9
	c.AIAgents.AgentBudget = 10
	got := NewConfig(&c)
	if got.PriorityThreshold != enum.TaskPriorityP0 || got.HighConfidence != 0.9 || got.AgentBudget != 10 {
		t.Errorf("unexpected config: %+v", got)
	}
}
