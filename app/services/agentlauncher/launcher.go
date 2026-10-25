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

// Package agentlauncher closes the loop between a task and the fix agent: it asks the policy whether an
// agent may run, picks a free model, creates and starts the AI task, and on completion records the outcome and
// decides the next step (retry with another model, escalate to a human or abandon).
//
// The real agent run happens in the gitspace machinery (aitaskevent -> orchestrator); this package only
// talks to it through the AI task store and the AI task start event, so everything is testable with fakes.
package agentlauncher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/agentpolicy"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/services/taskassign"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Kind is the kind of fix an agent is launched for.
type Kind string

const (
	KindFixError      Kind = "fix_error"
	KindFixDependency Kind = "fix_dependency"
	KindFixSAST       Kind = "fix_sast"
)

const (
	// agentMinContext is the minimum context window of a coding agent model.
	agentMinContext = 32000
	// maxDescriptionChars bounds the task description put into the prompt.
	maxDescriptionChars = 4000
)

var (
	// ErrNoGitspace is returned when the repository has no gitspace config the agent can run in.
	ErrNoGitspace = errors.New("no gitspace config is available for the agent")
	// ErrUnknownKind is returned for an unsupported fix kind.
	ErrUnknownKind = errors.New("unknown fix kind")
)

// Policy decides whether an agent may be launched.
type Policy interface {
	Evaluate(ctx context.Context, in agentpolicy.EvaluateInput) (*agentpolicy.Verdict, error)
}

// Models selects models and learns from the outcomes.
type Models interface {
	SelectModel(ctx context.Context, req aigateway.SelectRequest) (*aigateway.SelectResult, error)
	RecordOutcome(ctx context.Context, decisionID int64, modelID, kind string, success bool) error
}

// Guard keeps the attempt bookkeeping.
type Guard interface {
	StartAttempt(ctx context.Context, req agentguard.Request) (agentguard.Attempt, error)
	FinishAttempt(ctx context.Context, attemptID int64, status agentguard.AttemptStatus) error
}

// Tasks is the part of the task service used here.
type Tasks interface {
	Transition(ctx context.Context, principalID, taskID int64, to enum.TaskState, reason string) (*types.Task, error)
	Update(ctx context.Context, principalID, taskID int64, in task.UpdateInput) (*types.Task, error)
	AddComment(ctx context.Context, principalID, taskID int64, text string) error
	RecordAgentRun(ctx context.Context, principalID, taskID int64, in task.AgentRunInput) error
}

// TaskFinder loads tasks.
type TaskFinder interface {
	Find(ctx context.Context, id int64) (*types.Task, error)
}

// RepoFinder loads repositories.
type RepoFinder interface {
	FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error)
}

// AITaskStore is the part of the AI task store used here.
type AITaskStore interface {
	Create(ctx context.Context, in *types.AITask) error
	Update(ctx context.Context, in *types.AITask) error
}

// StartEmitter publishes the AI task start event.
type StartEmitter interface {
	EmitAITaskEvent(ctx context.Context, event events.EventType, payload *aitaskevents.AITaskEventPayload) error
}

// GitspaceResolver finds the gitspace config the agent of a repository runs in.
// It returns ErrNoGitspace when there is none.
type GitspaceResolver interface {
	Resolve(ctx context.Context, repo *types.RepositoryCore) (*types.GitspaceConfig, error)
}

// ModelRegistry hands the chosen model to the gitspace orchestrator.
type ModelRegistry interface {
	Set(gitspaceConfigID int64, model string)
	Delete(gitspaceConfigID int64)
}

// Assigner suggests a human assignee.
type Assigner interface {
	SuggestForTask(ctx context.Context, t *types.Task, paths []string) (taskassign.Suggestion, *task.AssigneeInput, error)
}

// Redactor removes secrets and personal data from text.
type Redactor interface {
	Redact(s string) string
}

// Deps are the collaborators of the Launcher.
type Deps struct {
	Policy     Policy
	Models     Models
	Guard      Guard
	Tasks      Tasks
	TaskFinder TaskFinder
	Repos      RepoFinder
	AITasks    AITaskStore
	Events     StartEmitter
	Gitspaces  GitspaceResolver
	Registry   ModelRegistry
	Decider    Decider
	Assigner   Assigner
	Redactor   Redactor
	// Principal returns the agent service principal (nil if not bootstrapped).
	Principal func() *types.Principal
}

// Launcher launches fix agents for tasks.
type Launcher struct {
	deps Deps
	now  func() time.Time
}

// New creates a Launcher.
func New(deps Deps) *Launcher {
	return &Launcher{deps: deps, now: time.Now}
}

// ConsiderInput is the input of Consider.
type ConsiderInput struct {
	Task *types.Task
	Repo *types.RepositoryCore
	// TriggerDecision is the autofix_candidate or fix_strategy decision that proposes the agent.
	TriggerDecision agentpolicy.TriggerDecision
	Kind            Kind
	// TriggerPrincipalID is the principal that caused the triggering event (0 if unknown).
	TriggerPrincipalID int64
	// PreviousAttempts are earlier attempts at the task, oldest first. Their models are not selected again.
	PreviousAttempts []questions.Attempt
}

// Outcome tells what Consider did.
type Outcome string

const (
	OutcomeLaunched Outcome = "launched"
	OutcomeWaiting  Outcome = "waiting_quota"
	OutcomeDenied   Outcome = "denied"
	// OutcomeNoModel means no usable free model exists; a human is needed.
	OutcomeNoModel Outcome = "no_model"
)

// Result is the result of Consider.
type Result struct {
	Outcome Outcome
	Reason  agentpolicy.Reason
	AITask  *types.AITask
	Model   string
}

// metadata is stored in AITask.OutputMetadata so the completion handler needs no extra state.
type metadata struct {
	Kind              Kind                `json:"agent_kind"`
	Model             string              `json:"agent_model"`
	AttemptID         int64               `json:"agent_attempt_id"`
	TriggerValue      string              `json:"trigger_value"`
	TriggerConfidence float64             `json:"trigger_confidence"`
	TriggerID         int64               `json:"trigger_id"`
	Previous          []questions.Attempt `json:"previous_attempts,omitempty"`
}

func (l *Launcher) principalID() int64 {
	if p := l.deps.Principal(); p != nil {
		return p.ID
	}
	return 0
}

// Consider launches a fix agent for the task when the policy allows it.
//
//nolint:gocognit // linear sequence of steps
func (l *Launcher) Consider(ctx context.Context, in ConsiderInput) (*Result, error) {
	if in.Task == nil || in.Repo == nil {
		return nil, errors.New("agent launcher: task and repo are required")
	}
	if _, err := promptFor(in.Kind); err != nil {
		return nil, err
	}
	pid := l.principalID()

	verdict, err := l.deps.Policy.Evaluate(ctx, agentpolicy.EvaluateInput{
		Repo:               in.Repo,
		Task:               in.Task,
		TriggerDecision:    in.TriggerDecision,
		TriggerPrincipalID: in.TriggerPrincipalID,
		SessionID:          fmt.Sprintf("agent-task-%d", in.Task.ID),
		PreviousAttempts:   in.PreviousAttempts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate agent policy: %w", err)
	}

	switch {
	case verdict.Reason == agentpolicy.ReasonWaitingQuota:
		if _, err = l.deps.Tasks.Transition(ctx, pid, in.Task.ID, enum.TaskStateWaitingQuota,
			"Waiting for AI quota: "+verdict.Details); err != nil {
			return nil, fmt.Errorf("failed to move task to %s: %w", enum.TaskStateWaitingQuota, err)
		}
		return &Result{Outcome: OutcomeWaiting, Reason: verdict.Reason}, nil
	case !verdict.Allowed:
		l.comment(ctx, pid, in.Task.ID, fmt.Sprintf("The fix agent was not launched (%s): %s", verdict.Reason, verdict.Details))
		return &Result{Outcome: OutcomeDenied, Reason: verdict.Reason}, nil
	}

	sel, err := l.selectModel(ctx, in)
	if err != nil {
		if errors.Is(err, aigateway.ErrNoCandidates) || errors.Is(err, aigateway.ErrDisabled) || errors.Is(err, errOnlyFailedModels) {
			l.comment(ctx, pid, in.Task.ID, "The fix agent was not launched: no suitable free model is available ("+err.Error()+").")
			return &Result{Outcome: OutcomeNoModel, Reason: verdict.Reason}, nil
		}
		return nil, fmt.Errorf("failed to select model: %w", err)
	}

	gitspace, err := l.deps.Gitspaces.Resolve(ctx, in.Repo)
	if err != nil {
		if errors.Is(err, ErrNoGitspace) {
			l.comment(ctx, pid, in.Task.ID, "The fix agent was not launched: the repository has no gitspace config for agents.")
		}
		return nil, err
	}

	aiTask, err := l.createAITask(ctx, in, sel, gitspace, verdict)
	if err != nil {
		return nil, err
	}
	if err = l.start(ctx, in, aiTask, gitspace, pid); err != nil {
		return nil, err
	}
	return &Result{Outcome: OutcomeLaunched, Reason: verdict.Reason, AITask: aiTask, Model: sel.Model}, nil
}

var errOnlyFailedModels = errors.New("every candidate model already failed for this task")

func (l *Launcher) selectModel(ctx context.Context, in ConsiderInput) (*aigateway.SelectResult, error) {
	taskID := in.Task.ID
	desc := aigateway.TaskDescription{}
	failed := map[string]bool{}
	for _, a := range in.PreviousAttempts {
		desc.Attempts = append(desc.Attempts, aigateway.Attempt{Model: a.Model, Outcome: a.Outcome})
		failed[a.Model] = true
	}
	sel, err := l.deps.Models.SelectModel(ctx, aigateway.SelectRequest{
		Kind:         string(in.Kind),
		Requirements: aigateway.Requirements{Tools: true, MinContext: agentMinContext},
		Task:         desc,
		SessionID:    fmt.Sprintf("agent-task-%d", in.Task.ID),
		TaskID:       &taskID,
	})
	if err != nil {
		return nil, err
	}
	if failed[sel.Model] {
		return nil, errOnlyFailedModels
	}
	return sel, nil
}

func (l *Launcher) createAITask(
	ctx context.Context,
	in ConsiderInput,
	sel *aigateway.SelectResult,
	gitspace *types.GitspaceConfig,
	verdict *agentpolicy.Verdict,
) (*types.AITask, error) {
	prompt, err := renderPrompt(in.Kind, promptData{
		Number:      in.Task.Number,
		Title:       l.redact(in.Task.Title),
		Priority:    string(in.Task.Priority),
		Description: truncate(l.redact(in.Task.Description), maxDescriptionChars),
	})
	if err != nil {
		return nil, err
	}

	meta, err := json.Marshal(metadata{
		Kind:              in.Kind,
		Model:             sel.Model,
		TriggerValue:      in.TriggerDecision.Value,
		TriggerConfidence: in.TriggerDecision.Confidence,
		TriggerID:         in.TriggerDecision.ID,
		Previous:          in.PreviousAttempts,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode ai task metadata: %w", err)
	}

	agent := l.deps.Principal()
	if agent == nil {
		return nil, errors.New("agent service principal is not set up")
	}
	taskID := in.Task.ID
	aiTask := &types.AITask{
		Identifier:       fmt.Sprintf("fix-task-%d-%d", in.Task.Number, l.now().UnixMilli()),
		GitspaceConfigID: gitspace.ID,
		InitialPrompt:    prompt,
		DisplayName:      fmt.Sprintf("Fix TASK-%d", in.Task.Number),
		UserUID:          agent.UID,
		SpaceID:          gitspace.SpaceID,
		AIAgent:          enum.AIAgentClaudeCode,
		State:            enum.AITaskStateUninitialized,
		OutputMetadata:   meta,
		TaskID:           &taskID,
		RequestBudget:    l.budget(verdict),
	}
	if sel.DecisionID != 0 {
		id := sel.DecisionID
		aiTask.ModelDecisionID = &id
	}
	if err = l.deps.AITasks.Create(ctx, aiTask); err != nil {
		return nil, fmt.Errorf("failed to create ai task: %w", err)
	}
	return aiTask, nil
}

func (l *Launcher) budget(_ *agentpolicy.Verdict) int {
	return types.DefaultAITaskRequestBudget
}

// start assigns the task to the agent, moves it to en_curso, records the attempt and publishes the start
// event. If anything fails after the AI task exists the work is parked as blocked for a human.
func (l *Launcher) start(
	ctx context.Context,
	in ConsiderInput,
	aiTask *types.AITask,
	gitspace *types.GitspaceConfig,
	pid int64,
) error {
	fail := func(step string, err error) error {
		l.abort(ctx, in.Task.ID, aiTask, pid, step+": "+err.Error())
		return fmt.Errorf("failed to %s: %w", step, err)
	}

	if _, err := l.deps.Tasks.Update(ctx, pid, in.Task.ID, task.UpdateInput{
		Assignee: &task.AssigneeInput{IsAgent: true},
	}); err != nil {
		return fail("assign task to the agent", err)
	}
	if in.Task.State != enum.TaskStateInProgress {
		if _, err := l.deps.Tasks.Transition(ctx, pid, in.Task.ID, enum.TaskStateInProgress,
			"Fix agent launched"); err != nil {
			return fail("move task to "+string(enum.TaskStateInProgress), err)
		}
	}

	attempt, err := l.deps.Guard.StartAttempt(ctx, agentguard.Request{
		RepoID:  in.Repo.ID,
		TaskKey: taskKey(in.Task.ID),
	})
	if err != nil {
		return fail("start attempt", err)
	}
	var meta metadata
	_ = json.Unmarshal(aiTask.OutputMetadata, &meta)
	meta.AttemptID = attempt.ID
	if aiTask.OutputMetadata, err = json.Marshal(meta); err != nil {
		return fail("encode ai task metadata", err)
	}
	if err = l.deps.AITasks.Update(ctx, aiTask); err != nil {
		_ = l.deps.Guard.FinishAttempt(ctx, attempt.ID, agentguard.AttemptFailed)
		return fail("save ai task", err)
	}

	l.deps.Registry.Set(gitspace.ID, meta.Model)
	if err = l.deps.Events.EmitAITaskEvent(ctx, aitaskevents.AITaskEvent, &aitaskevents.AITaskEventPayload{
		Type:             enum.AITaskEventStart,
		AITaskIdentifier: aiTask.Identifier,
		AITaskSpaceID:    aiTask.SpaceID,
	}); err != nil {
		l.deps.Registry.Delete(gitspace.ID)
		_ = l.deps.Guard.FinishAttempt(ctx, attempt.ID, agentguard.AttemptFailed)
		return fail("publish ai task start event", err)
	}
	return nil
}

// abort parks the task for a human after a launch failure.
func (l *Launcher) abort(ctx context.Context, taskID int64, aiTask *types.AITask, pid int64, reason string) {
	msg := reason
	aiTask.State = enum.AITaskStateError
	aiTask.ErrorMessage = &msg
	if err := l.deps.AITasks.Update(ctx, aiTask); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to mark ai task as failed")
	}
	if _, err := l.deps.Tasks.Update(ctx, pid, taskID, task.UpdateInput{Assignee: &task.AssigneeInput{}}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to unassign task")
	}
	if _, err := l.deps.Tasks.Transition(ctx, pid, taskID, enum.TaskStateBlocked,
		"Fix agent launch failed: "+reason); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to block task")
	}
}

func (l *Launcher) comment(ctx context.Context, pid, taskID int64, text string) {
	if err := l.deps.Tasks.AddComment(ctx, pid, taskID, text); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task_id", taskID).Msg("agent launcher: failed to record activity")
	}
}

func (l *Launcher) redact(s string) string {
	if l.deps.Redactor == nil {
		return s
	}
	return l.deps.Redactor.Redact(s)
}

func taskKey(taskID int64) string { return fmt.Sprintf("task:%d", taskID) }

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "") + "..."
}
