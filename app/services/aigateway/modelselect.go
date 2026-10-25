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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/harness/gitness/app/services/aigateway/decisions"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const (
	// QuestionModelForTask is the id of the decision question used to pick a model for a task.
	QuestionModelForTask = "model_for_task"

	// modelConfidenceThreshold is the minimum confidence to accept the choice of the decision model.
	modelConfidenceThreshold = 0.6

	// ModelSourceSingle, ModelSourceDecision and ModelSourceHistory tell how a model was selected.
	ModelSourceSingle   = "single_candidate"
	ModelSourceDecision = "decision"
	ModelSourceHistory  = "history"

	// countKeySuffix marks the observation counter inside AIModel.SuccessRateByTask.
	countKeySuffix = "#n"

	// deterministicBuild is stored as model build of decisions that were not answered by a model.
	deterministicBuild = "deterministic"
)

// ErrNoCandidates is returned when no free model satisfies the requirements.
var ErrNoCandidates = errors.New("no free model satisfies the requirements")

// Attempt is a previous attempt of the task with the model that made it.
type Attempt struct {
	Model   string `json:"model"`
	Outcome string `json:"outcome"` // e.g. "failure" or "success"
}

// TaskDescription is the compact, already non-sensitive description of the task used as decision state.
type TaskDescription struct {
	Language   string    `json:"language,omitempty"`
	Size       string    `json:"size,omitempty"`       // e.g. "small", "medium", "large"
	Difficulty string    `json:"difficulty,omitempty"` // e.g. "easy", "hard"
	Attempts   []Attempt `json:"previous_attempts,omitempty"`
}

// SelectRequest asks for the best free model for a kind of task.
type SelectRequest struct {
	// Kind is the task kind, e.g. fix_error, fix_dependency, fix_sast, triage_summary or explain.
	Kind         string
	Requirements Requirements
	Task         TaskDescription
	SessionID    string
	// TaskID links the recorded decision to a task, when there is one.
	TaskID *int64
}

// SelectResult is the selected model and how it was selected.
type SelectResult struct {
	Model string
	// Source is one of ModelSourceSingle, ModelSourceDecision or ModelSourceHistory.
	Source string
	// DecisionID is the id of the recorded ai_decisions row; 0 if nothing was recorded.
	DecisionID int64
	// Confidence is the confidence reported by the decision model, if it was asked.
	Confidence float64
	// QuotaFallback is true when the deterministic pick was used because the quota was exhausted.
	QuotaFallback bool
	// DecisionFallback is true when the fallback decision model answered.
	DecisionFallback bool
	// Reason is a short human-readable explanation of the pick.
	Reason string
}

type candidateStats struct {
	model Model
	rate  float64 // -1 when there are no observations
	n     int
	p95   int
}

// SelectModel picks the free model to use for a task. Candidates are filtered deterministically;
// only when several remain, the decision model is asked. The quota is never spent for 0 or 1 candidates.
func (g *Gateway) SelectModel(ctx context.Context, req SelectRequest) (*SelectResult, error) {
	if !g.Enabled() {
		return nil, ErrDisabled
	}

	candidates := g.selectionCandidates(req.Requirements)
	switch len(candidates) {
	case 0:
		return nil, ErrNoCandidates
	case 1:
		return &SelectResult{
			Model:  candidates[0].ID,
			Source: ModelSourceSingle,
			Reason: "only one candidate satisfies the requirements",
		}, nil
	}

	stats := g.candidateStats(ctx, req.Kind, candidates)
	state := map[string]any{"task_kind": req.Kind, "task": req.Task}
	stateHash, err := selectionStateHash(state, stats)
	if err != nil {
		return nil, err
	}

	questions := map[string]decisions.Question{
		QuestionModelForTask: decisions.Choice(
			fmt.Sprintf("Choose the model best suited to this %q task. Prefer models with a high observed "+
				"success rate for the task kind, avoid models that already failed in previous attempts, "+
				"and consider context size and latency.", req.Kind),
			describeCandidates(req.Kind, stats),
		),
	}

	res, err := g.Decide(ctx, req.SessionID, state, questions)
	if err != nil {
		if ctx.Err() != nil {
			return nil, err
		}
		quota := errors.Is(err, ErrQuotaExhausted) || IsRateLimited(err)
		log.Ctx(ctx).Warn().Err(err).Bool("quota", quota).
			Msg("ai gateway: model selection falls back to the historical success rate")
		out := g.deterministicPick(stats)
		out.QuotaFallback = quota
		out.Reason = "decision unavailable: " + out.Reason
		g.recordSelection(ctx, req, stateHash, deterministicBuild, out, false)
		return out, nil
	}

	d := res.Decisions[QuestionModelForTask]
	if d.Confidence >= modelConfidenceThreshold {
		out := &SelectResult{
			Model:            d.Value,
			Source:           ModelSourceDecision,
			Confidence:       d.Confidence,
			DecisionFallback: res.Fallback,
			Reason:           "chosen by the decision model",
		}
		g.recordSelection(ctx, req, stateHash, res.Model, out, res.Fallback, d.Probabilities)
		return out, nil
	}

	out := g.deterministicPick(stats)
	out.Confidence = d.Confidence
	out.DecisionFallback = res.Fallback
	out.Reason = fmt.Sprintf("decision confidence %.2f below %.2f: %s", d.Confidence, modelConfidenceThreshold, out.Reason)
	g.recordSelection(ctx, req, stateHash, res.Model, out, res.Fallback, d.Probabilities)
	return out, nil
}

// selectionCandidates are the catalog candidates without the decision models.
func (g *Gateway) selectionCandidates(req Requirements) []Model {
	all := g.catalog.Candidates(req)
	out := make([]Model, 0, len(all))
	for _, m := range all {
		if m.ID == g.config.DecisionModel || m.ID == g.config.DecisionFallbackModel {
			continue
		}
		out = append(out, m)
	}
	return out
}

// candidateStats joins the candidates with their observed performance from the store.
func (g *Gateway) candidateStats(ctx context.Context, kind string, candidates []Model) []candidateStats {
	known := map[string]*types.AIModel{}
	if g.modelStore != nil {
		stored, err := g.modelStore.List(ctx, types.AIModelFilter{})
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to load model statistics")
		}
		for _, m := range stored {
			known[m.ID] = m
		}
	}

	out := make([]candidateStats, 0, len(candidates))
	for _, m := range candidates {
		s := candidateStats{model: m, rate: -1}
		if stored, ok := known[m.ID]; ok {
			s.p95 = stored.P95LatencyMS
			if rate, ok := stored.SuccessRateByTask[kind]; ok {
				s.rate = rate
				s.n = int(stored.SuccessRateByTask[kind+countKeySuffix])
			}
		}
		out = append(out, s)
	}
	return out
}

func describeCandidates(kind string, stats []candidateStats) map[string]string {
	out := make(map[string]string, len(stats))
	for _, s := range stats {
		rate := "no observations yet"
		if s.rate >= 0 {
			rate = fmt.Sprintf("%.0f%% over %d attempts", s.rate*100, s.n)
		}
		latency := "unknown"
		if s.p95 > 0 {
			latency = fmt.Sprintf("%d ms", s.p95)
		}
		tools := "no"
		if s.model.SupportsTools() {
			tools = "yes"
		}
		out[s.model.ID] = fmt.Sprintf("context %d tokens; tool calling: %s; success rate for %s: %s; p95 latency: %s",
			s.model.ContextLength, tools, kind, rate, latency)
	}
	return out
}

// deterministicPick returns the candidate with the best observed success rate for the task kind.
// Models without observations rank last; ties go to the lowest p95 latency (unknown ranks last), then the id.
func (g *Gateway) deterministicPick(stats []candidateStats) *SelectResult {
	sorted := append([]candidateStats(nil), stats...)
	p95 := func(s candidateStats) int {
		if s.p95 <= 0 {
			return math.MaxInt
		}
		return s.p95
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.rate != b.rate {
			return a.rate > b.rate
		}
		if p95(a) != p95(b) {
			return p95(a) < p95(b)
		}
		return a.model.ID < b.model.ID
	})
	return &SelectResult{
		Model:  sorted[0].model.ID,
		Source: ModelSourceHistory,
		Reason: "best historical success rate for the task kind",
	}
}

func selectionStateHash(state any, stats []candidateStats) (string, error) {
	ids := make([]string, len(stats))
	for i, s := range stats {
		ids[i] = s.model.ID
	}
	raw, err := json.Marshal(struct {
		State      any      `json:"s"`
		Candidates []string `json:"c"`
	}{state, ids})
	if err != nil {
		return "", fmt.Errorf("failed to hash model selection state: %w", err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// recordSelection stores the decision in ai_decisions when a store is available; failures are only logged.
func (g *Gateway) recordSelection(
	ctx context.Context,
	req SelectRequest,
	stateHash, build string,
	out *SelectResult,
	fallback bool,
	probabilities ...map[string]float64,
) {
	if g.decisionStore == nil {
		return
	}
	value, err := json.Marshal(out.Model)
	if err != nil {
		return
	}
	answer := types.AIDecisionAnswer{Value: value, Confidence: out.Confidence}
	if len(probabilities) > 0 {
		answer.Probabilities = probabilities[0]
	}
	rec := &types.AIDecision{
		TaskID:     req.TaskID,
		QuestionID: QuestionModelForTask,
		ModelBuild: build,
		StateHash:  stateHash,
		Answer:     answer,
		Fallback:   fallback,
	}
	if err := g.decisionStore.Create(ctx, rec); err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("ai gateway: failed to record model selection")
		return
	}
	out.DecisionID = rec.ID
}

// RecordOutcome stores the observed outcome of a selection. It marks the decision (when decisionID is
// not zero) and folds the outcome into the success rate of the model for the task kind.
//
// Format of AIModel.SuccessRateByTask: "<kind>" holds the running average of the outcomes (1 for
// success, 0 for failure) and "<kind>#n" holds the number of observations behind it. Consumers that
// iterate the map must skip keys ending in "#n".
func (g *Gateway) RecordOutcome(ctx context.Context, decisionID int64, modelID, kind string, success bool) error {
	outcome := enum.AIDecisionOutcomeFailure
	if success {
		outcome = enum.AIDecisionOutcomeSuccess
	}
	if g.decisionStore != nil && decisionID != 0 {
		if err := g.decisionStore.SetOutcome(ctx, decisionID, outcome); err != nil {
			return fmt.Errorf("failed to set outcome of decision %d: %w", decisionID, err)
		}
	}
	if g.modelStore == nil {
		return nil
	}

	g.outcomeMu.Lock()
	defer g.outcomeMu.Unlock()

	m, err := g.modelStore.Find(ctx, modelID)
	if errors.Is(err, gitnessstore.ErrResourceNotFound) {
		cm, ok := g.catalog.Get(modelID)
		if !ok {
			return fmt.Errorf("model %s is not in the catalog: %w", modelID, err)
		}
		m = &types.AIModel{
			ID:            modelID,
			Kind:          g.catalog.kindOf(modelID),
			ContextLength: cm.ContextLength,
			SupportsTools: cm.SupportsTools(),
			Status:        enum.AIModelStatusActive,
			LastSeen:      time.Now().UnixMilli(),
		}
	} else if err != nil {
		return fmt.Errorf("failed to find model %s: %w", modelID, err)
	}

	m.SuccessRateByTask = foldOutcome(m.SuccessRateByTask, kind, success)
	if err := g.modelStore.Upsert(ctx, m); err != nil {
		return fmt.Errorf("failed to update success rate of model %s: %w", modelID, err)
	}
	return nil
}

func foldOutcome(rates map[string]float64, kind string, success bool) map[string]float64 {
	if rates == nil {
		rates = map[string]float64{}
	}
	kind = strings.TrimSuffix(kind, countKeySuffix)
	n := rates[kind+countKeySuffix]
	v := 0.0
	if success {
		v = 1
	}
	rates[kind] = (rates[kind]*n + v) / (n + 1)
	rates[kind+countKeySuffix] = n + 1
	return rates
}
