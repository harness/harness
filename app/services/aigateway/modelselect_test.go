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
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"testing"

	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeModelStore struct {
	mu     sync.Mutex
	models map[string]*types.AIModel
}

func newFakeModelStore() *fakeModelStore { return &fakeModelStore{models: map[string]*types.AIModel{}} }

func (f *fakeModelStore) Upsert(_ context.Context, m *types.AIModel) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	c := *m
	f.models[m.ID] = &c
	return nil
}

func (f *fakeModelStore) Find(_ context.Context, id string) (*types.AIModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.models[id]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	c := *m
	rates := map[string]float64{}
	for k, v := range m.SuccessRateByTask {
		rates[k] = v
	}
	c.SuccessRateByTask = rates
	return &c, nil
}

func (f *fakeModelStore) List(_ context.Context, filter types.AIModelFilter) ([]*types.AIModel, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []*types.AIModel
	for _, m := range f.models {
		if filter.Status != "" && m.Status != filter.Status {
			continue
		}
		if filter.Kind != "" && m.Kind != filter.Kind {
			continue
		}
		c := *m
		out = append(out, &c)
	}
	return out, nil
}

func (f *fakeModelStore) UpdateStatus(_ context.Context, id string, st enum.AIModelStatus, reason string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.models[id]
	if !ok {
		return gitnessstore.ErrResourceNotFound
	}
	m.Status, m.QuarantinedReason = st, reason
	return nil
}

func (f *fakeModelStore) UpdateLatency(_ context.Context, id string, p95 int) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	m, ok := f.models[id]
	if !ok {
		return gitnessstore.ErrResourceNotFound
	}
	m.P95LatencyMS = p95
	return nil
}

type fakeDecisionStore struct {
	mu   sync.Mutex
	rows []*types.AIDecision
}

func (f *fakeDecisionStore) Create(_ context.Context, d *types.AIDecision) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d.ID = int64(len(f.rows) + 1)
	c := *d
	f.rows = append(f.rows, &c)
	return nil
}

func (f *fakeDecisionStore) Find(_ context.Context, id int64) (*types.AIDecision, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ID == id {
			return r, nil
		}
	}
	return nil, gitnessstore.ErrResourceNotFound
}

func (f *fakeDecisionStore) ListByTask(context.Context, int64) ([]*types.AIDecision, error) {
	return nil, nil
}

func (f *fakeDecisionStore) SetOutcome(_ context.Context, id int64, o enum.AIDecisionOutcome) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ID == id {
			r.Outcome = &o
			return nil
		}
	}
	return gitnessstore.ErrResourceNotFound
}

func choiceAnswer(model, choice string, confidence float64) func(w http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": model + "-build",
			"answers": map[string]any{
				QuestionModelForTask: map[string]any{
					"type": "choice", "choice": choice, "confidence": confidence,
					"probabilities": map[string]float64{choice: confidence},
				},
			},
			"usage": map[string]any{"cost": 0},
		})
	}
}

// newSelectGateway builds a gateway whose catalog was synced with the stores attached.
func newSelectGateway(t *testing.T, f *fakeOpenRouter, ms *fakeModelStore, ds *fakeDecisionStore) *Gateway {
	t.Helper()
	g := newTestGateway(t, f, 100)
	g.SetStores(context.Background(), ms, ds)
	if _, err := g.Catalog().Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestSyncPersistsModelsAndRetiresMissing(t *testing.T) {
	ms := newFakeModelStore()
	ms.models["vendor/gone"] = &types.AIModel{ID: "vendor/gone", Kind: enum.AIModelKindChat, Status: enum.AIModelStatusActive}
	g := newSelectGateway(t, &fakeOpenRouter{}, ms, &fakeDecisionStore{})

	coder := ms.models["vendor/coder:free"]
	if coder == nil || coder.Kind != enum.AIModelKindChat || !coder.SupportsTools || coder.LastSeen == 0 {
		t.Fatalf("unexpected coder row: %+v", coder)
	}
	if m := ms.models[g.config.DecisionModel]; m == nil || m.Kind != enum.AIModelKindDecision {
		t.Fatalf("decision model row: %+v", m)
	}
	if ms.models["vendor/gone"].Status != enum.AIModelStatusRetired {
		t.Fatal("missing model must be retired")
	}
	if _, ok := ms.models["vendor/paid"]; ok {
		t.Fatal("paid model must not be stored")
	}
}

func TestQuarantineIsPersistedAndRestored(t *testing.T) {
	ms := newFakeModelStore()
	g := newSelectGateway(t, &fakeOpenRouter{}, ms, &fakeDecisionStore{})

	g.Guard().Quarantine("vendor/coder:free", "reported cost 1 USD")
	row := ms.models["vendor/coder:free"]
	if row.Status != enum.AIModelStatusQuarantined || row.QuarantinedReason != "reported cost 1 USD" {
		t.Fatalf("quarantine not persisted: %+v", row)
	}

	// a resync keeps it, a new process restores it
	if _, err := g.Catalog().Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	if ms.models["vendor/coder:free"].Status != enum.AIModelStatusQuarantined {
		t.Fatal("resync must keep the quarantine")
	}
	g2 := newTestGateway(t, &fakeOpenRouter{}, 100)
	g2.SetStores(context.Background(), ms, nil)
	if !g2.Guard().IsQuarantined("vendor/coder:free") {
		t.Fatal("quarantine must survive a restart")
	}
}

func TestSelectModelNoCandidatesAndSingle(t *testing.T) {
	g := newSelectGateway(t, &fakeOpenRouter{}, newFakeModelStore(), &fakeDecisionStore{})

	if _, err := g.SelectModel(context.Background(), SelectRequest{
		Kind: "fix_error", Requirements: Requirements{MinContext: 1 << 30},
	}); !errors.Is(err, ErrNoCandidates) {
		t.Fatalf("got %v, want ErrNoCandidates", err)
	}

	f := &fakeOpenRouter{}
	g = newSelectGateway(t, f, newFakeModelStore(), &fakeDecisionStore{})
	res, err := g.SelectModel(context.Background(), SelectRequest{
		Kind: "fix_error", Requirements: Requirements{Tools: true},
	})
	if err != nil || res.Model != "vendor/coder:free" || res.Source != ModelSourceSingle {
		t.Fatalf("got %+v, %v", res, err)
	}
	if f.count("/api/alpha/decisions") != 0 {
		t.Fatal("a single candidate must not spend quota")
	}
}

func TestSelectModelUsesDecisionAndRecordsOutcome(t *testing.T) {
	ms, ds := newFakeModelStore(), &fakeDecisionStore{}
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: choiceAnswer("m", "vendor/small-free", 0.8),
	}}
	g := newSelectGateway(t, f, ms, ds)

	task := int64(7)
	res, err := g.SelectModel(context.Background(), SelectRequest{
		Kind: "fix_error", TaskID: &task, SessionID: "s1",
		Task: TaskDescription{Language: "go", Attempts: []Attempt{{Model: "vendor/coder:free", Outcome: "failure"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "vendor/small-free" || res.Source != ModelSourceDecision || res.DecisionID != 1 {
		t.Fatalf("unexpected result %+v", res)
	}

	q := f.last()["questions"].(map[string]any)[QuestionModelForTask].(map[string]any)
	opts := q["criteria"].(map[string]any)
	if len(opts) != 2 || opts["vendor/coder:free"] == nil {
		t.Fatalf("criteria must describe both candidates: %v", opts)
	}

	row := ds.rows[0]
	if row.QuestionID != QuestionModelForTask || row.ModelBuild != "m-build" || row.StateHash == "" ||
		row.TaskID == nil || *row.TaskID != 7 || row.Fallback {
		t.Fatalf("unexpected decision row %+v", row)
	}

	if err := g.RecordOutcome(context.Background(), res.DecisionID, res.Model, "fix_error", true); err != nil {
		t.Fatal(err)
	}
	if err := g.RecordOutcome(context.Background(), 0, res.Model, "fix_error", false); err != nil {
		t.Fatal(err)
	}
	if ds.rows[0].Outcome == nil || *ds.rows[0].Outcome != enum.AIDecisionOutcomeSuccess {
		t.Fatal("decision outcome not set")
	}
	rates := ms.models["vendor/small-free"].SuccessRateByTask
	if rates["fix_error"] != 0.5 || rates["fix_error#n"] != 2 {
		t.Fatalf("unexpected rates %v", rates)
	}
}

func TestSelectModelLowConfidenceUsesHistory(t *testing.T) {
	ms, ds := newFakeModelStore(), &fakeDecisionStore{}
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: choiceAnswer("m", "vendor/small-free", 0.4),
	}}
	g := newSelectGateway(t, f, ms, ds)
	ms.models["vendor/coder:free"].SuccessRateByTask = map[string]float64{"fix_error": 0.9, "fix_error#n": 10}
	ms.models["vendor/small-free"].SuccessRateByTask = map[string]float64{"fix_error": 0.3, "fix_error#n": 10}

	res, err := g.SelectModel(context.Background(), SelectRequest{Kind: "fix_error"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Model != "vendor/coder:free" || res.Source != ModelSourceHistory || res.Confidence != 0.4 {
		t.Fatalf("unexpected result %+v", res)
	}
	if len(ds.rows) != 1 {
		t.Fatal("decision must be recorded")
	}
}

func TestSelectModelQuotaExhaustedFallsBack(t *testing.T) {
	ms, ds := newFakeModelStore(), &fakeDecisionStore{}
	f := &fakeOpenRouter{decisions: map[string]func(http.ResponseWriter){
		DefaultDecisionModel: choiceAnswer("m", "vendor/small-free", 0.9),
	}}
	g := newSelectGateway(t, f, ms, ds)
	g.planner = NewPlanner(1, 18, nil)
	// spend the whole planner so the selection hits a quota error
	_, _ = g.Decide(context.Background(), "", "x", fixQuestion)
	_, _ = g.Decide(context.Background(), "", "y", fixQuestion)

	// no history: tie on rate and latency, lowest id wins
	res, err := g.SelectModel(context.Background(), SelectRequest{Kind: "explain"})
	if err != nil {
		t.Fatal(err)
	}
	if !res.QuotaFallback || res.Source != ModelSourceHistory || res.Model != "vendor/coder:free" {
		t.Fatalf("unexpected result %+v", res)
	}
}

func TestDeterministicPickTieBreaks(t *testing.T) {
	g := &Gateway{}
	pick := func(s ...candidateStats) string { return g.deterministicPick(s).Model }
	mk := func(id string, rate float64, p95 int) candidateStats {
		return candidateStats{model: Model{ID: id}, rate: rate, p95: p95}
	}
	if got := pick(mk("b", 0.5, 100), mk("a", 0.5, 200)); got != "b" {
		t.Fatalf("lowest p95 expected, got %s", got)
	}
	if got := pick(mk("b", 0.5, 100), mk("a", 0.5, 100)); got != "a" {
		t.Fatalf("lowest id expected, got %s", got)
	}
	if got := pick(mk("a", -1, 0), mk("b", 0, 0)); got != "b" {
		t.Fatalf("observed beats unobserved, got %s", got)
	}
}
