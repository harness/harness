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

package eval

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/aigateway/decisions"
)

// Result is the outcome of one batch of questions.
type Result struct {
	// Model is the exact build that answered, as reported by the provider.
	Model     string
	Decisions map[string]decisions.Decision
}

// Decider answers the batched questions of one case.
type Decider interface {
	Decide(ctx context.Context, caseID string, state any, qs map[string]decisions.Question) (*Result, error)
}

// GatewayDecider asks the real decision model through the AI gateway.
type GatewayDecider struct {
	Gateway *aigateway.Gateway
}

func (d GatewayDecider) Decide(
	ctx context.Context,
	caseID string,
	state any,
	qs map[string]decisions.Question,
) (*Result, error) {
	res, err := d.Gateway.Decide(ctx, "eval-"+caseID, state, qs)
	if err != nil {
		return nil, err
	}
	return &Result{Model: res.Model, Decisions: res.Decisions}, nil
}

// recordedDecision is the JSON form of decisions.Decision used in replay files.
type recordedDecision struct {
	Type          decisions.QuestionType `json:"type"`
	Value         string                 `json:"value"`
	Probability   float64                `json:"probability,omitempty"`
	Score         float64                `json:"score,omitempty"`
	Probabilities map[string]float64     `json:"probabilities,omitempty"`
	Confidence    float64                `json:"confidence"`
}

// Recording is one line of a replay file: the answers of the model for one case.
type Recording struct {
	CaseID    string                      `json:"case_id"`
	Model     string                      `json:"model"`
	Decisions map[string]recordedDecision `json:"decisions"`
}

func newRecording(caseID string, r *Result) Recording {
	rec := Recording{CaseID: caseID, Model: r.Model, Decisions: make(map[string]recordedDecision, len(r.Decisions))}
	for id, d := range r.Decisions {
		rec.Decisions[id] = recordedDecision{
			Type: d.Type, Value: d.Value, Probability: d.Probability, Score: d.Score,
			Probabilities: d.Probabilities, Confidence: d.Confidence,
		}
	}
	return rec
}

func (r Recording) result() *Result {
	out := &Result{Model: r.Model, Decisions: make(map[string]decisions.Decision, len(r.Decisions))}
	for id, d := range r.Decisions {
		out.Decisions[id] = decisions.Decision{
			QuestionID: id, Type: d.Type, Value: d.Value, Probability: d.Probability, Score: d.Score,
			Probabilities: d.Probabilities, Confidence: d.Confidence,
		}
	}
	return out
}

// ReplayDecider answers from recorded answers, without any network access.
type ReplayDecider struct {
	byCase map[string]Recording
}

// NewReplayDecider reads a replay JSONL file; a later line for the same case wins.
func NewReplayDecider(r io.Reader) (*ReplayDecider, error) {
	d := &ReplayDecider{byCase: map[string]Recording{}}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 1; sc.Scan(); line++ {
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		var rec Recording
		if err := json.Unmarshal(raw, &rec); err != nil {
			return nil, fmt.Errorf("replay line %d: %w", line, err)
		}
		if rec.CaseID == "" {
			return nil, fmt.Errorf("replay line %d: empty case_id", line)
		}
		d.byCase[rec.CaseID] = rec
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed to read replay file: %w", err)
	}
	return d, nil
}

// NewReplayDeciderFile reads a replay file from disk.
func NewReplayDeciderFile(path string) (*ReplayDecider, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open replay file: %w", err)
	}
	defer f.Close()
	return NewReplayDecider(f)
}

func (d *ReplayDecider) Decide(
	_ context.Context,
	caseID string,
	_ any,
	_ map[string]decisions.Question,
) (*Result, error) {
	rec, ok := d.byCase[caseID]
	if !ok {
		return nil, fmt.Errorf("%w: %q", ErrNoRecording, caseID)
	}
	return rec.result(), nil
}

// Recorder wraps a Decider and writes every successful answer to a replay file.
type Recorder struct {
	Inner Decider

	mu  sync.Mutex
	enc *json.Encoder
}

// NewRecorder returns a Recorder that appends JSONL recordings to w.
func NewRecorder(inner Decider, w io.Writer) *Recorder {
	return &Recorder{Inner: inner, enc: json.NewEncoder(w)}
}

func (r *Recorder) Decide(
	ctx context.Context,
	caseID string,
	state any,
	qs map[string]decisions.Question,
) (*Result, error) {
	res, err := r.Inner.Decide(ctx, caseID, state, qs)
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := r.enc.Encode(newRecording(caseID, res)); err != nil {
		return nil, fmt.Errorf("failed to write recording: %w", err)
	}
	return res, nil
}
