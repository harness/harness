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

package agentlauncher

import (
	"context"
	"errors"
	"strings"
	"testing"

	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/agentpolicy"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/services/taskassign"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type fakePolicy struct {
	v      *agentpolicy.Verdict
	err    error
	inputs []agentpolicy.EvaluateInput
}

func (f *fakePolicy) Evaluate(_ context.Context, in agentpolicy.EvaluateInput) (*agentpolicy.Verdict, error) {
	f.inputs = append(f.inputs, in)
	return f.v, f.err
}

type fakeModels struct {
	model    string
	decision int64
	err      error
	reqs     []aigateway.SelectRequest
	outcomes []string
}

func (f *fakeModels) SelectModel(_ context.Context, r aigateway.SelectRequest) (*aigateway.SelectResult, error) {
	f.reqs = append(f.reqs, r)
	if f.err != nil {
		return nil, f.err
	}
	return &aigateway.SelectResult{Model: f.model, DecisionID: f.decision}, nil
}

func (f *fakeModels) RecordOutcome(_ context.Context, id int64, model, kind string, ok bool) error {
	f.outcomes = append(f.outcomes, strings.Join([]string{model, kind, boolStr(ok)}, "|"))
	_ = id
	return nil
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

type fakeGuard struct {
	started  int
	finished map[int64]agentguard.AttemptStatus
}

func (f *fakeGuard) StartAttempt(context.Context, agentguard.Request) (agentguard.Attempt, error) {
	f.started++
	return agentguard.Attempt{ID: int64(f.started)}, nil
}

func (f *fakeGuard) FinishAttempt(_ context.Context, id int64, s agentguard.AttemptStatus) error {
	if f.finished == nil {
		f.finished = map[int64]agentguard.AttemptStatus{}
	}
	f.finished[id] = s
	return nil
}

type fakeTasks struct {
	task        *types.Task
	transitions []enum.TaskState
	comments    []string
	updates     []task.UpdateInput
	runs        []task.AgentRunInput
}

func (f *fakeTasks) Transition(_ context.Context, _, _ int64, to enum.TaskState, _ string) (*types.Task, error) {
	f.transitions = append(f.transitions, to)
	return f.task, nil
}

func (f *fakeTasks) Update(_ context.Context, _, _ int64, in task.UpdateInput) (*types.Task, error) {
	f.updates = append(f.updates, in)
	return f.task, nil
}

func (f *fakeTasks) AddComment(_ context.Context, _, _ int64, text string) error {
	f.comments = append(f.comments, text)
	return nil
}

func (f *fakeTasks) RecordAgentRun(_ context.Context, _, _ int64, in task.AgentRunInput) error {
	f.runs = append(f.runs, in)
	return nil
}

func (f *fakeTasks) Find(context.Context, int64) (*types.Task, error) { return f.task, nil }

type fakeRepos struct{ repo *types.RepositoryCore }

func (f fakeRepos) FindByID(context.Context, int64) (*types.RepositoryCore, error) {
	return f.repo, nil
}

type fakeAITasks struct {
	created []*types.AITask
	updated []*types.AITask
}

func (f *fakeAITasks) Create(_ context.Context, in *types.AITask) error {
	in.ID = int64(len(f.created) + 1)
	f.created = append(f.created, in)
	return nil
}

func (f *fakeAITasks) Update(_ context.Context, in *types.AITask) error {
	f.updated = append(f.updated, in)
	return nil
}

type fakeEmitter struct {
	payloads []*aitaskevents.AITaskEventPayload
	err      error
}

func (f *fakeEmitter) EmitAITaskEvent(_ context.Context, _ events.EventType, p *aitaskevents.AITaskEventPayload) error {
	f.payloads = append(f.payloads, p)
	return f.err
}

type fakeGitspaces struct{ err error }

func (f fakeGitspaces) Resolve(context.Context, *types.RepositoryCore) (*types.GitspaceConfig, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &types.GitspaceConfig{ID: 77, SpaceID: 5}, nil
}

type fakeRegistry struct{ m map[int64]string }

func (f *fakeRegistry) Set(id int64, model string) { f.m[id] = model }
func (f *fakeRegistry) Delete(id int64)            { delete(f.m, id) }

type fakeDecider struct {
	step questions.NextStep
	err  error
}

func (f *fakeDecider) Decide(_ context.Context, in aidecide.DecideInput) (*aidecide.Result, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &aidecide.Result{Decisions: map[string]decisions.Decision{
		questions.QNextStep: {
			QuestionID: questions.QNextStep, Type: decisions.TypeChoice, Value: string(f.step), Confidence: 0.9,
		},
	}}, nil
}

type fakeAssigner struct {
	in    *task.AssigneeInput
	paths *[]string
}

func (f fakeAssigner) SuggestForTask(
	_ context.Context, _ *types.Task, paths []string,
) (taskassign.Suggestion, *task.AssigneeInput, error) {
	if f.paths != nil {
		*f.paths = paths
	}
	return taskassign.Suggestion{}, f.in, nil
}

type fakePaths struct {
	paths []string
	err   error
}

func (f fakePaths) TaskPaths(context.Context, *types.Task) ([]string, error) { return f.paths, f.err }

type rig struct {
	l        *Launcher
	policy   *fakePolicy
	models   *fakeModels
	guard    *fakeGuard
	tasks    *fakeTasks
	aitasks  *fakeAITasks
	emitter  *fakeEmitter
	registry *fakeRegistry
	decider  *fakeDecider
	in       ConsiderInput
}

func newRig() *rig {
	tk := &types.Task{ID: 9, Number: 12, RepoID: 3, Title: "NPE in handler", Description: "stack", Priority: enum.TaskPriorityP1,
		State: enum.TaskStateTriage}
	repo := &types.RepositoryCore{ID: 3, ParentID: 5}
	r := &rig{
		policy:   &fakePolicy{v: &agentpolicy.Verdict{Allowed: true, Reason: agentpolicy.ReasonAllowed}},
		models:   &fakeModels{model: "free/model-a:free", decision: 42},
		guard:    &fakeGuard{},
		tasks:    &fakeTasks{task: tk},
		aitasks:  &fakeAITasks{},
		emitter:  &fakeEmitter{},
		registry: &fakeRegistry{m: map[int64]string{}},
		decider:  &fakeDecider{step: questions.NextEscalateHuman},
		in: ConsiderInput{
			Task: tk, Repo: repo, Kind: KindFixError,
			TriggerDecision: agentpolicy.TriggerDecision{ID: 1, Value: "fix", Confidence: 0.9},
		},
	}
	r.l = New(Deps{
		Policy: r.policy, Models: r.models, Guard: r.guard, Tasks: r.tasks, TaskFinder: r.tasks,
		Repos: fakeRepos{repo}, AITasks: r.aitasks, Events: r.emitter, Gitspaces: fakeGitspaces{},
		Registry: r.registry, Decider: r.decider, Assigner: fakeAssigner{},
		Principal: func() *types.Principal { return &types.Principal{ID: 100, UID: "harness-agent"} },
	})
	return r
}

func TestConsiderAllowedLaunches(t *testing.T) {
	r := newRig()
	res, err := r.l.Consider(context.Background(), r.in)
	if err != nil || res.Outcome != OutcomeLaunched {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(r.aitasks.created) != 1 {
		t.Fatalf("created=%d", len(r.aitasks.created))
	}
	ai := r.aitasks.created[0]
	if ai.TaskID == nil || *ai.TaskID != 9 || ai.ModelDecisionID == nil || *ai.ModelDecisionID != 42 ||
		ai.RequestBudget != types.DefaultAITaskRequestBudget || ai.GitspaceConfigID != 77 || ai.SpaceID != 5 {
		t.Fatalf("unexpected ai task %+v", ai)
	}
	if !strings.Contains(ai.InitialPrompt, "Fixes TASK-12") || !strings.Contains(ai.InitialPrompt, "failing test") {
		t.Fatalf("prompt: %s", ai.InitialPrompt)
	}
	req := r.models.reqs[0]
	if req.Kind != "fix_error" || !req.Requirements.Tools || req.Requirements.MinContext != 32000 {
		t.Fatalf("select request %+v", req)
	}
	if len(r.tasks.updates) != 1 || !r.tasks.updates[0].Assignee.IsAgent {
		t.Fatalf("assignee updates %+v", r.tasks.updates)
	}
	if len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateInProgress {
		t.Fatalf("transitions %v", r.tasks.transitions)
	}
	if r.guard.started != 1 {
		t.Fatal("attempt not started")
	}
	if len(r.emitter.payloads) != 1 || r.emitter.payloads[0].Type != enum.AITaskEventStart ||
		r.emitter.payloads[0].AITaskIdentifier != ai.Identifier {
		t.Fatalf("events %+v", r.emitter.payloads)
	}
	if r.registry.m[77] != "free/model-a:free" {
		t.Fatalf("registry %v", r.registry.m)
	}
}

func TestConsiderWaitingQuota(t *testing.T) {
	r := newRig()
	r.policy.v = &agentpolicy.Verdict{Reason: agentpolicy.ReasonWaitingQuota, Details: "quota"}
	res, err := r.l.Consider(context.Background(), r.in)
	if err != nil || res.Outcome != OutcomeWaiting {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateWaitingQuota || len(r.aitasks.created) != 0 {
		t.Fatalf("transitions=%v created=%d", r.tasks.transitions, len(r.aitasks.created))
	}
}

func TestConsiderDeniedRecordsReasonOnly(t *testing.T) {
	r := newRig()
	r.policy.v = &agentpolicy.Verdict{Reason: agentpolicy.ReasonPriority, Details: "too low"}
	res, err := r.l.Consider(context.Background(), r.in)
	if err != nil || res.Outcome != OutcomeDenied || res.Reason != agentpolicy.ReasonPriority {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if len(r.tasks.transitions) != 0 || len(r.tasks.comments) != 1 || !strings.Contains(r.tasks.comments[0], "too low") {
		t.Fatalf("transitions=%v comments=%v", r.tasks.transitions, r.tasks.comments)
	}
	if len(r.models.reqs) != 0 || len(r.aitasks.created) != 0 {
		t.Fatal("nothing must be selected or created")
	}
}

func TestConsiderNoModel(t *testing.T) {
	r := newRig()
	r.models.err = aigateway.ErrNoCandidates
	res, err := r.l.Consider(context.Background(), r.in)
	if err != nil || res.Outcome != OutcomeNoModel || len(r.tasks.comments) != 1 || len(r.aitasks.created) != 0 {
		t.Fatalf("res=%+v err=%v comments=%v", res, err, r.tasks.comments)
	}
}

func TestConsiderNoGitspace(t *testing.T) {
	r := newRig()
	r.l.deps.Gitspaces = fakeGitspaces{err: ErrNoGitspace}
	_, err := r.l.Consider(context.Background(), r.in)
	if !errors.Is(err, ErrNoGitspace) || len(r.tasks.comments) != 1 || len(r.aitasks.created) != 0 {
		t.Fatalf("err=%v comments=%v", err, r.tasks.comments)
	}
}

func TestConsiderUnknownKind(t *testing.T) {
	r := newRig()
	r.in.Kind = "nope"
	if _, err := r.l.Consider(context.Background(), r.in); !errors.Is(err, ErrUnknownKind) {
		t.Fatalf("err=%v", err)
	}
}

func TestConsiderSkipsFailedModel(t *testing.T) {
	r := newRig()
	r.in.PreviousAttempts = []questions.Attempt{{Model: "free/model-a:free", Outcome: "failed"}}
	res, err := r.l.Consider(context.Background(), r.in)
	if err != nil || res.Outcome != OutcomeNoModel {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if got := r.models.reqs[0].Task.Attempts; len(got) != 1 || got[0].Model != "free/model-a:free" {
		t.Fatalf("attempts %+v", got)
	}
}

func TestConsiderStartEventFailureBlocksTask(t *testing.T) {
	r := newRig()
	r.emitter.err = errors.New("boom")
	if _, err := r.l.Consider(context.Background(), r.in); err == nil {
		t.Fatal("expected error")
	}
	last := r.tasks.transitions[len(r.tasks.transitions)-1]
	if last != enum.TaskStateBlocked || r.guard.finished[1] != agentguard.AttemptFailed {
		t.Fatalf("transitions=%v finished=%v", r.tasks.transitions, r.guard.finished)
	}
	if _, ok := r.registry.m[77]; ok {
		t.Fatal("registry must be cleaned")
	}
}

// launched returns the AI task of a launched attempt, in the given final state.
func launched(t *testing.T, r *rig, state enum.AITaskState) *types.AITask {
	t.Helper()
	if _, err := r.l.Consider(context.Background(), r.in); err != nil {
		t.Fatal(err)
	}
	ai := r.aitasks.updated[len(r.aitasks.updated)-1]
	ai.State = state
	ai.RequestsUsed = 7
	if state == enum.AITaskStateError {
		msg := "tests failed"
		ai.ErrorMessage = &msg
	}
	r.tasks.transitions = nil
	r.tasks.updates = nil
	r.tasks.comments = nil
	r.tasks.task.State = enum.TaskStateInProgress
	return ai
}

func TestCompletionSuccess(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateCompleted)
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if r.guard.finished[1] != agentguard.AttemptSucceeded {
		t.Fatalf("finished %v", r.guard.finished)
	}
	if len(r.tasks.runs) != 1 || r.tasks.runs[0].Outcome != "succeeded" || r.tasks.runs[0].RequestsUsed != 7 ||
		r.tasks.runs[0].Model != "free/model-a:free" {
		t.Fatalf("runs %+v", r.tasks.runs)
	}
	if len(r.models.outcomes) != 1 || r.models.outcomes[0] != "free/model-a:free|fix_error|true" {
		t.Fatalf("outcomes %v", r.models.outcomes)
	}
	if len(r.tasks.transitions) != 0 {
		t.Fatalf("no transition expected: %v", r.tasks.transitions)
	}
	if _, ok := r.registry.m[77]; ok {
		t.Fatal("registry must be cleaned")
	}
}

func TestCompletionFailureEscalatesToHuman(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	uid := int64(55)
	r.l.deps.Assigner = fakeAssigner{in: &task.AssigneeInput{PrincipalID: &uid}}
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if r.guard.finished[1] != agentguard.AttemptFailed || r.tasks.runs[0].Outcome != "failed" ||
		r.models.outcomes[0] != "free/model-a:free|fix_error|false" {
		t.Fatalf("finished=%v runs=%+v outcomes=%v", r.guard.finished, r.tasks.runs, r.models.outcomes)
	}
	if len(r.tasks.updates) != 1 || r.tasks.updates[0].Assignee.PrincipalID == nil ||
		*r.tasks.updates[0].Assignee.PrincipalID != 55 {
		t.Fatalf("updates %+v", r.tasks.updates)
	}
	if len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateBlocked {
		t.Fatalf("transitions %v", r.tasks.transitions)
	}
}

func TestCompletionFailureDeciderDownEscalates(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	r.decider.err = aigateway.ErrQuotaExhausted
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateBlocked {
		t.Fatalf("transitions %v", r.tasks.transitions)
	}
}

func TestCompletionFailureRetryOtherModel(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	r.decider.step = questions.NextRetryOtherModel
	r.models.model = "free/model-b:free"
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(r.aitasks.created) != 2 {
		t.Fatalf("created=%d", len(r.aitasks.created))
	}
	second := r.aitasks.created[1]
	if !strings.Contains(string(second.OutputMetadata), "model-b") || !strings.Contains(string(second.OutputMetadata), "model-a") {
		t.Fatalf("metadata %s", second.OutputMetadata)
	}
	req := r.models.reqs[1]
	if len(req.Task.Attempts) != 1 || req.Task.Attempts[0].Model != "free/model-a:free" {
		t.Fatalf("second select %+v", req)
	}
	for _, s := range r.tasks.transitions {
		if s == enum.TaskStateBlocked {
			t.Fatal("must not block")
		}
	}
}

func TestCompletionRetryIsMarkedSoTheCooldownDoesNotApply(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	r.decider.step = questions.NextRetryOtherModel
	r.models.model = "free/model-b:free"
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(r.policy.inputs) != 2 || r.policy.inputs[0].RetryOtherModel || !r.policy.inputs[1].RetryOtherModel {
		t.Fatalf("policy inputs %+v", r.policy.inputs)
	}
}

func TestEscalationPassesTaskPathsToAssigner(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	var got []string
	r.l.deps.Assigner = fakeAssigner{paths: &got}
	r.l.deps.Paths = fakePaths{paths: []string{"pay/charge.go", "go.mod"}}
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != "pay/charge.go" {
		t.Fatalf("paths %v", got)
	}
}

func TestEscalationPathErrorIsBestEffort(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	var got []string
	r.l.deps.Assigner = fakeAssigner{paths: &got}
	r.l.deps.Paths = fakePaths{err: errors.New("boom")}
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 || len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateBlocked {
		t.Fatalf("paths=%v transitions=%v", got, r.tasks.transitions)
	}
}

func TestConsiderFindingKinds(t *testing.T) {
	for _, tc := range []struct {
		kind enum.ShieldFindingKind
		want string
	}{
		{enum.ShieldFindingKindDependency, "fix_dependency"},
		{enum.ShieldFindingKindSAST, "fix_sast"},
	} {
		r := newRig()
		tid := r.in.Task.ID
		r.l.ConsiderFinding(context.Background(), r.in.Repo,
			&types.ShieldFinding{ID: 4, Kind: tc.kind, TaskID: &tid}, 12, 0.9)
		if len(r.models.reqs) != 1 || r.models.reqs[0].Kind != tc.want {
			t.Fatalf("%s: select requests %+v", tc.kind, r.models.reqs)
		}
		td := r.policy.inputs[0].TriggerDecision
		if td.ID != 12 || td.Value != "agent" || td.Confidence != 0.9 {
			t.Fatalf("%s: trigger %+v", tc.kind, td)
		}
	}
}

func TestConsiderFindingWithoutTaskDoesNothing(t *testing.T) {
	r := newRig()
	r.l.ConsiderFinding(context.Background(), r.in.Repo, &types.ShieldFinding{ID: 4}, 12, 1)
	if len(r.policy.inputs) != 0 {
		t.Fatal("policy must not be asked")
	}
}

func TestCompletionFailureRetryButSameModelEscalates(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	r.decider.step = questions.NextRetryOtherModel // model stays model-a: nothing else to try
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(r.aitasks.created) != 1 {
		t.Fatalf("created=%d", len(r.aitasks.created))
	}
	if last := r.tasks.transitions[len(r.tasks.transitions)-1]; last != enum.TaskStateBlocked {
		t.Fatalf("transitions %v", r.tasks.transitions)
	}
}

func TestCompletionFailureAbandon(t *testing.T) {
	r := newRig()
	ai := launched(t, r, enum.AITaskStateError)
	r.decider.step = questions.NextAbandon
	if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
		t.Fatal(err)
	}
	if len(r.tasks.comments) != 1 || len(r.tasks.transitions) != 1 || r.tasks.transitions[0] != enum.TaskStateTriage {
		t.Fatalf("comments=%v transitions=%v", r.tasks.comments, r.tasks.transitions)
	}
	if len(r.tasks.updates) != 1 || r.tasks.updates[0].Assignee.IsAgent {
		t.Fatalf("agent must be released: %+v", r.tasks.updates)
	}
}

func TestCompletionIgnoresForeignAITasks(t *testing.T) {
	r := newRig()
	id := int64(1)
	for _, ai := range []*types.AITask{
		nil,
		{State: enum.AITaskStateError},
		{State: enum.AITaskStateRunning, TaskID: &id},
		{State: enum.AITaskStateError, TaskID: &id},
	} {
		if err := r.l.AITaskFinished(context.Background(), ai); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.tasks.runs) != 0 {
		t.Fatal("nothing must be recorded")
	}
}

func TestPromptsRenderForAllKinds(t *testing.T) {
	for _, k := range []Kind{KindFixError, KindFixDependency, KindFixSAST} {
		p, err := renderPrompt(k, promptData{Number: 3, Title: "t", Priority: "p1", Description: "d"})
		if err != nil || !strings.Contains(p, "Fixes TASK-3") || !strings.Contains(strings.ToLower(p), "test") {
			t.Fatalf("%s: %v %s", k, err, p)
		}
	}
}
