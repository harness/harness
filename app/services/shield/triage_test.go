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

package shield

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/shield/risk"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type triageFindings struct{ *fakeFindings }

func (f triageFindings) UpdateTriage(_ context.Context, id int64, in *types.ShieldFindingTriage, _ int64) error {
	for _, r := range f.rows {
		if r.ID == id {
			r.RiskScore, r.Reachable, r.AIDecisionIDs = in.RiskScore, in.Reachable, in.AIDecisionIDs
			if in.FixStrategy != nil {
				r.FixStrategy = in.FixStrategy
			}
		}
	}
	return nil
}

type triageTasks struct {
	*fakeTasks
	priorities map[int64]enum.TaskPriority
}

func (f triageTasks) Update(_ context.Context, _, id int64, in task.UpdateInput) (*types.Task, error) {
	if in.Priority != nil {
		f.priorities[id] = *in.Priority
	}
	return nil, nil
}

type fakeDecide struct {
	reach, fix string
	reachConf  float64
	fp         string
	fpConf     float64
	err        error // returned from call number errAt on (1-based)
	errAt      int
	calls      int
	states     []any
	rules      bool
}

func (d *fakeDecide) Decide(_ context.Context, in aidecide.DecideInput) (*aidecide.Result, error) {
	d.calls++
	d.states = append(d.states, in.State)
	if d.err != nil && d.calls >= d.errAt {
		return nil, d.err
	}
	ids := map[string]int64{questions.QReachable: 10, questions.QFalsePositive: 11, questions.QFixStrategy: 12}
	if d.rules {
		return &aidecide.Result{Decisions: in.Fallback(in.Questions), Rules: true, Fallback: true, DecisionIDs: ids}, nil
	}
	return &aidecide.Result{
		Decisions: map[string]decisions.Decision{
			questions.QReachable: {QuestionID: questions.QReachable, Type: decisions.TypeChoice,
				Value: d.reach, Confidence: d.reachConf},
			questions.QFalsePositive: {QuestionID: questions.QFalsePositive, Type: decisions.TypeNoul,
				Value: d.fp, Confidence: d.fpConf},
			questions.QFixStrategy: {QuestionID: questions.QFixStrategy, Type: decisions.TypeChoice,
				Value: d.fix, Confidence: 0.9},
		},
		DecisionIDs: ids,
	}, nil
}

type fakeModes struct{ mode enum.AIMode }

func (m fakeModes) Resolve(context.Context, *types.RepositoryCore) (enum.AIMode, error) {
	return m.mode, nil
}

type triageEnv struct {
	tr    *Triager
	f     *fakeFindings
	tasks triageTasks
	scan  *types.ShieldScan
	repo  *types.RepositoryCore
}

func newTriageEnv(
	t *testing.T,
	mode enum.AIMode,
	autonomy int,
	dec *fakeDecide,
	rows ...*types.ShieldFinding,
) *triageEnv {
	t.Helper()
	e := newEnv(t, 70, true)
	e.git.files = map[string]string{
		"src/a.go":      "package a\n\nimport \"github.com/x/lodash/v2\"\n",
		"src/b.go":      "package b\n\nimport (\n\t\"github.com/x/lodash/v2/sub\"\n)\n",
		"go.mod":        "module m\n",
		"web/app.js":    "const l = require('left-pad')\n",
		"web/other.txt": "x",
	}
	ff := &fakeFindings{}
	for i, r := range rows {
		r.ID = int64(i + 1)
		r.RepoID = e.repo.ID
		r.State = enum.ShieldFindingStateOpen
		r.LastSeenScanID = 1
		r.Kind = enum.ShieldFindingKindDependency
		ff.rows = append(ff.rows, r)
	}
	e.svc.findings = ff
	tasks := triageTasks{fakeTasks: e.tasks, priorities: map[int64]enum.TaskPriority{}}
	tr := NewTriager(TriageConfig{}, e.svc, triageFindings{ff}, tasks, dec, fakeModes{mode},
		func(context.Context, int64) (int, error) { return autonomy, nil })
	return &triageEnv{tr: tr, f: ff, tasks: tasks, scan: &types.ShieldScan{ID: 1, CommitSHA: "abc"}, repo: e.repo}
}

func vulnFinding(pkg string, cvss float64, riskScore int) *types.ShieldFinding {
	return &types.ShieldFinding{
		Ecosystem: "Go", Package: pkg, InstalledVersion: "1.2.0", VulnID: "CVE-2024-" + pkg, Fingerprint: pkg,
		CVSSScore: cvss, RiskScore: riskScore, MinFixedVersion: "1.2.5", Reachable: enum.ShieldReachabilityUnknown,
	}
}

func TestTriageRescoreNotReachable(t *testing.T) {
	dec := &fakeDecide{reach: "not_reachable", reachConf: 0.9, fp: "false", fpConf: 0.9, fix: "deterministic_bump"}
	e := newTriageEnv(t, enum.AIModeFull, 1, dec, vulnFinding("lodash", 9.0, 90))
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	f := e.f.rows[0]
	if f.Reachable != enum.ShieldReachabilityNotReachable || f.RiskScore >= 90 {
		t.Fatalf("expected lowered risk, got %+v", f)
	}
	if f.FixStrategy == nil || *f.FixStrategy != enum.ShieldFixStrategyBump {
		t.Fatalf("fix strategy = %v", f.FixStrategy)
	}
	if len(f.AIDecisionIDs) != 3 {
		t.Fatalf("decision ids = %v", f.AIDecisionIDs)
	}
}

func TestTriageLowConfidenceNotReachableKeepsRisk(t *testing.T) {
	dec := &fakeDecide{reach: "not_reachable", reachConf: 0.5, fp: "false", fpConf: 0.9, fix: "manual"}
	e := newTriageEnv(t, enum.AIModeFull, 1, dec, vulnFinding("lodash", 9.0, 90))
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	f := e.f.rows[0]
	if f.Reachable != enum.ShieldReachabilityUnknown || f.RiskScore != 90 {
		t.Fatalf("expected unchanged risk, got %+v", f)
	}
}

func TestTriageReachableRaisesRiskAndPriority(t *testing.T) {
	dec := &fakeDecide{reach: "reachable", reachConf: 0.9, fp: "false", fpConf: 0.9, fix: "agent"}
	f := vulnFinding("lodash", 6.0, 60) // deterministic risk 60 -> P2
	tid := int64(7)
	f.TaskID = &tid
	e := newTriageEnv(t, enum.AIModeFull, 1, dec, f)
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	got := e.f.rows[0]
	if got.Reachable != enum.ShieldReachabilityReachable || got.RiskScore <= 60 {
		t.Fatalf("expected raised risk, got %+v", got)
	}
	if want := PriorityForRisk(got.RiskScore); want != PriorityForRisk(60) && e.tasks.priorities[tid] != want {
		t.Fatalf("priority = %q, want %q", e.tasks.priorities[tid], want)
	}
}

func TestTriageFalsePositiveRespectsAutonomy(t *testing.T) {
	for _, tc := range []struct {
		autonomy int
		fpConf   float64
		ignored  bool
	}{{0, 0.95, false}, {1, 0.95, true}, {1, 0.6, false}} {
		dec := &fakeDecide{reach: "unknown", reachConf: 0.9, fp: "true", fpConf: tc.fpConf, fix: "manual"}
		e := newTriageEnv(t, enum.AIModeFull, tc.autonomy, dec, vulnFinding("lodash", 8.0, 80))
		if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
			t.Fatal(err)
		}
		f := e.f.rows[0]
		if (f.State == enum.ShieldFindingStateIgnored) != tc.ignored {
			t.Fatalf("autonomy %d conf %.2f: state = %s", tc.autonomy, tc.fpConf, f.State)
		}
		if tc.ignored && f.StateReason != "false positive (AI, confidence 0.95)" {
			t.Fatalf("reason = %q", f.StateReason)
		}
		if len(f.AIDecisionIDs) == 0 {
			t.Fatal("decisions must be recorded in every case")
		}
	}
}

func TestTriageQuotaStopsAndLeavesRest(t *testing.T) {
	dec := &fakeDecide{reach: "unknown", reachConf: 0.9, fp: "false", fpConf: 0.9, fix: "manual",
		err: &aigateway.QuotaError{Daily: true}, errAt: 2}
	e := newTriageEnv(t, enum.AIModeFull, 1, dec,
		vulnFinding("low", 5.0, 50), vulnFinding("high", 9.0, 90), vulnFinding("mid", 7.0, 70))
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	if dec.calls != 2 {
		t.Fatalf("calls = %d, want 2 (stop at the first quota error)", dec.calls)
	}
	for _, f := range e.f.rows {
		if triaged := len(f.AIDecisionIDs) > 0; (f.Package == "high") != triaged {
			t.Fatalf("%s triaged = %t", f.Package, triaged)
		}
	}
	// the next scan, with quota available, picks up the rest
	dec.err = nil
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	for _, f := range e.f.rows {
		if len(f.AIDecisionIDs) == 0 {
			t.Fatalf("%s still untriaged", f.Package)
		}
	}
	if dec.calls != 4 { // 2 earlier + mid + low; high is not asked again
		t.Fatalf("calls = %d, want 4", dec.calls)
	}
}

func TestTriageBelowThresholdSkipped(t *testing.T) {
	dec := &fakeDecide{reach: "unknown", fp: "false", fix: "manual"}
	e := newTriageEnv(t, enum.AIModeFull, 1, dec, vulnFinding("lodash", 3.0, 30))
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	if dec.calls != 0 {
		t.Fatal("finding below threshold must not be triaged")
	}
}

func TestTriageRulesFallback(t *testing.T) {
	tests := []struct {
		name      string
		installed string
		fixed     string
		risk      int
		want      enum.ShieldFixStrategy
	}{
		{"same line fix", "1.2.0", "1.2.5", 50, enum.ShieldFixStrategyBump},
		{"major bump high risk", "1.2.0", "2.0.0", 80, enum.ShieldFixStrategyPatch},
		{"major bump low risk", "1.2.0", "2.0.0", 50, enum.ShieldFixStrategyManual},
		{"no fix high risk", "1.2.0", "", 75, enum.ShieldFixStrategyPatch},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := vulnFinding("lodash", 8.0, tt.risk)
			f.InstalledVersion, f.MinFixedVersion = tt.installed, tt.fixed
			dec := &fakeDecide{rules: true}
			e := newTriageEnv(t, enum.AIModeOff, 3, dec, f)
			if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
				t.Fatal(err)
			}
			got := e.f.rows[0]
			if got.State != enum.ShieldFindingStateOpen || got.Reachable != enum.ShieldReachabilityUnknown {
				t.Fatalf("rules must not ignore or set reachability: %+v", got)
			}
			if got.FixStrategy == nil || *got.FixStrategy != tt.want {
				t.Fatalf("strategy = %v, want %s", got.FixStrategy, tt.want)
			}
		})
	}
}

func TestTriageStateByMode(t *testing.T) {
	dec := &fakeDecide{reach: "unknown", reachConf: 0.9, fp: "false", fpConf: 0.9, fix: "manual"}
	e := newTriageEnv(t, enum.AIModeDecisionsOnly, 1, dec, vulnFinding("github.com/x/lodash", 8.0, 80))
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	st := dec.states[0].(*questions.ShieldState)
	if st.ImportCount != 2 {
		t.Fatalf("import count = %d, want 2", st.ImportCount)
	}
	if len(st.CallSites) != 0 || st.Code != "" {
		t.Fatalf("decisions-only state must carry no code: %+v", st)
	}
	raw, _ := json.Marshal(st)
	if strings.Contains(string(raw), "import (") || strings.Contains(string(raw), "lodash/v2") {
		t.Fatalf("state leaks code: %s", raw)
	}

	dec2 := &fakeDecide{reach: "unknown", reachConf: 0.9, fp: "false", fpConf: 0.9, fix: "manual"}
	e2 := newTriageEnv(t, enum.AIModeFull, 1, dec2, vulnFinding("github.com/x/lodash", 8.0, 80))
	if err := e2.tr.Triage(context.Background(), e2.repo, e2.scan); err != nil {
		t.Fatal(err)
	}
	st2 := dec2.states[0].(*questions.ShieldState)
	if len(st2.CallSites) != 2 || len(st2.CallSites) > maxImportLines {
		t.Fatalf("full mode call sites = %v", st2.CallSites)
	}
	if st2.Code != "" {
		t.Fatal("no code excerpt expected")
	}
}

func TestTriageReapplyKeepsReachability(t *testing.T) {
	f := vulnFinding("lodash", 9.0, 90) // a rescan reset the risk to the deterministic value
	f.Reachable = enum.ShieldReachabilityNotReachable
	f.AIDecisionIDs = []int64{1, 2, 3}
	dec := &fakeDecide{}
	e := newTriageEnv(t, enum.AIModeFull, 1, dec, f)
	if err := e.tr.Triage(context.Background(), e.repo, e.scan); err != nil {
		t.Fatal(err)
	}
	if dec.calls != 0 {
		t.Fatal("triaged finding must not be asked again")
	}
	want := rescore(&risk.Scorer{}, e.f.rows[0], risk.ExposurePublic, risk.ReachabilityNotReachable, 1)
	if e.f.rows[0].RiskScore != want || want >= 90 {
		t.Fatalf("risk = %d, want %d", e.f.rows[0].RiskScore, want)
	}
}
