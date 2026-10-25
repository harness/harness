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

package aidecide

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeGateway struct {
	decide     func() (*aigateway.DecideResult, error)
	escalate   func() (*aigateway.EscalationResult, error)
	decideN    int
	escalateN  int
	redactedTo any
	// attributed is the attribution seen by the last gateway call.
	attributedSpace, attributedRepo *int64
}

func (f *fakeGateway) Decide(
	ctx context.Context, _ string, _ any, _ map[string]decisions.Question,
) (*aigateway.DecideResult, error) {
	f.attributedSpace, f.attributedRepo = aigateway.AttributionFrom(ctx)
	f.decideN++
	return f.decide()
}

func (f *fakeGateway) DecideWithEscalation(context.Context, aigateway.EscalationRequest) (*aigateway.EscalationResult, error) {
	f.escalateN++
	return f.escalate()
}

func (f *fakeGateway) RedactState(any) (any, error) { return f.redactedTo, nil }

type fakeModes struct{ mode enum.AIMode }

func (f fakeModes) Resolve(context.Context, *types.RepositoryCore) (enum.AIMode, error) {
	return f.mode, nil
}

type fakeRepos struct{}

func (fakeRepos) FindByID(_ context.Context, id int64) (*types.RepositoryCore, error) {
	return &types.RepositoryCore{ID: id, ParentID: 90 + id}, nil
}

type fakeStore struct{ rows []*types.AIDecision }

func (f *fakeStore) Create(_ context.Context, d *types.AIDecision) error {
	d.ID = int64(len(f.rows) + 1)
	f.rows = append(f.rows, d)
	return nil
}

func (f *fakeStore) Find(context.Context, int64) (*types.AIDecision, error) { return nil, nil }

func (f *fakeStore) ListByTask(context.Context, int64) ([]*types.AIDecision, error) { return nil, nil }

func (f *fakeStore) SetOutcome(context.Context, int64, enum.AIDecisionOutcome) error { return nil }

type fakeTasks struct {
	taskIDs []int64
	inputs  []task.DecisionSummaryInput
}

func (f *fakeTasks) RecordDecisionSummary(_ context.Context, _, taskID int64, in task.DecisionSummaryInput) error {
	f.taskIDs = append(f.taskIDs, taskID)
	f.inputs = append(f.inputs, in)
	return nil
}

func questions() map[string]decisions.Question {
	return map[string]decisions.Question{
		"severity": decisions.Score("severity?", "low", "high"),
		"action":   decisions.Choice("action?", map[string]string{"fix": "fix it", "ignore": "ignore it"}),
	}
}

func answers(sev, act string, conf float64) map[string]decisions.Decision {
	return map[string]decisions.Decision{
		"severity": {QuestionID: "severity", Type: decisions.TypeScore, Value: sev, Confidence: conf,
			Probabilities: map[string]float64{"1": conf}},
		"action": {QuestionID: "action", Type: decisions.TypeChoice, Value: act, Confidence: 0.74},
	}
}

func setup(g *fakeGateway, mode enum.AIMode) (*Service, *fakeStore, *fakeTasks) {
	st, tk := &fakeStore{}, &fakeTasks{}
	return NewService(g, fakeModes{mode}, fakeRepos{}, st, tk), st, tk
}

func TestDecideAttributesGatewayUsageToRepo(t *testing.T) {
	g := &fakeGateway{
		decide: func() (*aigateway.DecideResult, error) {
			return &aigateway.DecideResult{Model: "m", Decisions: answers("high", "fix", 0.8)}, nil
		},
	}
	svc, _, _ := setup(g, enum.AIModeFull)

	if _, err := svc.Decide(context.Background(), DecideInput{RepoID: 3, State: "s", Questions: questions()}); err != nil {
		t.Fatal(err)
	}
	if g.attributedSpace == nil || *g.attributedSpace != 93 || g.attributedRepo == nil || *g.attributedRepo != 3 {
		t.Fatalf("expected space 93 and repo 3, got %v/%v", g.attributedSpace, g.attributedRepo)
	}
}

func TestDecidePersistsRowsAndActivity(t *testing.T) {
	g := &fakeGateway{
		redactedTo: map[string]any{"a": "[REDACTED]"},
		decide: func() (*aigateway.DecideResult, error) {
			return &aigateway.DecideResult{Model: "m-build", Fallback: true, Decisions: answers("high", "fix", 0.81)}, nil
		},
	}
	svc, st, tk := setup(g, enum.AIModeFull)
	taskID := int64(7)

	res, err := svc.Decide(context.Background(), DecideInput{
		TaskID: &taskID, RepoID: 1, State: map[string]any{"a": "secret"}, Questions: questions(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.decideN != 1 || g.escalateN != 0 {
		t.Fatalf("unexpected gateway calls: %d/%d", g.decideN, g.escalateN)
	}
	if len(st.rows) != 2 || len(res.DecisionIDs) != 2 {
		t.Fatalf("want 2 rows, got %d", len(st.rows))
	}

	raw, _ := json.Marshal(g.redactedTo)
	sum := sha256.Sum256(raw)
	want := hex.EncodeToString(sum[:])
	for _, r := range st.rows {
		if r.ModelBuild != "m-build" || !r.Fallback || r.StateHash != want || r.TaskID == nil || *r.TaskID != 7 {
			t.Fatalf("unexpected row %+v", r)
		}
	}
	// rows are ordered by question id
	if st.rows[0].QuestionID != "action" || string(st.rows[1].Answer.Value) != `"high"` ||
		st.rows[1].Answer.Confidence != 0.81 {
		t.Fatalf("unexpected rows %+v %+v", st.rows[0], st.rows[1])
	}

	if len(tk.inputs) != 1 || tk.taskIDs[0] != 7 {
		t.Fatalf("want one activity, got %d", len(tk.inputs))
	}
	if got := tk.inputs[0].Summary; got != "action: fix (0.74) · severity: high (0.81)" {
		t.Fatalf("summary %q", got)
	}
	if len(tk.inputs[0].Decisions) != 2 || tk.inputs[0].Outcome != "" {
		t.Fatalf("unexpected payload %+v", tk.inputs[0])
	}
}

func TestDecideWithoutTaskSkipsActivity(t *testing.T) {
	g := &fakeGateway{decide: func() (*aigateway.DecideResult, error) {
		return &aigateway.DecideResult{Model: "m", Decisions: answers("low", "ignore", 0.9)}, nil
	}}
	svc, st, tk := setup(g, enum.AIModeDecisionsOnly)
	if _, err := svc.Decide(context.Background(), DecideInput{RepoID: 1, State: "s", Questions: questions()}); err != nil {
		t.Fatal(err)
	}
	if len(st.rows) != 2 || len(tk.inputs) != 0 || st.rows[0].TaskID != nil {
		t.Fatalf("rows=%d activities=%d", len(st.rows), len(tk.inputs))
	}
}

func TestDecideEscalation(t *testing.T) {
	long := strings.Repeat("é", 800) // 1600 bytes
	first := &aigateway.DecideResult{Model: "first", Decisions: answers("high", "fix", 0.6)}
	second := &aigateway.DecideResult{Model: "second", Decisions: answers("high", "fix", 0.9)}
	g := &fakeGateway{escalate: func() (*aigateway.EscalationResult, error) {
		return &aigateway.EscalationResult{
			Outcome: aigateway.OutcomeAct, First: first, Second: second, Escalated: true,
			SecondOpinion: long, SecondOpinionModel: "s2",
		}, nil
	}}
	svc, st, tk := setup(g, enum.AIModeFull)
	taskID := int64(3)

	res, err := svc.Decide(context.Background(), DecideInput{
		TaskID: &taskID, RepoID: 1, State: "s", Questions: questions(), KeyQuestion: "severity",
	})
	if err != nil {
		t.Fatal(err)
	}
	if g.escalateN != 1 || g.decideN != 0 || res.Model != "second" || res.Escalation == nil {
		t.Fatalf("unexpected result %+v", res)
	}
	if st.rows[0].ModelBuild != "second" {
		t.Fatalf("row build %q", st.rows[0].ModelBuild)
	}
	in := tk.inputs[0]
	if in.Outcome != "act" || in.SecondOpinionModel != "s2" || len(in.SecondOpinion) > 1024 || len(in.SecondOpinion) < 1000 {
		t.Fatalf("unexpected payload outcome=%q model=%q len=%d", in.Outcome, in.SecondOpinionModel, len(in.SecondOpinion))
	}
}

func TestDecideFallbackRulesWhenModelsFail(t *testing.T) {
	g := &fakeGateway{decide: func() (*aigateway.DecideResult, error) { return nil, errors.New("both models failed") }}
	svc, st, tk := setup(g, enum.AIModeFull)
	taskID := int64(1)
	rules := func(map[string]decisions.Question) map[string]decisions.Decision {
		return map[string]decisions.Decision{
			"severity": {Value: "high", Confidence: 1},
			"action":   {Value: "fix", Confidence: 1},
		}
	}

	res, err := svc.Decide(context.Background(), DecideInput{
		TaskID: &taskID, RepoID: 1, State: "s", Questions: questions(), Fallback: rules,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Rules || !res.Fallback || res.Model != RulesBuild {
		t.Fatalf("unexpected result %+v", res)
	}
	for _, r := range st.rows {
		if r.ModelBuild != "rules" || !r.Fallback {
			t.Fatalf("unexpected row %+v", r)
		}
	}
	if !strings.HasSuffix(tk.inputs[0].Summary, "[rules]") {
		t.Fatalf("summary %q", tk.inputs[0].Summary)
	}
}

func TestDecideFallbackIncomplete(t *testing.T) {
	g := &fakeGateway{decide: func() (*aigateway.DecideResult, error) { return nil, errors.New("boom") }}
	svc, st, _ := setup(g, enum.AIModeFull)
	partial := func(map[string]decisions.Question) map[string]decisions.Decision {
		return map[string]decisions.Decision{"severity": {Value: "high"}}
	}
	for _, fb := range []FallbackFunc{nil, partial} {
		_, err := svc.Decide(context.Background(), DecideInput{RepoID: 1, State: "s", Questions: questions(), Fallback: fb})
		if !errors.Is(err, ErrNoFallback) {
			t.Fatalf("want ErrNoFallback, got %v", err)
		}
	}
	if len(st.rows) != 0 {
		t.Fatal("nothing should be recorded")
	}
}

func TestDecideQuotaErrorsReturnedUnchanged(t *testing.T) {
	for _, quotaErr := range []error{aigateway.ErrQuotaExhausted} {
		g := &fakeGateway{
			decide:   func() (*aigateway.DecideResult, error) { return nil, quotaErr },
			escalate: func() (*aigateway.EscalationResult, error) { return nil, quotaErr },
		}
		svc, st, tk := setup(g, enum.AIModeFull)
		rules := func(map[string]decisions.Question) map[string]decisions.Decision {
			t.Fatal("rules must not run on quota errors")
			return nil
		}
		for _, key := range []string{"", "severity"} {
			_, err := svc.Decide(context.Background(), DecideInput{
				RepoID: 1, State: "s", Questions: questions(), KeyQuestion: key, Fallback: rules,
			})
			if err != quotaErr {
				t.Fatalf("want unchanged quota error, got %v", err)
			}
		}
		if len(st.rows) != 0 || len(tk.inputs) != 0 {
			t.Fatal("nothing should be recorded")
		}
	}
}

func TestDecideModeOffUsesRules(t *testing.T) {
	g := &fakeGateway{}
	svc, st, _ := setup(g, enum.AIModeOff)
	rules := func(map[string]decisions.Question) map[string]decisions.Decision {
		return map[string]decisions.Decision{"severity": {Value: "low"}, "action": {Value: "ignore"}}
	}

	res, err := svc.Decide(context.Background(), DecideInput{RepoID: 1, State: "s", Questions: questions(), Fallback: rules})
	if err != nil {
		t.Fatal(err)
	}
	if g.decideN != 0 || g.escalateN != 0 {
		t.Fatal("gateway must not be called when mode is off")
	}
	if !res.Rules || res.Mode != enum.AIModeOff || len(st.rows) != 2 || st.rows[0].ModelBuild != "rules" {
		t.Fatalf("unexpected result %+v", res)
	}

	if _, err := svc.Decide(context.Background(), DecideInput{RepoID: 1, State: "s", Questions: questions()}); !errors.Is(err, ErrNoFallback) {
		t.Fatalf("want ErrNoFallback, got %v", err)
	}
}

func TestDecideInvalidInput(t *testing.T) {
	svc, _, _ := setup(&fakeGateway{}, enum.AIModeFull)
	if _, err := svc.Decide(context.Background(), DecideInput{RepoID: 1}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v", err)
	}
	_, err := svc.Decide(context.Background(), DecideInput{RepoID: 1, Questions: questions(), KeyQuestion: "nope"})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("got %v", err)
	}
}
