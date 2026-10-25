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

package eval

import (
	"bytes"
	"context"
	"errors"
	"math"
	"sort"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// fakeDecider answers every question with the label of the case (when the case has one) or with
// the first option, at a fixed confidence. flip makes it answer wrongly on purpose.
type fakeDecider struct {
	expected   map[string]map[string]string
	confidence float64
	flip       bool
	failFor    string
}

func (f *fakeDecider) Decide(
	_ context.Context, caseID string, _ any, qs map[string]decisions.Question,
) (*Result, error) {
	if caseID == f.failFor {
		return nil, errors.New("boom")
	}
	res := &Result{Model: "fake-model@1", Decisions: map[string]decisions.Decision{}}
	for id, q := range qs {
		want, ok := f.expected[caseID][id]
		d := decisions.Decision{QuestionID: id, Type: q.Type, Confidence: f.confidence}
		switch c := q.Criteria.(type) {
		case map[string]string:
			keys := make([]string, 0, len(c))
			for k := range c {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			setValue(&d, keys, want, ok, f.flip)
		case []string:
			setValue(&d, c, want, ok, f.flip)
		}
		if q.Type == decisions.TypeNoul {
			setValue(&d, []string{"false", "true"}, want, ok, f.flip)
			d.Probability = 1 - f.confidence
			if d.Value == "true" {
				d.Probability = f.confidence
			}
		}
		res.Decisions[id] = d
	}
	return res, nil
}

func setValue(d *decisions.Decision, options []string, want string, ok, flip bool) {
	d.Value = options[0]
	if ok {
		d.Value = want
		if flip {
			d.Value = otherKey(options, want)
		}
	}
}

func otherKey(keys []string, not string) string {
	for _, k := range keys {
		if k != not {
			return k
		}
	}
	return not
}

func oracle(cases []Case, conf float64) *fakeDecider {
	f := &fakeDecider{expected: map[string]map[string]string{}, confidence: conf}
	for _, c := range cases {
		f.expected[c.ID] = c.Expected
	}
	return f
}

func TestDatasetLoadsAndBuilds(t *testing.T) {
	cases, err := LoadCasesFile("testdata/cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 40 {
		t.Fatalf("want at least 40 cases, got %d", len(cases))
	}
	perGroup, unreviewed := DatasetStats(cases)
	for _, g := range []Group{GroupShield, GroupInsights, GroupGeneric} {
		if perGroup[g] == 0 {
			t.Errorf("no cases for group %s", g)
		}
	}
	if unreviewed != len(cases) {
		t.Errorf("all drafted cases must be marked reviewed=false, %d of %d are", unreviewed, len(cases))
	}
	for _, c := range cases {
		if !strings.HasPrefix(c.Notes, "DRAFT") || c.Source == "" {
			t.Errorf("case %s: notes must start with DRAFT and source must be set", c.ID)
		}
	}
}

func TestOracleRunIsPerfect(t *testing.T) {
	cases, err := LoadCasesFile("testdata/cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Run(context.Background(), cases, oracle(cases, 0.9), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) != 0 {
		t.Fatalf("unexpected case errors: %+v", rep.Errors)
	}
	if rep.Dataset.Evaluated != len(cases) || rep.Overall.Accuracy != 1 {
		t.Fatalf("want perfect run, got %+v accuracy %v", rep.Dataset, rep.Overall.Accuracy)
	}
	if rep.Model != "fake-model@1" {
		t.Errorf("model = %q", rep.Model)
	}
	if rep.Overall.Brier == nil || math.Abs(*rep.Overall.Brier-0.01) > 1e-9 {
		t.Errorf("brier = %v, want 0.01 (p=0.9 always right)", rep.Overall.Brier)
	}
	if !rep.Overall.Threshold.Found || rep.Overall.Threshold.Confidence != 0.9 {
		t.Errorf("threshold = %+v", rep.Overall.Threshold)
	}
	// Confidence 0.9 sits in the last bucket and is right 100% of the time: under-confident by 0.1.
	last := rep.Calibrate[len(rep.Calibrate)-1]
	if last.Label != "0.9-1.0" || last.N != rep.Overall.N || last.Accuracy != 1 {
		t.Errorf("last bucket = %+v", last)
	}
}

func TestWrongAnswersAndCalibration(t *testing.T) {
	cases, err := LoadCasesFile("testdata/tiny_cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	f := oracle(cases, 0.95)
	f.flip = true
	rep, err := Run(context.Background(), cases, f, Options{Groups: []Group{GroupShield}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Dataset.Cases != 2 || rep.Overall.Correct != 0 || rep.Overall.Accuracy != 0 {
		t.Fatalf("unexpected report %+v %+v", rep.Dataset, rep.Overall)
	}
	if math.Abs(rep.Overall.ECE-0.95) > 1e-9 {
		t.Errorf("ece = %v, want 0.95", rep.Overall.ECE)
	}
	if rep.Overall.Threshold.Found {
		t.Errorf("a model that is always wrong must get no threshold: %+v", rep.Overall.Threshold)
	}
}

func TestReplayTinyFile(t *testing.T) {
	cases, err := LoadCasesFile("testdata/tiny_cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewReplayDeciderFile("testdata/tiny_replay.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Run(context.Background(), cases, d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Dataset.Cases != 4 || rep.Dataset.Evaluated != 3 || rep.Dataset.Skipped != 1 || len(rep.Errors) != 0 {
		t.Fatalf("dataset = %+v errors = %+v", rep.Dataset, rep.Errors)
	}
	if rep.Overall.N != 8 || rep.Overall.Correct != 7 {
		t.Fatalf("overall = %+v", rep.Overall)
	}
	if rep.Overall.Brier == nil || math.Abs(*rep.Overall.Brier-0.02125) > 1e-9 {
		t.Errorf("brier = %v", rep.Overall.Brier)
	}
	want := Threshold{Found: true, Confidence: 0.7, N: 7, Accuracy: 1}
	if rep.Overall.Threshold != want {
		t.Errorf("threshold = %+v, want %+v", rep.Overall.Threshold, want)
	}
	for _, q := range rep.Questions {
		if q.Question == "reachable" && (q.N != 2 || q.Correct != 1 || q.Suggestion.Found) {
			t.Errorf("reachable stats = %+v", q)
		}
	}
	if rep.Model != "baseten/mammoth-1:free@test" {
		t.Errorf("model = %q", rep.Model)
	}

	var md, js bytes.Buffer
	if err := rep.WriteMarkdown(&md, nil); err != nil {
		t.Fatal(err)
	}
	for _, s := range []string{"Model build: baseten/mammoth-1:free@test", "DRAFT labels", "| 0.9-1.0 |", "## Wrong answers", "shield-lodash-dev-only"} {
		if !strings.Contains(md.String(), s) {
			t.Errorf("markdown lacks %q:\n%s", s, md.String())
		}
	}
	if err := rep.WriteJSON(&js); err != nil || !strings.Contains(js.String(), `"suggested_threshold"`) {
		t.Errorf("json summary: %v %s", err, js.String())
	}
}

func TestRecordThenReplayRoundTrip(t *testing.T) {
	cases, err := LoadCasesFile("testdata/tiny_cases.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	live, err := Run(context.Background(), cases, NewRecorder(oracle(cases, 0.8), &buf), Options{})
	if err != nil {
		t.Fatal(err)
	}
	d, err := NewReplayDecider(&buf)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := Run(context.Background(), cases, d, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(live.Outcomes) == 0 || len(Compare(live, replayed)) != 0 {
		t.Fatalf("replay differs from the live run")
	}
	if live.Overall.Accuracy != replayed.Overall.Accuracy || live.Model != replayed.Model {
		t.Errorf("live %+v vs replay %+v", live.Overall, replayed.Overall)
	}
}

func TestCompareFindsRegressions(t *testing.T) {
	cases, _ := LoadCasesFile("testdata/tiny_cases.jsonl")
	good, err := Run(context.Background(), cases, oracle(cases, 0.9), Options{})
	if err != nil {
		t.Fatal(err)
	}
	f := oracle(cases, 0.9)
	f.flip = true
	bad, err := Run(context.Background(), cases, f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	reg := Compare(good, bad)
	if len(reg) != len(good.Outcomes) || reg[0].Fixed {
		t.Fatalf("regressions = %+v", reg)
	}
	if fixed := Compare(bad, good); len(fixed) != len(reg) || !fixed[0].Fixed {
		t.Fatalf("fixed = %+v", fixed)
	}
	var md bytes.Buffer
	if err := bad.WriteMarkdown(&md, good); err != nil || !strings.Contains(md.String(), "regression") {
		t.Errorf("markdown lacks regressions: %v", err)
	}
}

func TestDeciderErrorIsReportedNotFatal(t *testing.T) {
	cases, _ := LoadCasesFile("testdata/tiny_cases.jsonl")
	f := oracle(cases, 0.9)
	f.failFor = "insights-flaky-timeout"
	rep, err := Run(context.Background(), cases, f, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Errors) != 1 || rep.Errors[0].CaseID != "insights-flaky-timeout" || rep.Dataset.Evaluated != 3 {
		t.Fatalf("errors = %+v dataset = %+v", rep.Errors, rep.Dataset)
	}
}

func TestSuggestThreshold(t *testing.T) {
	mk := func(conf float64, ok bool) Outcome { return Outcome{Confidence: conf, Correct: ok} }
	outs := []Outcome{
		mk(0.99, true), mk(0.95, true), mk(0.9, true), mk(0.85, true), mk(0.8, false),
		mk(0.7, true), mk(0.6, false), mk(0.55, false),
	}
	got := SuggestThreshold(outs, 0.9, 3)
	if !got.Found || got.Confidence != 0.85 || got.N != 4 || got.Accuracy != 1 {
		t.Errorf("target 0.9: %+v", got)
	}
	// At 0.7 and above: 5 of 6 correct = 83%, which reaches 0.8 but not 0.9.
	got = SuggestThreshold(outs, 0.8, 3)
	if !got.Found || got.Confidence != 0.7 || got.N != 6 {
		t.Errorf("target 0.8: %+v", got)
	}
	if got = SuggestThreshold(outs, 0.9, 5); got.Found {
		t.Errorf("min support 5 must find nothing: %+v", got)
	}
	if got = SuggestThreshold(nil, 0.9, 1); got.Found {
		t.Errorf("empty input: %+v", got)
	}
}

func TestInvalidCases(t *testing.T) {
	for name, line := range map[string]string{
		"unknown field":   `{"id":"a","group":"shield","state":{"kind":"vulnerability","direct_dependency":true,"import_count":1},"expected":{"reachable":"reachable"},"extra":1}`,
		"unknown state":   `{"id":"a","group":"shield","state":{"nope":1},"expected":{"reachable":"reachable"}}`,
		"illegal answer":  `{"id":"a","group":"shield","state":{"kind":"vulnerability","direct_dependency":true,"import_count":1},"expected":{"reachable":"maybe"}}`,
		"unknown id":      `{"id":"a","group":"shield","state":{"kind":"vulnerability","direct_dependency":true,"import_count":1},"expected":{"severity":"low"}}`,
		"no expected":     `{"id":"a","group":"shield","state":{"kind":"vulnerability","direct_dependency":true,"import_count":1},"expected":{}}`,
		"unknown group":   `{"id":"a","group":"nope","state":{},"expected":{"x":"y"}}`,
		"next step alone": `{"id":"a","group":"generic","state":{"task":{"kind":"fix"},"quota":{}},"expected":{"next_step":"abandon"}}`,
	} {
		if _, err := LoadCases(strings.NewReader(line)); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
	dup := `{"id":"a","group":"generic","state":{"task":{"kind":"fix"},"quota":{}},"expected":{"worth_agent_budget":"true"}}`
	if _, err := LoadCases(strings.NewReader(dup + "\n" + dup)); err == nil {
		t.Error("duplicate id: want an error")
	}
}
