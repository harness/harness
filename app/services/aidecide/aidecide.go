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

// Package aidecide is the single entry point for features that ask the AI gateway
// for decisions: it respects the repo AI mode, persists the audit trail (ai_decisions),
// writes the task history and applies deterministic rules when the models are unavailable.
package aidecide

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// RulesBuild is stored as model build of decisions made by the caller's deterministic rules.
	RulesBuild = "rules"

	// maxSecondOpinionBytes is the size the second opinion is truncated to in the task history.
	maxSecondOpinionBytes = 1024
)

var (
	// ErrNoFallback is returned when AI can't be used and the caller supplied no (complete) fallback rules.
	ErrNoFallback = errors.New("ai decisions are unavailable and no fallback rules were provided")
	// ErrInvalidInput is returned for malformed inputs.
	ErrInvalidInput = errors.New("invalid decide input")
)

// Gateway is the part of the AI gateway used by this package.
type Gateway interface {
	Decide(
		ctx context.Context,
		sessionID string,
		state any,
		questions map[string]decisions.Question,
	) (*aigateway.DecideResult, error)
	DecideWithEscalation(ctx context.Context, req aigateway.EscalationRequest) (*aigateway.EscalationResult, error)
	// RedactState returns the state as it is sent to the models.
	RedactState(state any) (any, error)
}

// ModeResolver resolves the AI mode of a repository.
type ModeResolver interface {
	Resolve(ctx context.Context, repo *types.RepositoryCore) (enum.AIMode, error)
}

// RepoFinder finds repositories by id.
type RepoFinder interface {
	FindByID(ctx context.Context, repoID int64) (*types.RepositoryCore, error)
}

// TaskRecorder writes decisions to the task history.
type TaskRecorder interface {
	RecordDecisionSummary(ctx context.Context, principalID int64, taskID int64, in task.DecisionSummaryInput) error
}

// FallbackFunc answers the questions with deterministic rules (e.g. CVSS + KEV).
// It must answer every question.
type FallbackFunc func(questions map[string]decisions.Question) map[string]decisions.Decision

// DecideInput describes a batch of questions.
type DecideInput struct {
	// TaskID links the decisions to a task and enables the task history entry. Optional.
	TaskID    *int64
	RepoID    int64
	SessionID string
	State     any
	Questions map[string]decisions.Question
	// KeyQuestion enables confidence based escalation to System 2. Optional.
	KeyQuestion string
	// Thresholds defaults to aigateway.DefaultThresholds. Used only with KeyQuestion.
	Thresholds aigateway.Thresholds
	// Fallback supplies deterministic rules used when AI is off or both decision models fail.
	Fallback FallbackFunc
}

// Result is the outcome of Decide.
type Result struct {
	Decisions map[string]decisions.Decision
	// Model is the build that answered, or RulesBuild.
	Model string
	// Fallback is true when the fallback decision model or the rules answered.
	Fallback bool
	// Rules is true when the deterministic rules answered.
	Rules bool
	// Mode is the AI mode that applied.
	Mode enum.AIMode
	// Escalation is set when KeyQuestion was given and the models answered.
	Escalation *aigateway.EscalationResult
	// DecisionIDs maps question id to the id of the recorded ai_decisions row.
	DecisionIDs map[string]int64
}

// Service orchestrates decisions.
type Service struct {
	gateway   Gateway
	modes     ModeResolver
	repos     RepoFinder
	decisions store.AIDecisionStore
	tasks     TaskRecorder
}

// NewService creates the service. tasks and decisionStore may be nil to skip those writes.
func NewService(
	gateway Gateway,
	modes ModeResolver,
	repos RepoFinder,
	decisionStore store.AIDecisionStore,
	tasks TaskRecorder,
) *Service {
	return &Service{gateway: gateway, modes: modes, repos: repos, decisions: decisionStore, tasks: tasks}
}

// Decide answers the questions. Quota errors are returned unchanged so the caller can wait for quota.
func (s *Service) Decide(ctx context.Context, in DecideInput) (*Result, error) {
	if len(in.Questions) == 0 {
		return nil, fmt.Errorf("%w: at least one question is required", ErrInvalidInput)
	}
	if in.KeyQuestion != "" {
		if _, ok := in.Questions[in.KeyQuestion]; !ok {
			return nil, fmt.Errorf("%w: key question %q is not in the questions", ErrInvalidInput, in.KeyQuestion)
		}
	}

	mode, err := s.mode(ctx, in.RepoID)
	if err != nil {
		return nil, err
	}

	stateHash := s.stateHash(in.State)

	if !aigateway.AllowsDecisions(mode) {
		res, err := s.rules(in, mode, nil)
		if err != nil {
			return nil, err
		}
		s.persist(ctx, in, res, stateHash)
		return res, nil
	}

	res, err := s.askModels(ctx, in, mode)
	if err != nil {
		if !canUseRules(ctx, err) {
			return nil, err
		}
		log.Ctx(ctx).Warn().Err(err).Msg("ai decide: decision models failed, using fallback rules")
		res, err = s.rules(in, mode, err)
		if err != nil {
			return nil, err
		}
	}
	s.persist(ctx, in, res, stateHash)
	return res, nil
}

func (s *Service) mode(ctx context.Context, repoID int64) (enum.AIMode, error) {
	repo, err := s.repos.FindByID(ctx, repoID)
	if err != nil {
		return "", fmt.Errorf("failed to find repo %d: %w", repoID, err)
	}
	mode, err := s.modes.Resolve(ctx, repo)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ai mode: %w", err)
	}
	return mode, nil
}

// canUseRules tells whether an error from the gateway lets the caller's rules answer.
// Quota errors and cancellations are returned to the caller unchanged.
func canUseRules(ctx context.Context, err error) bool {
	if ctx.Err() != nil || errors.Is(err, aigateway.ErrQuotaExhausted) || aigateway.IsRateLimited(err) {
		return false
	}
	return !errors.Is(err, decisions.ErrInvalidRequest) && !errors.Is(err, aigateway.ErrInvalidEscalation) &&
		!errors.Is(err, aigateway.ErrInvalidThresholds)
}

func (s *Service) askModels(ctx context.Context, in DecideInput, mode enum.AIMode) (*Result, error) {
	if in.KeyQuestion != "" {
		esc, err := s.gateway.DecideWithEscalation(ctx, aigateway.EscalationRequest{
			SessionID:   in.SessionID,
			State:       in.State,
			Questions:   in.Questions,
			KeyQuestion: in.KeyQuestion,
			Thresholds:  in.Thresholds,
		})
		if err != nil {
			return nil, err
		}
		final := esc.First
		if esc.Second != nil {
			final = esc.Second
		}
		return &Result{
			Decisions:  final.Decisions,
			Model:      final.Model,
			Fallback:   final.Fallback,
			Mode:       mode,
			Escalation: esc,
		}, nil
	}

	res, err := s.gateway.Decide(ctx, in.SessionID, in.State, in.Questions)
	if err != nil {
		return nil, err
	}
	return &Result{Decisions: res.Decisions, Model: res.Model, Fallback: res.Fallback, Mode: mode}, nil
}

func (s *Service) rules(in DecideInput, mode enum.AIMode, cause error) (*Result, error) {
	if in.Fallback == nil {
		if cause != nil {
			return nil, fmt.Errorf("%w: %w", ErrNoFallback, cause)
		}
		return nil, fmt.Errorf("%w (ai mode %q)", ErrNoFallback, mode)
	}
	out := in.Fallback(in.Questions)
	for id := range in.Questions {
		d, ok := out[id]
		if !ok {
			return nil, fmt.Errorf("%w: fallback rules did not answer %q", ErrNoFallback, id)
		}
		if d.QuestionID == "" {
			d.QuestionID = id
			out[id] = d
		}
	}
	return &Result{Decisions: out, Model: RulesBuild, Fallback: true, Rules: true, Mode: mode}, nil
}

// stateHash is the sha256 of the redacted state JSON.
func (s *Service) stateHash(state any) string {
	redacted, err := s.gateway.RedactState(state)
	if err != nil {
		redacted = state
	}
	raw, err := json.Marshal(redacted)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", redacted))
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// persist stores the audit rows and the task history entry. Failures are logged and don't fail the decision.
func (s *Service) persist(ctx context.Context, in DecideInput, res *Result, stateHash string) {
	ids := sortedIDs(res.Decisions)
	res.DecisionIDs = make(map[string]int64, len(ids))
	inputs := make([]task.DecisionInput, 0, len(ids))

	for _, id := range ids {
		d := res.Decisions[id]
		value, err := json.Marshal(d.Value)
		if err != nil {
			continue
		}
		if s.decisions != nil {
			rec := &types.AIDecision{
				TaskID:     in.TaskID,
				QuestionID: id,
				ModelBuild: res.Model,
				StateHash:  stateHash,
				Answer: types.AIDecisionAnswer{
					Value:         value,
					Probabilities: d.Probabilities,
					Confidence:    d.Confidence,
				},
				Fallback: res.Fallback,
			}
			if err := s.decisions.Create(ctx, rec); err != nil {
				log.Ctx(ctx).Warn().Err(err).Str("question", id).Msg("ai decide: failed to record decision")
			} else {
				res.DecisionIDs[id] = rec.ID
			}
		}
		inputs = append(inputs, task.DecisionInput{
			QuestionID:    id,
			Value:         d.Value,
			Confidence:    d.Confidence,
			Probabilities: d.Probabilities,
			Model:         res.Model,
			Fallback:      res.Fallback,
		})
	}

	if in.TaskID == nil || s.tasks == nil || len(inputs) == 0 {
		return
	}
	summary := task.DecisionSummaryInput{Summary: summarize(ids, res), Decisions: inputs}
	if esc := res.Escalation; esc != nil {
		summary.Outcome = string(esc.Outcome)
		summary.Reason = esc.Reason
		summary.SecondOpinion = truncate(esc.SecondOpinion, maxSecondOpinionBytes)
		summary.SecondOpinionModel = esc.SecondOpinionModel
	}
	if err := s.tasks.RecordDecisionSummary(ctx, 0, *in.TaskID, summary); err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("task", *in.TaskID).Msg("ai decide: failed to record task activity")
	}
}

// summarize renders e.g. "severity: high (0.81) · action: fix (0.74)".
func summarize(ids []string, res *Result) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		d := res.Decisions[id]
		parts = append(parts, fmt.Sprintf("%s: %s (%.2f)", id, d.Value, d.Confidence))
	}
	out := strings.Join(parts, " · ")
	if res.Rules {
		out += " [rules]"
	}
	return out
}

func sortedIDs(m map[string]decisions.Decision) []string {
	ids := make([]string, 0, len(m))
	for id := range m {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// truncate cuts s to at most limit bytes without splitting a UTF-8 sequence.
func truncate(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	cut := limit
	for cut > 0 && (s[cut]&0xC0) == 0x80 {
		cut--
	}
	return s[:cut]
}
