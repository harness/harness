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
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/aigateway/questions"
	"github.com/harness/gitness/app/services/task"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// QAutofixCandidate is the decision recorded for the future policy engine when the model
	// chose action=fix with enough confidence and the repository allows agents.
	QAutofixCandidate = "autofix_candidate"

	rulesModel       = "rules"
	rulesConfidence  = 0.5
	updateRetries    = 3
	highFrequency    = 100
	highUsers        = 50
	lowFrequency     = 5
	quotaReason      = "AI quota exhausted, triage will resume when it is available"
	triageReason     = "Triaged from Insights issue"
	sessionIDPattern = "insights-issue-%d"
)

// outcome is the settled triage result, from the model or from rules.
type outcome struct {
	severity  questions.Severity
	action    questions.Action
	decisions []task.DecisionInput
	triage    *questions.InsightsTriage
	model     string
	fallback  bool
	human     bool
	quota     bool
	aiFailure string
}

func (o *Observer) principal() int64 {
	if o.deps.SystemPrincipal != nil {
		return o.deps.SystemPrincipal()
	}
	return bootstrap.NewSystemServiceSession().Principal.ID
}

func aiAllowed(mode enum.AIMode) bool {
	return mode == enum.AIModeFull || mode == enum.AIModeDecisionsOnly
}

func (o *Observer) process(ctx context.Context, j job) error {
	repo, err := o.deps.Repos.FindByID(ctx, j.project.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo: %w", err)
	}
	mode, err := o.deps.Modes.Resolve(ctx, repo)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("insights triage: unknown ai mode, using rules")
		mode = enum.AIModeOff
	}

	// work on the stored issue: other events may have raised the counters or set a task already
	issue, err := o.deps.Issues.Find(ctx, j.issue.ID)
	if err != nil {
		return fmt.Errorf("failed to find issue: %w", err)
	}
	if issue.TaskID != nil && !j.regression {
		return nil
	}

	info := parseEvent(j.event)
	var (
		dups     []dupCandidate
		suspects []questions.SuspectCommit
	)
	if aiAllowed(mode) {
		var releases []string
		dups, releases, err = o.scan(ctx, issue, info)
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("insights triage: duplicate scan failed")
		}
		releases = append(releases, issue.FirstRelease)
		prev := previousRelease(j.event.Release, releases...)
		suspects = o.findSuspects(ctx, repo, j.event.Release, prev, info, j.event.Timestamp)
	}

	state := questions.InsightsState{
		Error: questions.ErrorInfo{
			IssueID: strconv.FormatInt(issue.ID, 10), Type: info.Type, Message: info.Message,
			Platform: issue.Platform, Release: j.event.Release, Environment: j.event.Environment,
			Frames: frameLabels(info.Frames), EventCount: int(issue.TimesSeen), UsersAffected: int(issue.UsersSeen),
			Regression: j.regression,
		},
		SuspectCommits: suspects,
	}
	if mode == enum.AIModeFull && len(info.Frames) > 0 {
		state.Error.Code = info.Frames[0].ContextLine
	}
	for _, d := range dups {
		state.Candidates = append(state.Candidates, questions.Candidate{
			IssueID: strconv.FormatInt(d.issue.ID, 10), Type: d.info.Type, Message: d.issue.Title,
			FingerprintDistance: 1 - d.overlap, Frames: frameLabels(d.info.Frames),
			EventCount: int(d.issue.TimesSeen),
		})
	}

	out := o.decide(ctx, mode, issue, &state)
	return o.apply(ctx, repo, mode, j, issue, dups, &state, out)
}

// decide asks the model, or falls back to rules when the AI is off, unavailable or failed.
func (o *Observer) decide(
	ctx context.Context,
	mode enum.AIMode,
	issue *types.InsightsIssue,
	state *questions.InsightsState,
) *outcome {
	if !aiAllowed(mode) || o.deps.Decider == nil {
		return rulesOutcome(issue)
	}
	if err := state.Fit(questions.Limits{IncludeCode: mode == enum.AIModeFull}); err != nil {
		out := rulesOutcome(issue)
		out.aiFailure = err.Error()
		return out
	}
	qs, err := questions.InsightsQuestions(state)
	if err != nil {
		out := rulesOutcome(issue)
		out.aiFailure = err.Error()
		return out
	}
	res, err := o.deps.Decider.Decide(ctx, DecideRequest{
		SessionID: fmt.Sprintf(sessionIDPattern, issue.ID), State: state, Questions: qs, KeyQuestion: questions.QAction,
	})
	if err != nil {
		out := rulesOutcome(issue)
		out.aiFailure = err.Error()
		out.quota = errors.Is(err, ErrQuotaExhausted)
		return out
	}
	t, err := questions.ParseInsights(state, res.Decisions)
	if err != nil {
		out := rulesOutcome(issue)
		out.aiFailure = err.Error()
		return out
	}
	out := &outcome{
		severity: t.Severity, action: t.Action, triage: &t, model: res.Model, fallback: res.Fallback,
		human: res.Human, decisions: toDecisionInputs(res),
	}
	if out.human && out.action == questions.ActionFix {
		out.action = questions.ActionNotify // low confidence: a person decides first
	}
	return out
}

func toDecisionInputs(res *DecideResult) []task.DecisionInput {
	ids := make([]string, 0, len(res.Decisions))
	for id := range res.Decisions {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]task.DecisionInput, 0, len(ids))
	for _, id := range ids {
		d := res.Decisions[id]
		in := task.DecisionInput{
			QuestionID: id, Value: d.Value, Confidence: clamp01(d.Confidence), Probabilities: d.Probabilities,
			Model: res.Model, Fallback: res.Fallback,
		}
		if d.Type == "noul" || (in.Probabilities == nil && (d.Value == "true" || d.Value == "false")) {
			in.Probabilities = map[string]float64{"true": d.Probability, "false": 1 - d.Probability}
		}
		out = append(out, in)
	}
	return out
}

func clamp01(v float64) float64 {
	return min(max(v, 0), 1)
}

func priorityFor(sev questions.Severity, regression bool) enum.TaskPriority {
	level := 2 // unknown severities are medium
	switch sev {
	case questions.SeverityCritical:
		level = 0
	case questions.SeverityHigh:
		level = 1
	case questions.SeverityMedium:
		level = 2
	case questions.SeverityLow:
		level = 3
	}
	if regression {
		level = max(level-1, 0)
	}
	return [...]enum.TaskPriority{
		enum.TaskPriorityP0, enum.TaskPriorityP1, enum.TaskPriorityP2, enum.TaskPriorityP3,
	}[level]
}

// rulesOutcome is the deterministic fallback: severity from the level and the frequency, action notify.
func rulesOutcome(issue *types.InsightsIssue) *outcome {
	sev := questions.SeverityLow
	switch strings.ToLower(issue.Level) {
	case "fatal":
		sev = questions.SeverityCritical
	case "error":
		sev = questions.SeverityHigh
	case "warning":
		sev = questions.SeverityMedium
		if issue.TimesSeen < lowFrequency {
			sev = questions.SeverityLow
		}
	}
	if issue.TimesSeen >= highFrequency || issue.UsersSeen >= highUsers {
		sev = bumpSeverity(sev)
	}
	out := &outcome{severity: sev, action: questions.ActionNotify, model: rulesModel, fallback: true}
	out.decisions = []task.DecisionInput{
		{QuestionID: questions.QSeverity, Value: string(sev), Confidence: rulesConfidence, Model: rulesModel, Fallback: true},
		{QuestionID: questions.QAction, Value: string(out.action), Confidence: rulesConfidence, Model: rulesModel, Fallback: true},
	}
	return out
}

func bumpSeverity(s questions.Severity) questions.Severity {
	switch s {
	case questions.SeverityLow:
		return questions.SeverityMedium
	case questions.SeverityMedium:
		return questions.SeverityHigh
	case questions.SeverityHigh, questions.SeverityCritical:
		return questions.SeverityCritical
	}
	return s
}

// apply turns the outcome into task changes.
func (o *Observer) apply(
	ctx context.Context,
	repo *types.RepositoryCore,
	mode enum.AIMode,
	j job,
	issue *types.InsightsIssue,
	dups []dupCandidate,
	state *questions.InsightsState,
	out *outcome,
) error {
	pid := o.principal()
	refID := strconv.FormatInt(issue.ID, 10)

	if t := o.confirmedDuplicate(out, dups); t != nil && t.issue.TaskID != nil {
		taskID := *t.issue.TaskID
		if err := o.deps.Tasks.Link(ctx, pid, taskID, enum.TaskLinkKindInsightsIssue, refID); err != nil {
			return fmt.Errorf("failed to link duplicate issue: %w", err)
		}
		if err := o.setTaskID(ctx, issue, taskID); err != nil {
			return err
		}
		o.recordDecisions(ctx, pid, taskID, out)
		note := fmt.Sprintf("Insights issue #%d (%s) was detected as a duplicate of issue #%d and linked here.",
			issue.ID, issue.Title, t.issue.ID)
		if j.regression {
			note += " The error regressed in release " + j.event.Release + "."
		}
		return o.deps.Tasks.AddComment(ctx, pid, taskID, note)
	}

	prio := priorityFor(out.severity, j.regression)
	tk, created, err := o.deps.Tasks.Create(ctx, pid, task.CreateInput{
		RepoID:      repo.ID,
		Title:       issue.Title,
		Description: describe(issue, j, state, out),
		Origin:      enum.TaskOriginInsights,
		OriginRef:   fmt.Sprintf("%d:%s", issue.ProjectID, issue.Fingerprint),
		Priority:    prio,
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}
	if err = o.deps.Tasks.Link(ctx, pid, tk.ID, enum.TaskLinkKindInsightsIssue, refID); err != nil {
		return fmt.Errorf("failed to link issue: %w", err)
	}
	if err = o.setTaskID(ctx, issue, tk.ID); err != nil {
		return err
	}
	if !created {
		if j.regression {
			_ = o.deps.Tasks.AddComment(ctx, pid, tk.ID, "The error regressed in release "+j.event.Release+".")
		}
		return nil
	}

	to, reason := enum.TaskStateTriage, triageReason
	if out.quota {
		to, reason = enum.TaskStateWaitingQuota, quotaReason
	}
	if _, err = o.deps.Tasks.Transition(ctx, pid, tk.ID, to, reason); err != nil {
		return fmt.Errorf("failed to move task to %s: %w", to, err)
	}
	o.recordDecisions(ctx, pid, tk.ID, out)
	if out.action == questions.ActionFix && !out.human && out.triage != nil && mode == enum.AIModeFull {
		// marker for the future policy engine: no agent is launched from here
		o.recordDecision(ctx, pid, tk.ID, task.DecisionInput{
			QuestionID: QAutofixCandidate, Value: "true", Confidence: clamp01(out.triage.ActionConfidence),
			Model: out.model, Fallback: out.fallback,
		})
	}
	if note := triageNote(mode, out); note != "" {
		if err = o.deps.Tasks.AddComment(ctx, pid, tk.ID, note); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("insights triage: failed to add note")
		}
	}
	return nil
}

// confirmedDuplicate returns the best duplicate candidate when the model is sure enough.
func (o *Observer) confirmedDuplicate(out *outcome, dups []dupCandidate) *dupCandidate {
	if out.triage == nil || out.triage.BestDuplicate == nil {
		return nil
	}
	b := out.triage.BestDuplicate
	if b.Probability < o.cfg.DuplicateConfidence || b.Confidence < o.cfg.DuplicateConfidence {
		return nil
	}
	if b.Candidate < 1 || b.Candidate > len(dups) {
		return nil
	}
	return &dups[b.Candidate-1]
}

func (o *Observer) recordDecisions(ctx context.Context, pid, taskID int64, out *outcome) {
	for _, d := range out.decisions {
		o.recordDecision(ctx, pid, taskID, d)
	}
}

func (o *Observer) recordDecision(ctx context.Context, pid, taskID int64, d task.DecisionInput) {
	if err := o.deps.Tasks.RecordDecision(ctx, pid, taskID, d); err != nil {
		log.Ctx(ctx).Warn().Err(err).Str("question", d.QuestionID).Msg("insights triage: failed to record decision")
	}
}

// setTaskID stores the task on the issue, retrying on optimistic locking conflicts.
func (o *Observer) setTaskID(ctx context.Context, issue *types.InsightsIssue, taskID int64) error {
	cur := issue
	for range updateRetries {
		cur.TaskID = &taskID
		err := o.deps.Issues.Update(ctx, cur)
		if err == nil {
			return nil
		}
		if !errors.Is(err, gitness_store.ErrVersionConflict) {
			return fmt.Errorf("failed to set task on issue: %w", err)
		}
		if cur, err = o.deps.Issues.Find(ctx, issue.ID); err != nil {
			return fmt.Errorf("failed to reload issue: %w", err)
		}
	}
	return fmt.Errorf("failed to set task on issue: %w", gitness_store.ErrVersionConflict)
}

func describe(issue *types.InsightsIssue, j job, state *questions.InsightsState, out *outcome) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Insights issue #%d, fingerprint `%s`.\n\n", issue.ID, issue.Fingerprint)
	if issue.Culprit != "" {
		fmt.Fprintf(&b, "- Culprit: %s\n", issue.Culprit)
	}
	fmt.Fprintf(&b, "- Level: %s, seen %d times by %d users\n", issue.Level, issue.TimesSeen, issue.UsersSeen)
	if j.event.Release != "" {
		fmt.Fprintf(&b, "- Release: %s\n", j.event.Release)
	}
	if j.regression {
		b.WriteString("- Regression: the issue had been resolved and came back\n")
	}
	if out.triage != nil && out.triage.SuspectCommit != "" {
		for _, c := range state.SuspectCommits {
			if questions.ShortSHA(c.SHA) == out.triage.SuspectCommit {
				fmt.Fprintf(&b, "- Suspect commit: %s %s\n", out.triage.SuspectCommit, c.Subject)
			}
		}
	}
	return b.String()
}

// triageNote is the comment explaining how the task was triaged, when it adds information.
func triageNote(mode enum.AIMode, out *outcome) string {
	switch {
	case out.quota:
		return "AI quota exhausted: the task waits (esperando_cuota) and was prioritized by rules (" +
			string(out.severity) + "). " + out.aiFailure
	case out.aiFailure != "":
		return "AI triage failed, prioritized by rules (" + string(out.severity) + "): " + out.aiFailure
	case !aiAllowed(mode):
		return "AI mode is off for this repository, prioritized by rules (" + string(out.severity) + ")."
	case out.human:
		return "The model was not confident enough to act on its own, a person should review."
	}
	return ""
}
