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

package triage

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/insights/insightstest"
	"github.com/harness/gitness/app/services/insights/suspect"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakeTasks struct {
	mu        sync.Mutex
	nextID    int64
	byRef     map[string]*types.Task
	created   []task.CreateInput
	states    map[int64][]enum.TaskState
	links     map[int64][]string
	comments  map[int64][]string
	decisions map[int64][]task.DecisionInput
}

func newFakeTasks() *fakeTasks {
	return &fakeTasks{
		nextID: 100, byRef: map[string]*types.Task{}, states: map[int64][]enum.TaskState{},
		links: map[int64][]string{}, comments: map[int64][]string{}, decisions: map[int64][]task.DecisionInput{},
	}
}

func (f *fakeTasks) Create(_ context.Context, _ int64, in task.CreateInput) (*types.Task, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if t, ok := f.byRef[in.OriginRef]; ok {
		return t, false, nil
	}
	f.nextID++
	t := &types.Task{ID: f.nextID, RepoID: in.RepoID, Title: in.Title, Priority: in.Priority, State: enum.TaskStateOpen}
	f.byRef[in.OriginRef] = t
	f.created = append(f.created, in)
	return t, true, nil
}

func (f *fakeTasks) Transition(_ context.Context, _, id int64, to enum.TaskState, _ string) (*types.Task, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.states[id] = append(f.states[id], to)
	return &types.Task{ID: id, State: to}, nil
}

func (f *fakeTasks) Link(_ context.Context, _, id int64, kind enum.TaskLinkKind, ref string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.links[id] = append(f.links[id], string(kind)+":"+ref)
	return nil
}

func (f *fakeTasks) AddComment(_ context.Context, _, id int64, text string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.comments[id] = append(f.comments[id], text)
	return nil
}

func (f *fakeTasks) RecordDecision(_ context.Context, _, id int64, in task.DecisionInput) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.decisions[id] = append(f.decisions[id], in)
	return nil
}

func (f *fakeTasks) decision(id int64, q string) *task.DecisionInput {
	for i := range f.decisions[id] {
		if f.decisions[id][i].QuestionID == q {
			return &f.decisions[id][i]
		}
	}
	return nil
}

// fakeDecider answers every question with a neutral decision unless overridden.
type fakeDecider struct {
	err       error
	overrides map[string]decisions.Decision
	human     bool
	calls     int
	lastState *questions.InsightsState
}

func (d *fakeDecider) Decide(_ context.Context, req DecideRequest) (*DecideResult, error) {
	d.calls++
	if st, ok := req.State.(*questions.InsightsState); ok {
		d.lastState = st
	}
	if d.err != nil {
		return nil, d.err
	}
	out := map[string]decisions.Decision{}
	for id, q := range req.Questions {
		if o, ok := d.overrides[id]; ok {
			o.QuestionID = id
			out[id] = o
			continue
		}
		switch q.Type {
		case decisions.TypeNoul:
			out[id] = decisions.Decision{QuestionID: id, Type: q.Type, Value: "false", Probability: 0.05, Confidence: 0.9}
		case decisions.TypeScore:
			out[id] = decisions.Decision{QuestionID: id, Type: q.Type, Value: "medium", Score: 1, Confidence: 0.9}
		default:
			v := questions.SuspectNone
			if id == questions.QAction {
				v = string(questions.ActionNotify)
			}
			out[id] = decisions.Decision{QuestionID: id, Type: q.Type, Value: v, Confidence: 0.9}
		}
	}
	return &DecideResult{Decisions: out, Model: "free/model-2026-01", Human: d.human}, nil
}

type fakeRepos struct{}

func (fakeRepos) FindByID(_ context.Context, id int64) (*types.RepositoryCore, error) {
	return &types.RepositoryCore{ID: id, GitUID: "git-uid", DefaultBranch: "main"}, nil
}

type fakeModes struct{ mode enum.AIMode }

func (m fakeModes) Resolve(context.Context, *types.RepositoryCore) (enum.AIMode, error) {
	return m.mode, nil
}

type fakeSuspects struct {
	calls []suspect.Input
}

func (s *fakeSuspects) Find(_ context.Context, _ *types.RepositoryCore, in suspect.Input) ([]suspect.Candidate, error) {
	s.calls = append(s.calls, in)
	return []suspect.Candidate{{
		Commit:    suspect.CommitRef{SHA: "abcdef0123456789", Title: "Change checkout", When: time.UnixMilli(1_000)},
		FramesHit: 2, Reasons: []suspect.Reason{{Kind: "blame"}},
	}}, nil
}

const payloadTmpl = `{"level":"error","exception":{"values":[{"type":%q,"value":"boom","stacktrace":{"frames":[` +
	`{"module":"app.main","function":"run","filename":"main.go","lineno":10,"in_app":true},` +
	`{"module":"app.pay","function":"charge","filename":"pay.go","lineno":42,"in_app":true,"context_line":"x := y.z"}]}}]}}`

type env struct {
	t        *testing.T
	st       *insightstest.Store
	tasks    *fakeTasks
	decider  *fakeDecider
	suspects *fakeSuspects
	obs      *Observer
	project  *types.InsightsProject
}

func newEnv(t *testing.T, mode enum.AIMode) *env {
	t.Helper()
	e := &env{
		t: t, st: insightstest.New(), tasks: newFakeTasks(), decider: &fakeDecider{}, suspects: &fakeSuspects{},
		project: &types.InsightsProject{ID: 5, RepoID: 1, Identifier: "api"},
	}
	e.obs = New(Config{}, Deps{
		Issues: e.st.IssueStore(), Events: e.st.EventStore(), Repos: fakeRepos{}, Modes: fakeModes{mode},
		Suspects: e.suspects, Decider: e.decider, Tasks: e.tasks, SystemPrincipal: func() int64 { return 1 },
	})
	return e
}

func (e *env) addIssue(fp, level, errType, release string, taskID *int64) (*types.InsightsIssue, *types.InsightsEvent) {
	e.t.Helper()
	issue := &types.InsightsIssue{
		ProjectID: e.project.ID, RepoID: 1, Fingerprint: fp, Title: errType + ": boom", Level: level,
		FirstRelease: release, LastRelease: release, TimesSeen: 3, UsersSeen: 1, Status: enum.InsightsIssueStatusUnresolved,
		TaskID: taskID,
	}
	if err := e.st.IssueStore().Create(context.Background(), issue); err != nil {
		e.t.Fatal(err)
	}
	ev := &types.InsightsEvent{
		IssueID: issue.ID, ProjectID: e.project.ID, EventID: fmt.Sprintf("ev-%d", issue.ID), Release: release,
		Level: level, Message: "boom", Payload: fmt.Sprintf(payloadTmpl, errType), Timestamp: 86_400_000,
	}
	if err := e.st.EventStore().Create(context.Background(), ev); err != nil {
		e.t.Fatal(err)
	}
	return issue, ev
}

func (e *env) run(issue *types.InsightsIssue, ev *types.InsightsEvent, regression bool) {
	e.t.Helper()
	if err := e.obs.process(context.Background(), job{project: e.project, issue: issue, event: ev, regression: regression}); err != nil {
		e.t.Fatal(err)
	}
}

func (e *env) storedTaskID(issue *types.InsightsIssue) int64 {
	e.t.Helper()
	got, err := e.st.IssueStore().Find(context.Background(), issue.ID)
	if err != nil {
		e.t.Fatal(err)
	}
	if got.TaskID == nil {
		e.t.Fatal("issue has no task")
	}
	return *got.TaskID
}

func highSeverity() map[string]decisions.Decision {
	return map[string]decisions.Decision{
		questions.QSeverity: {Type: decisions.TypeScore, Value: "high", Score: 2.1, Confidence: 0.85,
			Probabilities: map[string]float64{"high": 0.8}},
		questions.QAction: {Type: decisions.TypeChoice, Value: "fix", Confidence: 0.92},
	}
}

func TestCreateTask(t *testing.T) {
	e := newEnv(t, enum.AIModeFull)
	e.decider.overrides = highSeverity()
	e.decider.overrides[questions.QSuspectCommit] = decisions.Decision{
		Type: decisions.TypeChoice, Value: "abcdef01", Confidence: 0.7,
	}
	issue, ev := e.addIssue("fp1", "error", "TypeError", "1.1.0", nil)
	e.addIssue("fpOld", "error", "KeyError", "1.0.0", nil) // unrelated, supplies previous release

	e.run(issue, ev, false)

	if len(e.tasks.created) != 1 {
		t.Fatalf("created %d tasks", len(e.tasks.created))
	}
	in := e.tasks.created[0]
	if in.Priority != enum.TaskPriorityP1 || in.Origin != enum.TaskOriginInsights || in.OriginRef != "5:fp1" {
		t.Fatalf("unexpected input %+v", in)
	}
	id := e.storedTaskID(issue)
	if got := e.tasks.states[id]; len(got) != 1 || got[0] != enum.TaskStateTriage {
		t.Fatalf("states %v", got)
	}
	if got := e.tasks.links[id]; len(got) != 1 || got[0] != fmt.Sprintf("insights_issue:%d", issue.ID) {
		t.Fatalf("links %v", got)
	}
	d := e.tasks.decision(id, questions.QSeverity)
	if d == nil || d.Value != "high" || d.Model != "free/model-2026-01" || d.Fallback || d.Confidence != 0.85 {
		t.Fatalf("severity decision %+v", d)
	}
	if e.tasks.decision(id, questions.QSuspectCommit) == nil {
		t.Fatal("suspect decision missing")
	}
	if m := e.tasks.decision(id, QAutofixCandidate); m == nil || m.Value != "true" {
		t.Fatalf("autofix marker %+v", m)
	}
	if len(e.suspects.calls) != 1 || e.suspects.calls[0].ReleaseSHA != "1.1.0" || e.suspects.calls[0].PreviousSHA != "1.0.0" {
		t.Fatalf("suspect input %+v", e.suspects.calls)
	}
	if e.decider.lastState.Error.Code == "" {
		t.Fatal("full mode should include the code line")
	}
	if len(e.decider.lastState.Candidates) != 0 {
		t.Fatalf("unexpected candidates %+v", e.decider.lastState.Candidates)
	}
}

func TestDecisionsOnlyHasNoCode(t *testing.T) {
	e := newEnv(t, enum.AIModeDecisionsOnly)
	issue, ev := e.addIssue("fp1", "error", "TypeError", "1.1.0", nil)
	e.run(issue, ev, false)
	if e.decider.calls != 1 || e.decider.lastState.Error.Code != "" {
		t.Fatalf("calls=%d code=%q", e.decider.calls, e.decider.lastState.Error.Code)
	}
	// decisions-only never launches agents, so no autofix marker even for action=fix
	id := e.storedTaskID(issue)
	if e.tasks.decision(id, QAutofixCandidate) != nil {
		t.Fatal("unexpected autofix marker")
	}
}

func TestDuplicateLinksExistingTask(t *testing.T) {
	e := newEnv(t, enum.AIModeFull)
	existingTask := int64(7)
	e.addIssue("fpA", "error", "TypeError", "1.0.0", &existingTask)
	issue, ev := e.addIssue("fpB", "error", "TypeError", "1.1.0", nil)
	e.decider.overrides = map[string]decisions.Decision{
		questions.DuplicateQuestionID(1): {Type: decisions.TypeNoul, Value: "true", Probability: 0.95, Confidence: 0.9},
	}

	e.run(issue, ev, false)

	if len(e.tasks.created) != 0 {
		t.Fatal("no task must be created for a duplicate")
	}
	if got := e.storedTaskID(issue); got != existingTask {
		t.Fatalf("issue task %d", got)
	}
	if len(e.tasks.links[existingTask]) != 1 || len(e.tasks.comments[existingTask]) != 1 {
		t.Fatalf("links %v comments %v", e.tasks.links, e.tasks.comments)
	}
	if e.tasks.decision(existingTask, questions.DuplicateQuestionID(1)) == nil {
		t.Fatal("duplicate decision not recorded")
	}
}

func TestUnsureDuplicateCreatesTask(t *testing.T) {
	e := newEnv(t, enum.AIModeFull)
	existingTask := int64(7)
	e.addIssue("fpA", "error", "TypeError", "1.0.0", &existingTask)
	issue, ev := e.addIssue("fpB", "error", "TypeError", "1.1.0", nil)
	e.decider.overrides = map[string]decisions.Decision{
		questions.DuplicateQuestionID(1): {Type: decisions.TypeNoul, Value: "true", Probability: 0.6, Confidence: 0.9},
	}
	e.run(issue, ev, false)
	if len(e.tasks.created) != 1 || len(e.decider.lastState.Candidates) != 1 {
		t.Fatalf("created=%d candidates=%d", len(e.tasks.created), len(e.decider.lastState.Candidates))
	}
}

func TestRegressionBumpsPriority(t *testing.T) {
	cases := []struct {
		sev        string
		regression bool
		want       enum.TaskPriority
	}{
		{"medium", false, enum.TaskPriorityP2},
		{"medium", true, enum.TaskPriorityP1},
		{"low", true, enum.TaskPriorityP2},
		{"critical", true, enum.TaskPriorityP0},
	}
	for _, c := range cases {
		e := newEnv(t, enum.AIModeFull)
		e.decider.overrides = map[string]decisions.Decision{
			questions.QSeverity: {Type: decisions.TypeScore, Value: c.sev, Confidence: 0.9},
		}
		issue, ev := e.addIssue("fp", "error", "TypeError", "1.1.0", nil)
		e.run(issue, ev, c.regression)
		if got := e.tasks.created[0].Priority; got != c.want {
			t.Errorf("%s regression=%v: got %s want %s", c.sev, c.regression, got, c.want)
		}
	}
}

func TestQuotaExhausted(t *testing.T) {
	e := newEnv(t, enum.AIModeFull)
	e.decider.err = fmt.Errorf("%w: wait", ErrQuotaExhausted)
	issue, ev := e.addIssue("fp", "fatal", "TypeError", "1.1.0", nil)

	e.run(issue, ev, false)

	if len(e.tasks.created) != 1 || e.tasks.created[0].Priority != enum.TaskPriorityP0 {
		t.Fatalf("created %+v", e.tasks.created)
	}
	id := e.storedTaskID(issue)
	if got := e.tasks.states[id]; len(got) != 1 || got[0] != enum.TaskStateWaitingQuota {
		t.Fatalf("states %v", got)
	}
	if len(e.tasks.comments[id]) != 1 {
		t.Fatalf("comments %v", e.tasks.comments[id])
	}
	if d := e.tasks.decision(id, questions.QAction); d == nil || !d.Fallback || d.Value != "notify" {
		t.Fatalf("rules decision %+v", d)
	}
}

func TestAIFailureUsesRules(t *testing.T) {
	e := newEnv(t, enum.AIModeFull)
	e.decider.err = ErrAIUnavailable
	issue, ev := e.addIssue("fp", "warning", "TypeError", "1.1.0", nil)
	e.run(issue, ev, false)
	id := e.storedTaskID(issue)
	if got := e.tasks.states[id]; len(got) != 1 || got[0] != enum.TaskStateTriage {
		t.Fatalf("states %v", got)
	}
	// warning seen 3 times: low
	if e.tasks.created[0].Priority != enum.TaskPriorityP3 {
		t.Fatalf("priority %s", e.tasks.created[0].Priority)
	}
}

func TestAIOff(t *testing.T) {
	e := newEnv(t, enum.AIModeOff)
	issue, ev := e.addIssue("fp", "fatal", "TypeError", "1.1.0", nil)
	e.run(issue, ev, false)
	if e.decider.calls != 0 || len(e.suspects.calls) != 0 {
		t.Fatalf("AI used: decider=%d suspects=%d", e.decider.calls, len(e.suspects.calls))
	}
	if e.tasks.created[0].Priority != enum.TaskPriorityP0 {
		t.Fatalf("priority %s", e.tasks.created[0].Priority)
	}
	id := e.storedTaskID(issue)
	if d := e.tasks.decision(id, questions.QSeverity); d == nil || d.Model != "rules" || !d.Fallback {
		t.Fatalf("decision %+v", d)
	}
	if got := e.tasks.states[id]; len(got) != 1 || got[0] != enum.TaskStateTriage {
		t.Fatalf("states %v", got)
	}
}

func TestRulesFrequencyBump(t *testing.T) {
	got := rulesOutcome(&types.InsightsIssue{Level: "error", TimesSeen: 500})
	if got.severity != questions.SeverityCritical {
		t.Fatalf("severity %s", got.severity)
	}
}

func TestAsyncObserver(t *testing.T) {
	e := newEnv(t, enum.AIModeOff)
	issue, ev := e.addIssue("fp", "error", "TypeError", "1.1.0", nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.obs.Start(ctx)
	if err := e.obs.OnIssueCreated(ctx, e.project, issue, ev); err != nil {
		t.Fatal(err)
	}
	e.obs.Stop()
	if len(e.tasks.created) != 1 {
		t.Fatalf("created %d", len(e.tasks.created))
	}
	if err := e.obs.OnIssueCreated(ctx, e.project, issue, ev); err == nil {
		t.Fatal("expected error after stop")
	}
}

func TestQueueFullDrops(t *testing.T) {
	e := newEnv(t, enum.AIModeOff)
	e.obs = New(Config{QueueSize: 1}, e.obs.deps) // not started: nothing drains
	issue, ev := e.addIssue("fp", "error", "TypeError", "1.1.0", nil)
	if err := e.obs.OnIssueCreated(context.Background(), e.project, issue, ev); err != nil {
		t.Fatal(err)
	}
	if err := e.obs.OnIssueCreated(context.Background(), e.project, issue, ev); err == nil {
		t.Fatal("expected backpressure error")
	}
}
