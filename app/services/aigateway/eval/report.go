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
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// WriteJSON writes the JSON summary of the report.
func (r *Report) WriteJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

// LoadReport reads a JSON summary written by WriteJSON, e.g. a baseline for Compare.
func LoadReport(path string) (*Report, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read report: %w", err)
	}
	var r Report
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("failed to parse report: %w", err)
	}
	return &r, nil
}

// Regression is an answer that was right in the baseline and is wrong now (or the reverse
// when Fixed is true).
type Regression struct {
	CaseID   string `json:"case_id"`
	Question string `json:"question"`
	Expected string `json:"expected"`
	Before   string `json:"before"`
	After    string `json:"after"`
	Fixed    bool   `json:"fixed,omitempty"`
}

// Compare lists the answers whose correctness changed between a baseline and a new report.
func Compare(base, cur *Report) []Regression {
	type key struct{ c, q string }
	was := map[key]Outcome{}
	for _, o := range base.Outcomes {
		was[key{o.CaseID, o.Question}] = o
	}
	var out []Regression
	for _, o := range cur.Outcomes {
		b, ok := was[key{o.CaseID, o.Question}]
		if !ok || b.Correct == o.Correct {
			continue
		}
		out = append(out, Regression{
			CaseID: o.CaseID, Question: o.Question, Expected: o.Expected,
			Before: b.Got, After: o.Got, Fixed: o.Correct,
		})
	}
	return out
}

func pct(f float64) string { return fmt.Sprintf("%.1f%%", f*100) }

func opt3(f *float64) string {
	if f == nil {
		return "n/a"
	}
	return fmt.Sprintf("%.3f", *f)
}

func (t Threshold) String() string {
	if !t.Found {
		return "none reaches target"
	}
	return fmt.Sprintf("%.2f (n=%d, acc %s)", t.Confidence, t.N, pct(t.Accuracy))
}

// WriteMarkdown writes the human readable report. baseline may be nil.
func (r *Report) WriteMarkdown(w io.Writer, baseline *Report) error {
	var b strings.Builder
	b.WriteString("# Mammoth decision evaluation\n\n")
	fmt.Fprintf(&b, "- Model build: %s\n", orNone(r.Model))
	fmt.Fprintf(&b, "- Cases: %d in scope, %d evaluated, %d skipped (no answer), %d failed\n",
		r.Dataset.Cases, r.Dataset.Evaluated, r.Dataset.Skipped, len(r.Errors))
	if r.Dataset.Unreviewed > 0 {
		fmt.Fprintf(&b, "- WARNING: %d of %d cases carry DRAFT labels that no human has reviewed; "+
			"treat every number below as provisional.\n", r.Dataset.Unreviewed, r.Dataset.Cases)
	}
	fmt.Fprintf(&b, "- Overall accuracy: %s (%d of %d answers)\n", pct(r.Overall.Accuracy), r.Overall.Correct, r.Overall.N)
	fmt.Fprintf(&b, "- Brier score (noul questions, lower is better): %s\n", opt3(r.Overall.Brier))
	fmt.Fprintf(&b, "- Expected calibration error: %.3f\n", r.Overall.ECE)
	fmt.Fprintf(&b, "- Target accuracy for thresholds: %s, minimum support %d\n\n", pct(r.Target), r.MinSupport)

	b.WriteString("## Accuracy per question\n\n")
	b.WriteString("| Question | N | Correct | Accuracy | Mean confidence | Brier | Suggested min confidence |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---|\n")
	for _, q := range r.Questions {
		fmt.Fprintf(&b, "| %s | %d | %d | %s | %.2f | %s | %s |\n",
			q.Question, q.N, q.Correct, pct(q.Accuracy), q.MeanConfidence, opt3(q.Brier), q.Suggestion)
	}
	fmt.Fprintf(&b, "| all questions | %d | %d | %s | | %s | %s |\n\n",
		r.Overall.N, r.Overall.Correct, pct(r.Overall.Accuracy), opt3(r.Overall.Brier), r.Overall.Threshold)

	b.WriteString("## Calibration\n\n")
	b.WriteString("A well calibrated model is right about as often as its confidence says " +
		"(confidence 0.8 should be right about 80% of the time).\n\n")
	b.WriteString("| Confidence | N | Correct | Accuracy | Mean confidence | Gap |\n|---|---:|---:|---:|---:|---:|\n")
	for _, k := range r.Calibrate {
		if k.N == 0 {
			fmt.Fprintf(&b, "| %s | 0 | 0 | | | |\n", k.Label)
			continue
		}
		fmt.Fprintf(&b, "| %s | %d | %d | %s | %.2f | %+.2f |\n",
			k.Label, k.N, k.Correct, pct(k.Accuracy), k.MeanConfidence, k.Accuracy-k.MeanConfidence)
	}
	b.WriteString("\n")

	var wrong []Outcome
	for _, o := range r.Outcomes {
		if !o.Correct {
			wrong = append(wrong, o)
		}
	}
	b.WriteString("## Wrong answers\n\n")
	if len(wrong) == 0 {
		b.WriteString("None.\n\n")
	} else {
		b.WriteString("| Case | Question | Expected | Got | Confidence |\n|---|---|---|---|---:|\n")
		for _, o := range wrong {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %.2f |\n", o.CaseID, o.Question, o.Expected, o.Got, o.Confidence)
		}
		b.WriteString("\n")
	}

	if len(r.Errors) > 0 {
		b.WriteString("## Cases that failed\n\n")
		for _, e := range r.Errors {
			fmt.Fprintf(&b, "- %s: %s\n", e.CaseID, e.Error)
		}
		b.WriteString("\n")
	}

	if baseline != nil {
		fmt.Fprintf(&b, "## Comparison with baseline (%s)\n\n", orNone(baseline.Model))
		fmt.Fprintf(&b, "Overall accuracy %s -> %s.\n\n", pct(baseline.Overall.Accuracy), pct(r.Overall.Accuracy))
		changes := Compare(baseline, r)
		if len(changes) == 0 {
			b.WriteString("No answer changed correctness.\n\n")
		} else {
			b.WriteString("| Case | Question | Expected | Before | After | Change |\n|---|---|---|---|---|---|\n")
			for _, c := range changes {
				kind := "regression"
				if c.Fixed {
					kind = "fixed"
				}
				fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s |\n", c.CaseID, c.Question, c.Expected, c.Before, c.After, kind)
			}
			b.WriteString("\n")
		}
	}

	_, err := io.WriteString(w, b.String())
	return err
}

func orNone(s string) string {
	if s == "" {
		return "unknown"
	}
	return s
}
