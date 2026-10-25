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
	"fmt"
	"strings"

	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// Question ids of the generic group. model_for_task is handled elsewhere.
const (
	QWorthAgentBudget = "worth_agent_budget"
	QNextStep         = "next_step"
	QAssigneeKind     = "assignee_kind"

	// TeamOptionPrefix prefixes the team name in assignee_kind option keys.
	TeamOptionPrefix = "team:"
	MaxTeamOptions   = 20
)

type NextStep string

const (
	NextRetryOtherModel NextStep = "retry_other_model"
	NextEscalateHuman   NextStep = "escalate_human"
	NextAbandon         NextStep = "abandon"
)

type AssigneeKind string

const (
	AssigneeAgent     AssigneeKind = "agent"
	AssigneeTeam      AssigneeKind = "team"
	AssigneeCodeOwner AssigneeKind = "code_owner"
)

// TaskInfo describes the work item.
type TaskInfo struct {
	Kind     string `json:"kind"`
	Title    string `json:"title,omitempty"`
	Priority string `json:"priority,omitempty"`
	Language string `json:"language,omitempty"`
	// DiffStats is a short size summary such as "3 files, +40 -12"; never a diff body.
	DiffStats string `json:"diff_stats,omitempty"`
	// Difficulty is an estimate between 0 and 1.
	Difficulty float64 `json:"difficulty,omitempty"`
	// SuccessProbability is the estimated chance that the agent succeeds, between 0 and 1.
	SuccessProbability float64 `json:"success_probability,omitempty"`
	// Code is a short excerpt, kept only when Limits.IncludeCode is true.
	Code string `json:"code,omitempty"`
}

// QuotaInfo is the agent quota situation.
type QuotaInfo struct {
	RemainingToday int `json:"remaining_today"`
	LimitToday     int `json:"limit_today"`
	// QueuedTasks is the number of tasks waiting for the agent.
	QueuedTasks int `json:"queued_tasks"`
}

// Attempt is a previous try of the same task.
type Attempt struct {
	Model   string `json:"model"`
	Outcome string `json:"outcome"` // e.g. "failed", "tests_failed", "timeout", "rejected"
	Error   string `json:"error,omitempty"`
}

// GenericState is the state of the generic questions.
type GenericState struct {
	Task  TaskInfo  `json:"task"`
	Quota QuotaInfo `json:"quota"`
	// Attempts are the previous attempts, oldest first. The newest ones are kept when shrinking.
	Attempts []Attempt `json:"attempts,omitempty"`
	// Teams are the team names offered as assignee_kind options.
	Teams []string `json:"teams,omitempty"`
	// CodeOwners are the owners resolved from CODEOWNERS for the touched paths (names only).
	CodeOwners []string `json:"code_owners,omitempty"`
}

// Fit compacts the state in place to fit the byte budget of l.
func (s *GenericState) Fit(l Limits) error {
	l = l.normalized()
	s.Task.Title = trunc(s.Task.Title, 200)
	s.Task.Code = l.code(s.Task.Code)
	if len(s.Teams) > MaxTeamOptions {
		s.Teams = s.Teams[:MaxTeamOptions]
	}
	s.Teams = truncList(s.Teams, MaxTeamOptions, 60)
	s.CodeOwners = truncList(s.CodeOwners, 10, 60)
	if len(s.Attempts) > l.MaxFrames {
		s.Attempts = s.Attempts[len(s.Attempts)-l.MaxFrames:]
	}
	for i := range s.Attempts {
		s.Attempts[i].Error = trunc(s.Attempts[i].Error, l.MaxMessageChars/2)
	}
	return fit(l.MaxStateBytes, s,
		func() bool { return shrinkStr(&s.Task.Code, 0) },
		func() bool {
			if len(s.Attempts) <= 2 {
				return false
			}
			s.Attempts = s.Attempts[len(s.Attempts)/2:]
			return true
		},
		func() bool {
			changed := false
			for i := range s.Attempts {
				changed = shrinkStr(&s.Attempts[i].Error, 80) || changed
			}
			return changed
		},
		func() bool { return shrinkStr(&s.Task.Title, 80) },
		func() bool { return halve(&s.CodeOwners, 2) },
	)
}

// GenericAsk selects which generic questions go into the batch.
type GenericAsk struct {
	WorthAgentBudget bool
	NextStep         bool
	AssigneeKind     bool
}

// GenericQuestions builds the selected generic questions for one batched request.
func GenericQuestions(s *GenericState, ask GenericAsk) (map[string]decisions.Question, error) {
	qs := map[string]decisions.Question{}
	if ask.WorthAgentBudget {
		qs[QWorthAgentBudget] = decisions.Noul(instr(
			"Is it worth spending agent quota on this task today? Consider quota.remaining_today against "+
				"quota.queued_tasks, task.priority and task.success_probability."),
			"Spend the quota: the priority and the chance of success justify the cost.",
			"Do not spend the quota today: the quota is scarce, the priority is low or success is unlikely.")
	}
	if ask.NextStep {
		if len(s.Attempts) == 0 {
			return nil, fmt.Errorf("%w: next_step needs at least one attempt", ErrInvalidInput)
		}
		qs[QNextStep] = decisions.Choice(instr(
			"The last attempt at this task failed. Choose the next step using attempts, task.difficulty "+
				"and the error of each attempt."),
			map[string]string{
				string(NextRetryOtherModel): "The failure looks model-specific or random, so another model may succeed.",
				string(NextEscalateHuman):   "The task looks hard or ambiguous, or several models failed the same way.",
				string(NextAbandon):         "The task is not worth more attempts: low value or unlikely to succeed.",
			})
	}
	if ask.AssigneeKind {
		opts := map[string]string{
			string(AssigneeAgent):     "The task is small and well defined, so the agent can do it.",
			string(AssigneeCodeOwner): "The task touches code with owners in code_owners, who know it best.",
		}
		seen := map[string]bool{}
		for _, t := range s.Teams {
			name := strings.TrimSpace(t)
			if name == "" || seen[name] {
				return nil, fmt.Errorf("%w: empty or duplicate team %q", ErrInvalidInput, t)
			}
			seen[name] = true
			opts[TeamOptionPrefix+name] = "Team " + name + " owns this area or kind of work."
		}
		qs[QAssigneeKind] = decisions.Choice(instr(
			"Pick who should take this task: the agent, one of the listed teams, or the code owners."), opts)
	}
	if len(qs) == 0 {
		return nil, fmt.Errorf("%w: no generic question selected", ErrInvalidInput)
	}
	return qs, nil
}

// Assignee is the typed answer of assignee_kind.
type Assignee struct {
	Kind       AssigneeKind
	Team       string // set when Kind is AssigneeTeam
	Confidence float64
}

// GenericTriage is the typed result of GenericQuestions. Fields of questions that were not
// asked stay nil.
type GenericTriage struct {
	WorthAgentBudget *BudgetDecision
	NextStep         *NextStepDecision
	Assignee         *Assignee
}

type BudgetDecision struct {
	Worth       bool
	Probability float64
	Confidence  float64
}

type NextStepDecision struct {
	Step       NextStep
	Confidence float64
}

// ParseGeneric maps decisions to a GenericTriage; every key selected by ask is required.
func ParseGeneric(s *GenericState, ask GenericAsk, ds map[string]decisions.Decision) (GenericTriage, error) {
	var t GenericTriage
	if ask.WorthAgentBudget {
		d, err := need(ds, QWorthAgentBudget, decisions.TypeNoul)
		if err != nil {
			return GenericTriage{}, err
		}
		t.WorthAgentBudget = &BudgetDecision{Worth: d.Value == "true", Probability: d.Probability, Confidence: d.Confidence}
	}
	if ask.NextStep {
		d, err := needOneOf(ds, QNextStep, decisions.TypeChoice,
			string(NextRetryOtherModel), string(NextEscalateHuman), string(NextAbandon))
		if err != nil {
			return GenericTriage{}, err
		}
		t.NextStep = &NextStepDecision{Step: NextStep(d.Value), Confidence: d.Confidence}
	}
	if ask.AssigneeKind {
		allowed := []string{string(AssigneeAgent), string(AssigneeCodeOwner)}
		for _, team := range s.Teams {
			allowed = append(allowed, TeamOptionPrefix+strings.TrimSpace(team))
		}
		d, err := needOneOf(ds, QAssigneeKind, decisions.TypeChoice, allowed...)
		if err != nil {
			return GenericTriage{}, err
		}
		a := &Assignee{Kind: AssigneeKind(d.Value), Confidence: d.Confidence}
		if team, ok := strings.CutPrefix(d.Value, TeamOptionPrefix); ok {
			a.Kind, a.Team = AssigneeTeam, team
		}
		t.Assignee = a
	}
	return t, nil
}
