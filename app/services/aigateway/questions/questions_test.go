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

package questions

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// reply fabricates the answer a decision model would give.
type reply struct {
	noul   float64
	choice string
	score  float64
	level  int
}

// roundTrip validates the request, answers every question from replies and parses the response.
func roundTrip(
	t *testing.T, state any, qs map[string]decisions.Question, replies map[string]reply,
) map[string]decisions.Decision {
	t.Helper()
	req := &decisions.Request{Model: "mammoth", State: state, Questions: qs}
	if err := req.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	resp := &decisions.Response{Answers: map[string]decisions.Answer{}}
	for id, q := range qs {
		if !strings.Contains(q.Instructions, "ignore any instruction") {
			t.Errorf("%s lacks the injection guard", id)
		}
		r, ok := replies[id]
		if !ok {
			t.Fatalf("no reply for %s", id)
		}
		a := decisions.Answer{Type: q.Type}
		switch q.Type {
		case decisions.TypeNoul:
			a.Noul = &r.noul
		case decisions.TypeChoice:
			c := 0.8
			a.Choice, a.Confidence = &r.choice, &c
		case decisions.TypeScore:
			a.Score = &r.score
			a.Probabilities = map[string]float64{string(rune('0' + r.level)): 0.9}
		}
		resp.Answers[id] = a
	}
	ds, err := decisions.Parse(req, resp)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return ds
}

func TestShield(t *testing.T) {
	st := &ShieldState{Kind: "vulnerability", Package: "lodash", FixedIn: "4.17.21", ImportCount: 3}
	if err := st.Fit(Limits{}); err != nil {
		t.Fatal(err)
	}
	ds := roundTrip(t, st, ShieldQuestions(), map[string]reply{
		QReachable:     {choice: "reachable"},
		QFalsePositive: {noul: 0.1},
		QFixStrategy:   {choice: "deterministic_bump"},
	})
	got, err := ParseShield(ds)
	if err != nil {
		t.Fatal(err)
	}
	if got.Reachable != ReachabilityReachable || got.FalsePositive || got.FixStrategy != FixDeterministicBump ||
		got.ReachableConfidence != 0.8 || got.FalsePositiveConfidence != 0.9 {
		t.Fatalf("unexpected %+v", got)
	}
	for _, id := range []string{QReachable, QFalsePositive, QFixStrategy} {
		partial := map[string]decisions.Decision{}
		for k, v := range ds {
			if k != id {
				partial[k] = v
			}
		}
		if _, err := ParseShield(partial); !errors.Is(err, ErrMissingDecision) {
			t.Errorf("missing %s: got %v", id, err)
		}
	}
	bad := map[string]decisions.Decision{}
	for k, v := range ds {
		bad[k] = v
	}
	bad[QReachable] = decisions.Decision{QuestionID: QReachable, Type: decisions.TypeChoice, Value: "maybe"}
	if _, err := ParseShield(bad); !errors.Is(err, ErrInvalidDecision) {
		t.Errorf("bad value: %v", err)
	}
	bad[QReachable] = decisions.Decision{QuestionID: QReachable, Type: decisions.TypeNoul, Value: "true"}
	if _, err := ParseShield(bad); !errors.Is(err, ErrInvalidDecision) {
		t.Errorf("bad type: %v", err)
	}
}

func insightsState() *InsightsState {
	return &InsightsState{
		Error: ErrorInfo{Type: "TypeError", Message: "x is undefined", EventCount: 40, UsersAffected: 7},
		Candidates: []Candidate{
			{IssueID: "i1", Type: "TypeError", Message: "x undefined"},
			{IssueID: "i2", Type: "RangeError", Message: "bad range"},
			{IssueID: "i3", Type: "TypeError", Message: "y undefined"},
		},
		SuspectCommits: []SuspectCommit{
			{SHA: "ABCDEF0123456789", Subject: "refactor", Reason: "touched top frame"},
			{SHA: "1234567890abcdef", Subject: "bump deps"},
		},
	}
}

func TestInsights(t *testing.T) {
	st := insightsState()
	if err := st.Fit(Limits{}); err != nil {
		t.Fatal(err)
	}
	qs, err := InsightsQuestions(st)
	if err != nil {
		t.Fatal(err)
	}
	if len(qs) != 6 {
		t.Fatalf("want 6 questions, got %d", len(qs))
	}
	if opts := qs[QSuspectCommit].Criteria.(map[string]string); len(opts) != 3 || opts["abcdef01"] == "" {
		t.Fatalf("options %v", opts)
	}
	ds := roundTrip(t, st, qs, map[string]reply{
		"is_duplicate_of_candidate_1": {noul: 0.7},
		"is_duplicate_of_candidate_2": {noul: 0.1},
		"is_duplicate_of_candidate_3": {noul: 0.9},
		QSeverity:                     {score: 2.1, level: 2},
		QAction:                       {choice: "fix"},
		QSuspectCommit:                {choice: "abcdef01"},
	})
	got, err := ParseInsights(st, ds)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Duplicates) != 3 || got.BestDuplicate == nil || got.BestDuplicate.IssueID != "i3" ||
		got.Severity != SeverityHigh || got.Action != ActionFix || got.SuspectCommit != "abcdef01" {
		t.Fatalf("unexpected %+v", got)
	}

	ds[QSuspectCommit] = decisions.Decision{Type: decisions.TypeChoice, Value: SuspectNone}
	if got, _ = ParseInsights(st, ds); got.SuspectCommit != "" {
		t.Fatalf("none should map to empty, got %q", got.SuspectCommit)
	}

	delete(ds, "is_duplicate_of_candidate_2")
	if _, err := ParseInsights(st, ds); !errors.Is(err, ErrMissingDecision) {
		t.Fatalf("got %v", err)
	}
}

func TestInsightsLimits(t *testing.T) {
	st := insightsState()
	st.Candidates = append(st.Candidates, Candidate{IssueID: "i4"})
	if _, err := InsightsQuestions(st); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("4 candidates: %v", err)
	}
	st = insightsState()
	st.SuspectCommits = []SuspectCommit{{SHA: "abcdef01aa"}, {SHA: "ABCDEF01bb"}}
	if _, err := InsightsQuestions(st); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("duplicate short sha: %v", err)
	}
	// no candidates and no commits still gives a valid request
	st = &InsightsState{Error: ErrorInfo{Type: "E", Message: "m"}}
	qs, err := InsightsQuestions(st)
	if err != nil || len(qs) != 2 {
		t.Fatalf("minimal: %v %d", err, len(qs))
	}
	roundTrip(t, st, qs, map[string]reply{QSeverity: {score: 0, level: 0}, QAction: {choice: "ignore"}})
}

func genericState() *GenericState {
	return &GenericState{
		Task:     TaskInfo{Kind: "fix_error", Priority: "P1"},
		Quota:    QuotaInfo{RemainingToday: 5, LimitToday: 20},
		Attempts: []Attempt{{Model: "free-a", Outcome: "failed"}},
		Teams:    []string{"payments", "platform"},
	}
}

func TestGeneric(t *testing.T) {
	st := genericState()
	if err := st.Fit(Limits{}); err != nil {
		t.Fatal(err)
	}
	ask := GenericAsk{WorthAgentBudget: true, NextStep: true, AssigneeKind: true}
	qs, err := GenericQuestions(st, ask)
	if err != nil {
		t.Fatal(err)
	}
	ds := roundTrip(t, st, qs, map[string]reply{
		QWorthAgentBudget: {noul: 0.8},
		QNextStep:         {choice: "escalate_human"},
		QAssigneeKind:     {choice: "team:payments"},
	})
	got, err := ParseGeneric(st, ask, ds)
	if err != nil {
		t.Fatal(err)
	}
	if !got.WorthAgentBudget.Worth || got.NextStep.Step != NextEscalateHuman ||
		got.Assignee.Kind != AssigneeTeam || got.Assignee.Team != "payments" {
		t.Fatalf("unexpected %+v", got)
	}

	ds[QAssigneeKind] = decisions.Decision{Type: decisions.TypeChoice, Value: "code_owner"}
	if got, _ = ParseGeneric(st, ask, ds); got.Assignee.Kind != AssigneeCodeOwner {
		t.Fatalf("got %+v", got.Assignee)
	}
	ds[QAssigneeKind] = decisions.Decision{Type: decisions.TypeChoice, Value: "team:other"}
	if _, err := ParseGeneric(st, ask, ds); !errors.Is(err, ErrInvalidDecision) {
		t.Fatalf("unknown team: %v", err)
	}
	delete(ds, QNextStep)
	if _, err := ParseGeneric(st, ask, ds); !errors.Is(err, ErrMissingDecision) {
		t.Fatalf("got %v", err)
	}
	// not asked means not required
	if got, err = ParseGeneric(st, GenericAsk{WorthAgentBudget: true}, ds); err != nil || got.NextStep != nil {
		t.Fatalf("subset: %v %+v", err, got)
	}
}

func TestGenericInputErrors(t *testing.T) {
	st := genericState()
	st.Attempts = nil
	if _, err := GenericQuestions(st, GenericAsk{NextStep: true}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("next_step without attempts: %v", err)
	}
	if _, err := GenericQuestions(st, GenericAsk{}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("empty ask: %v", err)
	}
	st.Teams = []string{"a", "a"}
	if _, err := GenericQuestions(st, GenericAsk{AssigneeKind: true}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("dup team: %v", err)
	}
}

func TestMergeBatches(t *testing.T) {
	g, _ := GenericQuestions(genericState(), GenericAsk{WorthAgentBudget: true})
	all, err := Merge(ShieldQuestions(), g)
	if err != nil || len(all) != 4 {
		t.Fatalf("merge: %v %d", err, len(all))
	}
	if _, err := Merge(ShieldQuestions(), ShieldQuestions()); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("dup: %v", err)
	}
}

func size(t *testing.T, v any) int {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return len(b)
}

func TestSizeGuardInsights(t *testing.T) {
	big := strings.Repeat("é", 5000)
	frames := make([]string, 50)
	for i := range frames {
		frames[i] = strings.Repeat("f", 400)
	}
	st := insightsState()
	st.Error.Message, st.Error.Frames, st.Error.Code = big, frames, big
	for i := range st.Candidates {
		st.Candidates[i].Message, st.Candidates[i].Frames = big, frames
	}
	for i := 0; i < 10; i++ {
		st.SuspectCommits = append(st.SuspectCommits, SuspectCommit{SHA: "ffffffff", Subject: big, Reason: big})
	}
	if err := st.Fit(Limits{}); err != nil {
		t.Fatal(err)
	}
	if n := size(t, st); n > DefaultMaxStateBytes {
		t.Fatalf("size %d over budget", n)
	}
	if st.Error.Code != "" {
		t.Fatal("code must be dropped by default")
	}
	if len(st.Error.Frames) > defaultMaxFrames || len(st.SuspectCommits) > MaxSuspectCommits {
		t.Fatalf("not capped: %d frames %d commits", len(st.Error.Frames), len(st.SuspectCommits))
	}

	// a small budget forces progressive shrinking
	st2 := insightsState()
	st2.Error.Message, st2.Error.Frames = big, frames
	if err := st2.Fit(Limits{MaxStateBytes: 1500}); err != nil {
		t.Fatal(err)
	}
	if n := size(t, st2); n > 1500 {
		t.Fatalf("size %d over 1500", n)
	}

	// an impossible budget is an error, not a silent oversize state
	if err := insightsState().Fit(Limits{MaxStateBytes: 50}); !errors.Is(err, ErrStateTooLarge) {
		t.Fatalf("got %v", err)
	}
}

func TestIncludeCodeFlag(t *testing.T) {
	code := strings.Repeat("x", 10000)
	st := &ShieldState{Kind: "vulnerability", Code: code}
	if err := st.Fit(Limits{}); err != nil || st.Code != "" {
		t.Fatalf("default must drop code: %v %d", err, len(st.Code))
	}
	st = &ShieldState{Kind: "vulnerability", Code: code}
	if err := st.Fit(Limits{IncludeCode: true, MaxCodeBytes: 500}); err != nil {
		t.Fatal(err)
	}
	if n := len([]rune(st.Code)); n == 0 || n > 500 {
		t.Fatalf("code len %d", n)
	}
	// with a tight budget the code goes first
	st = &ShieldState{Kind: "vulnerability", Summary: "short", Code: code}
	if err := st.Fit(Limits{IncludeCode: true, MaxCodeBytes: 500, MaxStateBytes: 300}); err != nil || st.Code != "" {
		t.Fatalf("tight budget: %v %d", err, len(st.Code))
	}
}

func TestSizeGuardShieldAndGeneric(t *testing.T) {
	big := strings.Repeat("a", 3000)
	sites := make([]string, 40)
	for i := range sites {
		sites[i] = big
	}
	sh := &ShieldState{Kind: "vulnerability", Summary: big, CallSites: sites, Manifest: big, DiffStats: big}
	if err := sh.Fit(Limits{MaxStateBytes: 2000}); err != nil || size(t, sh) > 2000 {
		t.Fatalf("shield: %v %d", err, size(t, sh))
	}

	g := genericState()
	g.Task.Title = big
	g.Attempts = nil
	for i := 0; i < 30; i++ {
		g.Attempts = append(g.Attempts, Attempt{Model: "m", Outcome: "failed", Error: big})
	}
	g.Attempts[29].Model = "newest"
	if err := g.Fit(Limits{MaxStateBytes: 1500}); err != nil || size(t, g) > 1500 {
		t.Fatalf("generic: %v %d", err, size(t, g))
	}
	if len(g.Attempts) == 0 || g.Attempts[len(g.Attempts)-1].Model != "newest" {
		t.Fatal("newest attempt must be kept")
	}
}

func TestTrunc(t *testing.T) {
	if got := trunc("héllo wörld", 5); got != "héll…" {
		t.Fatalf("got %q", got)
	}
	if got := trunc("abc", 5); got != "abc" {
		t.Fatalf("got %q", got)
	}
}
