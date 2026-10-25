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
	"encoding/json"
	"errors"
	"fmt"

	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/agentpolicy"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Decider asks the AI gateway for decisions (aidecide.Service).
type Decider interface {
	Decide(ctx context.Context, in aidecide.DecideInput) (*aidecide.Result, error)
}

// Completion outcomes recorded in the task history.
const (
	runSucceeded = "succeeded"
	runFailed    = "failed"
)

// AITaskFinished must be called when an AI task reached a final state (completed or error). AI tasks that
// were not launched by this package (no task link or no launcher metadata) are ignored.
func (l *Launcher) AITaskFinished(ctx context.Context, aiTask *types.AITask) error {
	if aiTask == nil || aiTask.TaskID == nil || !aiTask.State.IsFinalStatus() {
		return nil
	}
	var meta metadata
	if len(aiTask.OutputMetadata) == 0 || json.Unmarshal(aiTask.OutputMetadata, &meta) != nil || meta.Model == "" {
		return nil
	}
	pid := l.principalID()
	taskID := *aiTask.TaskID
	success := aiTask.State == enum.AITaskStateCompleted

	l.deps.Registry.Delete(aiTask.GitspaceConfigID)

	status := agentguard.AttemptFailed
	outcome := runFailed
	if success {
		status, outcome = agentguard.AttemptSucceeded, runSucceeded
	}
	if meta.AttemptID != 0 {
		if err := l.deps.Guard.FinishAttempt(ctx, meta.AttemptID, status); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to finish attempt")
		}
	}
	if err := l.deps.Tasks.RecordAgentRun(ctx, pid, taskID, task.AgentRunInput{
		Model:        meta.Model,
		RequestsUsed: aiTask.RequestsUsed,
		Outcome:      outcome,
		AITaskRef:    aiTask.Identifier,
	}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to record agent run")
	}
	var decisionID int64
	if aiTask.ModelDecisionID != nil {
		decisionID = *aiTask.ModelDecisionID
	}
	if err := l.deps.Models.RecordOutcome(ctx, decisionID, meta.Model, string(meta.Kind), success); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to record model outcome")
	}
	if success {
		return nil
	}

	errText := ""
	if aiTask.ErrorMessage != nil {
		errText = l.redact(*aiTask.ErrorMessage)
	}
	prev := append(append([]questions.Attempt{}, meta.Previous...),
		questions.Attempt{Model: meta.Model, Outcome: "failed", Error: errText})
	return l.afterFailure(ctx, taskID, meta, prev)
}

func (l *Launcher) afterFailure(ctx context.Context, taskID int64, meta metadata, prev []questions.Attempt) error {
	pid := l.principalID()
	t, err := l.deps.TaskFinder.Find(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to find task %d: %w", taskID, err)
	}

	step := l.nextStep(ctx, t, prev)
	switch step {
	case questions.NextRetryOtherModel:
		repo, rerr := l.deps.Repos.FindByID(ctx, t.RepoID)
		if rerr != nil {
			return fmt.Errorf("failed to find repo %d: %w", t.RepoID, rerr)
		}
		res, cerr := l.Consider(ctx, ConsiderInput{
			Task: t, Repo: repo, Kind: meta.Kind, PreviousAttempts: prev,
			TriggerDecision: agentpolicy.TriggerDecision{
				ID: meta.TriggerID, Value: meta.TriggerValue, Confidence: meta.TriggerConfidence,
			},
		})
		if cerr != nil {
			l.escalate(ctx, pid, t, "retry with another model failed: "+cerr.Error())
			return cerr
		}
		if res.Outcome == OutcomeNoModel || res.Outcome == OutcomeDenied {
			// Consider already explained the reason; park the task for a human
			l.escalate(ctx, pid, t, fmt.Sprintf("retry with another model was not possible (%s)", res.Outcome))
		}
		return nil
	case questions.NextAbandon:
		l.release(ctx, pid, t)
		l.comment(ctx, pid, t.ID, "The fix agent gave up on this task after "+fmt.Sprint(len(prev))+" attempt(s).")
		if _, terr := l.deps.Tasks.Transition(ctx, pid, t.ID, enum.TaskStateTriage, "Fix agent abandoned the task"); terr != nil {
			return fmt.Errorf("failed to move task to triage: %w", terr)
		}
		return nil
	default:
		l.escalate(ctx, pid, t, "the fix agent failed and the next step is a human")
		return nil
	}
}

// nextStep asks the models what to do next; any problem (AI off, quota) means a human.
func (l *Launcher) nextStep(ctx context.Context, t *types.Task, prev []questions.Attempt) questions.NextStep {
	state := &questions.GenericState{
		Task:     questions.TaskInfo{Kind: "fix", Title: l.redact(t.Title), Priority: string(t.Priority)},
		Attempts: prev,
	}
	ask := questions.GenericAsk{NextStep: true}
	qs, err := questions.GenericQuestions(state, ask)
	if err != nil {
		return questions.NextEscalateHuman
	}
	taskID := t.ID
	res, err := l.deps.Decider.Decide(ctx, aidecide.DecideInput{
		TaskID:    &taskID,
		RepoID:    t.RepoID,
		SessionID: fmt.Sprintf("agent-task-%d", t.ID),
		State:     state,
		Questions: qs,
		Fallback: func(qs map[string]decisions.Question) map[string]decisions.Decision {
			out := make(map[string]decisions.Decision, len(qs))
			for id := range qs {
				out[id] = decisions.Decision{
					QuestionID: id, Type: decisions.TypeChoice, Value: string(questions.NextEscalateHuman),
				}
			}
			return out
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: next_step decision failed, escalating to a human")
		return questions.NextEscalateHuman
	}
	triage, err := questions.ParseGeneric(state, ask, res.Decisions)
	if err != nil || triage.NextStep == nil {
		return questions.NextEscalateHuman
	}
	return triage.NextStep.Step
}

// release takes the task away from the agent.
func (l *Launcher) release(ctx context.Context, pid int64, t *types.Task) {
	if _, err := l.deps.Tasks.Update(ctx, pid, t.ID, task.UpdateInput{Assignee: &task.AssigneeInput{}}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to unassign the agent")
	}
}

// escalate hands the task to a human: the CODEOWNERS suggestion if there is one, and the state bloqueada.
func (l *Launcher) escalate(ctx context.Context, pid int64, t *types.Task, why string) {
	assignee := &task.AssigneeInput{}
	if l.deps.Assigner != nil {
		_, in, err := l.deps.Assigner.SuggestForTask(ctx, t, nil)
		switch {
		case err != nil && !errors.Is(err, context.Canceled):
			log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to suggest a human assignee")
		case in != nil:
			assignee = in
		}
	}
	if _, err := l.deps.Tasks.Update(ctx, pid, t.ID, task.UpdateInput{Assignee: assignee}); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to assign the task to a human")
	}
	if _, err := l.deps.Tasks.Transition(ctx, pid, t.ID, enum.TaskStateBlocked,
		"Needs a human: "+why); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("agent launcher: failed to block the task")
	}
}
