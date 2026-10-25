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
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
)

const (
	// DefaultTargetAccuracy is the accuracy a threshold suggestion must reach.
	DefaultTargetAccuracy = 0.9
	// DefaultMinSupport is the fewest answers a suggested threshold must be based on.
	DefaultMinSupport = 3
)

// Options controls a run.
type Options struct {
	// Groups limits the run to some groups; empty means all.
	Groups []Group
	// Limits is passed to the state compaction of the questions package.
	Limits questions.Limits
	// TargetAccuracy for the threshold suggestion; zero means DefaultTargetAccuracy.
	TargetAccuracy float64
	// MinSupport for the threshold suggestion; zero means DefaultMinSupport.
	MinSupport int
}

// Outcome is the scored answer to one expected question of one case.
type Outcome struct {
	CaseID   string `json:"case_id"`
	Group    Group  `json:"group"`
	Question string `json:"question"`
	Expected string `json:"expected"`
	Got      string `json:"got"`
	Correct  bool   `json:"correct"`
	// Confidence is the confidence reported by the model for its own answer.
	Confidence float64 `json:"confidence"`
	// Probability is P(true) for noul questions and zero otherwise.
	Probability float64 `json:"probability,omitempty"`
	Noul        bool    `json:"noul,omitempty"`
	Reviewed    bool    `json:"reviewed"`
}

// CaseError is a case that could not be evaluated.
type CaseError struct {
	CaseID string `json:"case_id"`
	Error  string `json:"error"`
}

// Run asks the questions of every case and scores the answers against the labels.
// Cases the decider has no answer for (ErrNoRecording) are skipped; other failures are
// listed in the report and do not stop the run, unless the context is done.
func Run(ctx context.Context, cases []Case, d Decider, opt Options) (*Report, error) {
	if opt.TargetAccuracy <= 0 || opt.TargetAccuracy > 1 {
		opt.TargetAccuracy = DefaultTargetAccuracy
	}
	if opt.MinSupport <= 0 {
		opt.MinSupport = DefaultMinSupport
	}
	groups := map[Group]bool{}
	for _, g := range opt.Groups {
		groups[g] = true
	}

	rep := &Report{Target: opt.TargetAccuracy, MinSupport: opt.MinSupport}
	models := map[string]bool{}
	for _, c := range cases {
		if len(groups) > 0 && !groups[c.Group] {
			continue
		}
		rep.Dataset.Cases++
		if !c.Reviewed {
			rep.Dataset.Unreviewed++
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		p, err := Prepare(c, opt.Limits)
		if err != nil {
			rep.Errors = append(rep.Errors, CaseError{CaseID: c.ID, Error: err.Error()})
			continue
		}
		res, err := d.Decide(ctx, c.ID, p.State, p.Questions)
		if err != nil {
			if errors.Is(err, ErrNoRecording) {
				rep.Dataset.Skipped++
				continue
			}
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			rep.Errors = append(rep.Errors, CaseError{CaseID: c.ID, Error: err.Error()})
			continue
		}
		if err := p.Validate(res.Decisions); err != nil {
			rep.Errors = append(rep.Errors, CaseError{CaseID: c.ID, Error: err.Error()})
			continue
		}
		rep.Dataset.Evaluated++
		if res.Model != "" {
			models[res.Model] = true
		}
		ids := make([]string, 0, len(c.Expected))
		for id := range c.Expected {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			dec := res.Decisions[id]
			o := Outcome{
				CaseID: c.ID, Group: c.Group, Question: id, Expected: c.Expected[id], Got: dec.Value,
				Correct: dec.Value == c.Expected[id], Confidence: dec.Confidence, Reviewed: c.Reviewed,
			}
			if dec.Type == decisions.TypeNoul {
				o.Noul, o.Probability = true, dec.Probability
			}
			rep.Outcomes = append(rep.Outcomes, o)
		}
	}
	for m := range models {
		rep.Models = append(rep.Models, m)
	}
	sort.Strings(rep.Models)
	rep.Model = strings.Join(rep.Models, ", ")
	rep.finish()
	return rep, nil
}

// StatKey maps a question id to the key its statistics are grouped under: the candidate
// duplicate questions share one key.
func StatKey(id string) string {
	if strings.HasPrefix(id, questions.QDuplicatePrefix) {
		return questions.QDuplicatePrefix + "n"
	}
	return id
}

// Threshold is the lowest confidence from which the answers reach the target accuracy.
type Threshold struct {
	Found bool `json:"found"`
	// Confidence is the minimum confidence; answers at or above it were right Accuracy of the time.
	Confidence float64 `json:"confidence,omitempty"`
	N          int     `json:"n,omitempty"`
	Accuracy   float64 `json:"accuracy,omitempty"`
}

// SuggestThreshold returns the minimum confidence c such that the outcomes with
// confidence >= c have accuracy >= target over at least minSupport answers. The result is
// meant to feed the High/Low thresholds of the policies; with few labelled cases it is only a hint.
func SuggestThreshold(outs []Outcome, target float64, minSupport int) Threshold {
	sorted := append([]Outcome(nil), outs...)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Confidence > sorted[j].Confidence })
	var best Threshold
	n, ok := 0, 0
	for i, o := range sorted {
		n++
		if o.Correct {
			ok++
		}
		last := i == len(sorted)-1 || sorted[i+1].Confidence != o.Confidence
		if !last || n < minSupport {
			continue
		}
		acc := float64(ok) / float64(n)
		if acc+1e-9 >= target {
			best = Threshold{Found: true, Confidence: o.Confidence, N: n, Accuracy: acc}
		}
	}
	return best
}

// QuestionStats are the scores of one question.
type QuestionStats struct {
	Question       string  `json:"question"`
	N              int     `json:"n"`
	Correct        int     `json:"correct"`
	Accuracy       float64 `json:"accuracy"`
	MeanConfidence float64 `json:"mean_confidence"`
	// Brier is the Brier score of P(true) for noul questions; nil for the other types.
	Brier      *float64  `json:"brier,omitempty"`
	Suggestion Threshold `json:"suggested_threshold"`
}

// Bucket is one row of the calibration table.
type Bucket struct {
	Label          string  `json:"label"`
	Low            float64 `json:"low"`
	High           float64 `json:"high"`
	N              int     `json:"n"`
	Correct        int     `json:"correct"`
	Accuracy       float64 `json:"accuracy"`
	MeanConfidence float64 `json:"mean_confidence"`
}

// Overall is the score over all questions.
type Overall struct {
	N        int     `json:"n"`
	Correct  int     `json:"correct"`
	Accuracy float64 `json:"accuracy"`
	// Brier is the Brier score over all noul answers; nil when there are none.
	Brier *float64 `json:"brier,omitempty"`
	// ECE is the expected calibration error: bucket weighted mean of |accuracy - mean confidence|.
	ECE       float64   `json:"ece"`
	Threshold Threshold `json:"suggested_threshold"`
}

// DatasetInfo counts the cases of a run.
type DatasetInfo struct {
	Cases      int `json:"cases"`
	Evaluated  int `json:"evaluated"`
	Skipped    int `json:"skipped"`
	Unreviewed int `json:"unreviewed"`
}

// Report is the result of a run. It is also the JSON summary.
type Report struct {
	// Model is the model build(s) that answered, joined with a comma.
	Model      string          `json:"model"`
	Models     []string        `json:"models"`
	Target     float64         `json:"target_accuracy"`
	MinSupport int             `json:"min_support"`
	Dataset    DatasetInfo     `json:"dataset"`
	Overall    Overall         `json:"overall"`
	Questions  []QuestionStats `json:"questions"`
	Calibrate  []Bucket        `json:"calibration"`
	Errors     []CaseError     `json:"errors,omitempty"`
	Outcomes   []Outcome       `json:"outcomes"`
}

var bucketEdges = []float64{0, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}

func bucketIndex(conf float64) int {
	for i := len(bucketEdges) - 2; i >= 0; i-- {
		if conf >= bucketEdges[i] {
			return i
		}
	}
	return 0
}

func brier(outs []Outcome) *float64 {
	var sum float64
	n := 0
	for _, o := range outs {
		if !o.Noul {
			continue
		}
		y := 0.0
		if o.Expected == "true" {
			y = 1
		}
		sum += (o.Probability - y) * (o.Probability - y)
		n++
	}
	if n == 0 {
		return nil
	}
	b := sum / float64(n)
	return &b
}

func ratio(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b)
}

func (r *Report) finish() {
	byQ := map[string][]Outcome{}
	buckets := make([]Bucket, len(bucketEdges)-1)
	sums := make([]float64, len(buckets))
	for i := range buckets {
		lo, hi := bucketEdges[i], bucketEdges[i+1]
		buckets[i] = Bucket{Low: lo, High: hi, Label: fmt.Sprintf("%.1f-%.1f", lo, hi)}
		if i == 0 {
			buckets[i].Label = "<0.5"
		}
	}
	var confSum float64
	for _, o := range r.Outcomes {
		k := StatKey(o.Question)
		byQ[k] = append(byQ[k], o)
		i := bucketIndex(o.Confidence)
		buckets[i].N++
		sums[i] += o.Confidence
		confSum += o.Confidence
		if o.Correct {
			buckets[i].Correct++
			r.Overall.Correct++
		}
	}
	r.Overall.N = len(r.Outcomes)
	r.Overall.Accuracy = ratio(r.Overall.Correct, r.Overall.N)
	r.Overall.Brier = brier(r.Outcomes)
	r.Overall.Threshold = SuggestThreshold(r.Outcomes, r.Target, r.MinSupport)
	for i := range buckets {
		b := &buckets[i]
		if b.N == 0 {
			continue
		}
		b.Accuracy = ratio(b.Correct, b.N)
		b.MeanConfidence = sums[i] / float64(b.N)
		r.Overall.ECE += float64(b.N) / float64(r.Overall.N) * math.Abs(b.Accuracy-b.MeanConfidence)
	}
	r.Calibrate = buckets

	keys := make([]string, 0, len(byQ))
	for k := range byQ {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		outs := byQ[k]
		qs := QuestionStats{Question: k, N: len(outs), Brier: brier(outs)}
		var cs float64
		for _, o := range outs {
			if o.Correct {
				qs.Correct++
			}
			cs += o.Confidence
		}
		qs.Accuracy = ratio(qs.Correct, qs.N)
		qs.MeanConfidence = cs / float64(qs.N)
		qs.Suggestion = SuggestThreshold(outs, r.Target, r.MinSupport)
		r.Questions = append(r.Questions, qs)
	}
}
