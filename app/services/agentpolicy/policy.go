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

// Package agentpolicy decides whether a fix agent may be launched for a task.
package agentpolicy

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/harness/gitness/app/services/agentguard"
	"github.com/harness/gitness/app/services/aidecide"
	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Reason explains the outcome of Evaluate.
type Reason string

const (
	ReasonAllowed       Reason = "allowed"
	ReasonAIMode        Reason = "ai_mode"
	ReasonPriority      Reason = "priority"
	ReasonLowConfidence Reason = "low_confidence"
	ReasonNotFix        Reason = "not_fix"
	ReasonGuardDenied   Reason = "guard_denied"
	ReasonQuota         Reason = "quota"
	ReasonNotWorth      Reason = "not_worth_budget"
	// ReasonWaitingQuota tells the caller to set the task to esperando_cuota.
	ReasonWaitingQuota Reason = "waiting_quota"
)

const (
	defaultPriorityThreshold = enum.TaskPriorityP1
	defaultHighConfidence    = 0.8
	defaultAgentBudget       = 60
)

// Config holds the thresholds of the policy.
type Config struct {
	// PriorityThreshold is the lowest priority that qualifies (p0 is the highest).
	PriorityThreshold enum.TaskPriority
	// HighConfidence is the minimum confidence of the trigger and the worth_agent_budget decisions.
	HighConfidence float64
	// AgentBudget is the agent quota that must be left.
	AgentBudget int
}

// NewConfig derives the policy configuration from the global config.
func NewConfig(global *types.Config) Config {
	return Config{
		PriorityThreshold: enum.TaskPriority(global.AIAgents.PriorityThreshold),
		HighConfidence:    global.AIAgents.HighConfidence,
		AgentBudget:       global.AIAgents.AgentBudget,
	}
}

// ModeResolver resolves the AI mode of a repository.
type ModeResolver interface {
	Resolve(ctx context.Context, repo *types.RepositoryCore) (enum.AIMode, error)
}

// Guard is the agent guard.
type Guard interface {
	Check(ctx context.Context, req agentguard.Request) agentguard.Decision
}

// Quota reports the remaining agent quota.
type Quota interface {
	Remaining(class aigateway.Class) int
}

// Decider asks the AI gateway for decisions.
type Decider interface {
	Decide(ctx context.Context, in aidecide.DecideInput) (*aidecide.Result, error)
}

// TriggerDecision is the action or fix_strategy decision that proposes launching an agent.
type TriggerDecision struct {
	// ID is the id of the recorded ai_decisions row (0 if unknown).
	ID int64
	// Value is the chosen option; "fix" and "agent" launch an agent.
	Value      string
	Confidence float64
}

// EvaluateInput is the input of Evaluate.
type EvaluateInput struct {
	Repo *types.RepositoryCore
	Task *types.Task
	// TriggerDecision is the action or fix_strategy decision.
	TriggerDecision TriggerDecision
	// TriggerPrincipalID is the principal that caused the triggering event (0 if unknown).
	TriggerPrincipalID int64
	SessionID          string
	// RetryOtherModel is true when the launch retries a failed attempt with a different model; the guard
	// cooldown does not apply to it.
	RetryOtherModel bool

	// ModelSuccessRate is the success rate of the model that would run, between 0 and 1 (0 if unknown).
	// It is the success probability proxy of the worth_agent_budget decision.
	ModelSuccessRate float64
	// PreviousAttempts are the previous attempts at the task, oldest first.
	PreviousAttempts []questions.Attempt
	// QueuedTasks is the number of tasks waiting for the agent.
	QueuedTasks int
}

// Verdict is the outcome of Evaluate.
type Verdict struct {
	Allowed bool
	Reason  Reason
	Details string
	// GuardReason is set when Reason is ReasonGuardDenied.
	GuardReason agentguard.Reason
	// DecisionIDs are the ids of the decisions used: the trigger and worth_agent_budget (when asked).
	DecisionIDs []int64
}

// Evaluator decides whether a fix agent may be launched.
type Evaluator struct {
	config  Config
	modes   ModeResolver
	guard   Guard
	quota   Quota
	decider Decider
}

// New creates an Evaluator.
func New(config Config, modes ModeResolver, guard Guard, quota Quota, decider Decider) *Evaluator {
	if config.PriorityThreshold == "" {
		config.PriorityThreshold = defaultPriorityThreshold
	}
	if config.HighConfidence <= 0 {
		config.HighConfidence = defaultHighConfidence
	}
	if config.AgentBudget <= 0 {
		config.AgentBudget = defaultAgentBudget
	}
	return &Evaluator{config: config, modes: modes, guard: guard, quota: quota, decider: decider}
}

// Evaluate checks whether a fix agent may be launched for the task. The cheap deterministic checks
// run first, in this order, and stop at the first failure so no quota is spent on a doomed request:
// AI mode, priority, trigger decision, agent guard, remaining quota. Only then worth_agent_budget is asked.
func (e *Evaluator) Evaluate(ctx context.Context, in EvaluateInput) (*Verdict, error) {
	if in.Repo == nil || in.Task == nil {
		return nil, errors.New("agent policy: repo and task are required")
	}
	var ids []int64
	if in.TriggerDecision.ID != 0 {
		ids = append(ids, in.TriggerDecision.ID)
	}
	deny := func(r Reason, format string, args ...any) *Verdict {
		return &Verdict{Reason: r, Details: fmt.Sprintf(format, args...), DecisionIDs: ids}
	}

	mode, err := e.modes.Resolve(ctx, in.Repo)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve ai mode: %w", err)
	}
	if !aigateway.AllowsAgents(mode) {
		return deny(ReasonAIMode, "ai mode %q does not allow agents", mode), nil
	}

	if !priorityQualifies(in.Task.Priority, e.config.PriorityThreshold) {
		return deny(ReasonPriority, "task priority %q is below threshold %q",
			in.Task.Priority, e.config.PriorityThreshold), nil
	}

	if v := strings.ToLower(strings.TrimSpace(in.TriggerDecision.Value)); v != "fix" && v != "agent" {
		return deny(ReasonNotFix, "trigger decision is %q, not fix or agent", in.TriggerDecision.Value), nil
	}
	if in.TriggerDecision.Confidence < e.config.HighConfidence {
		return deny(ReasonLowConfidence, "trigger confidence %.2f is below %.2f",
			in.TriggerDecision.Confidence, e.config.HighConfidence), nil
	}

	gd := e.guard.Check(ctx, agentguard.Request{
		RepoID:                 in.Repo.ID,
		SpacePath:              path.Dir(in.Repo.Path),
		RepoPath:               in.Repo.Path,
		TaskKey:                fmt.Sprintf("task:%d", in.Task.ID),
		MinAutonomy:            settings.AutonomyDraftPR,
		TriggeredByPrincipalID: in.TriggerPrincipalID,
		RetryOtherModel:        in.RetryOtherModel,
	})
	if !gd.Allowed {
		v := deny(ReasonGuardDenied, "%s: %s", gd.Reason, gd.Detail)
		v.GuardReason = gd.Reason
		return v, nil
	}

	remaining := e.quota.Remaining(aigateway.ClassAgent)
	if remaining < e.config.AgentBudget {
		return deny(ReasonQuota, "agent quota left %d is below the budget %d", remaining, e.config.AgentBudget), nil
	}

	return e.askWorth(ctx, in, remaining, ids)
}

func (e *Evaluator) askWorth(ctx context.Context, in EvaluateInput, remaining int, ids []int64) (*Verdict, error) {
	state := &questions.GenericState{
		Task: questions.TaskInfo{
			Kind:               "fix",
			Title:              in.Task.Title,
			Priority:           string(in.Task.Priority),
			SuccessProbability: in.ModelSuccessRate,
		},
		Quota:    questions.QuotaInfo{RemainingToday: remaining, QueuedTasks: in.QueuedTasks},
		Attempts: in.PreviousAttempts,
	}
	ask := questions.GenericAsk{WorthAgentBudget: true}
	qs, err := questions.GenericQuestions(state, ask)
	if err != nil {
		return nil, fmt.Errorf("failed to build worth_agent_budget question: %w", err)
	}

	taskID := in.Task.ID
	res, err := e.decider.Decide(ctx, aidecide.DecideInput{
		TaskID:    &taskID,
		RepoID:    in.Repo.ID,
		SessionID: in.SessionID,
		State:     state,
		Questions: qs,
		// without AI the budget is not spent
		Fallback: func(qs map[string]decisions.Question) map[string]decisions.Decision {
			out := make(map[string]decisions.Decision, len(qs))
			for id := range qs {
				out[id] = decisions.Decision{QuestionID: id, Type: decisions.TypeNoul, Value: "false"}
			}
			return out
		},
	})
	if err != nil {
		if errors.Is(err, aigateway.ErrQuotaExhausted) || aigateway.IsRateLimited(err) {
			return &Verdict{Reason: ReasonWaitingQuota, Details: err.Error(), DecisionIDs: ids}, nil
		}
		return nil, fmt.Errorf("failed to decide worth_agent_budget: %w", err)
	}
	if id, ok := res.DecisionIDs[questions.QWorthAgentBudget]; ok {
		ids = append(ids, id)
	}

	triage, err := questions.ParseGeneric(state, ask, res.Decisions)
	if err != nil {
		return nil, fmt.Errorf("failed to parse worth_agent_budget: %w", err)
	}
	w := triage.WorthAgentBudget
	if !w.Worth || w.Confidence < e.config.HighConfidence {
		return &Verdict{
			Reason: ReasonNotWorth,
			Details: fmt.Sprintf("worth_agent_budget=%t confidence %.2f (need true and >= %.2f)",
				w.Worth, w.Confidence, e.config.HighConfidence),
			DecisionIDs: ids,
		}, nil
	}
	return &Verdict{Allowed: true, Reason: ReasonAllowed, DecisionIDs: ids}, nil
}

// priorityQualifies reports whether p is at least as urgent as threshold (p0 is the most urgent).
func priorityQualifies(p, threshold enum.TaskPriority) bool {
	all, _ := enum.GetAllTaskPriorities()
	pi, ti := -1, -1
	for i, v := range all {
		if v == p {
			pi = i
		}
		if v == threshold {
			ti = i
		}
	}
	return pi >= 0 && ti >= 0 && pi <= ti
}
