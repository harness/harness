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

// Package eval is the offline evaluation harness of the Mammoth decision questions (design
// section 5.5). It loads a dataset of human-labelled cases, asks the questions through a Decider
// (the real gateway, or a replay of recorded answers) and reports accuracy per question, a
// calibration table, the Brier score and suggested confidence thresholds.
package eval

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/harness/gitness/app/services/aigateway/decisions"
	"github.com/harness/gitness/app/services/aigateway/questions"
)

// Group is the question group of a case.
type Group string

const (
	GroupShield   Group = "shield"
	GroupInsights Group = "insights"
	GroupGeneric  Group = "generic"
)

var (
	// ErrInvalidCase is returned for a dataset case that is malformed or inconsistent.
	ErrInvalidCase = errors.New("invalid eval case")
	// ErrNoRecording is returned by a replay decider that has no answers for a case.
	ErrNoRecording = errors.New("no recorded answer for case")
)

// Case is one labelled example. One JSON object per line in the dataset file.
type Case struct {
	ID    string `json:"id"`
	Group Group  `json:"group"`
	// State is the typed state of the group: questions.ShieldState, InsightsState or GenericState.
	State json.RawMessage `json:"state"`
	// Expected maps a question id to the expected value: "true"/"false" for noul questions, the
	// option key for choice questions and the level for score questions. Questions that are not
	// listed are asked but not scored.
	Expected map[string]string `json:"expected"`
	Notes    string            `json:"notes,omitempty"`
	// Source says where the facts of the case come from (advisory id, pattern, synthetic).
	Source string `json:"source,omitempty"`
	// Reviewed is false for draft labels that no human has confirmed yet.
	Reviewed bool `json:"reviewed"`
}

// LoadCases reads a JSONL dataset. Blank lines are skipped; unknown fields are rejected so that
// typos in the dataset do not silently change what is measured.
func LoadCases(r io.Reader) ([]Case, error) {
	var cases []Case
	seen := map[string]bool{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for line := 1; sc.Scan(); line++ {
		raw := bytes.TrimSpace(sc.Bytes())
		if len(raw) == 0 {
			continue
		}
		var c Case
		dec := json.NewDecoder(bytes.NewReader(raw))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&c); err != nil {
			return nil, fmt.Errorf("%w: line %d: %w", ErrInvalidCase, line, err)
		}
		if c.ID == "" || seen[c.ID] {
			return nil, fmt.Errorf("%w: line %d: empty or duplicate id %q", ErrInvalidCase, line, c.ID)
		}
		seen[c.ID] = true
		if _, err := Prepare(c, questions.Limits{}); err != nil {
			return nil, fmt.Errorf("case %q: %w", c.ID, err)
		}
		cases = append(cases, c)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("failed to read dataset: %w", err)
	}
	return cases, nil
}

// LoadCasesFile reads a JSONL dataset from a file.
func LoadCasesFile(path string) ([]Case, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open dataset: %w", err)
	}
	defer f.Close()
	return LoadCases(f)
}

// Prepared is a case turned into a gateway request.
type Prepared struct {
	State     any
	Questions map[string]decisions.Question
	// Validate runs the typed parser of the group over a set of decisions.
	Validate func(map[string]decisions.Decision) error
}

func decodeState[T any](raw json.RawMessage) (*T, error) {
	var s T
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("%w: state: %w", ErrInvalidCase, err)
	}
	return &s, nil
}

// Prepare builds the batched questions of a case with the questions package and checks that
// every expected answer is a legal answer of its question.
func Prepare(c Case, l questions.Limits) (*Prepared, error) {
	if len(c.Expected) == 0 {
		return nil, fmt.Errorf("%w: no expected answers", ErrInvalidCase)
	}
	var p Prepared
	switch c.Group {
	case GroupShield:
		s, err := decodeState[questions.ShieldState](c.State)
		if err != nil {
			return nil, err
		}
		if err = s.Fit(l); err != nil {
			return nil, err
		}
		p.State, p.Questions = s, questions.ShieldQuestions()
		p.Validate = func(ds map[string]decisions.Decision) error {
			_, perr := questions.ParseShield(ds)
			return perr
		}
	case GroupInsights:
		s, err := decodeState[questions.InsightsState](c.State)
		if err != nil {
			return nil, err
		}
		if err = s.Fit(l); err != nil {
			return nil, err
		}
		if p.Questions, err = questions.InsightsQuestions(s); err != nil {
			return nil, err
		}
		p.State = s
		p.Validate = func(ds map[string]decisions.Decision) error {
			_, perr := questions.ParseInsights(s, ds)
			return perr
		}
	case GroupGeneric:
		s, err := decodeState[questions.GenericState](c.State)
		if err != nil {
			return nil, err
		}
		if err = s.Fit(l); err != nil {
			return nil, err
		}
		ask := questions.GenericAsk{
			WorthAgentBudget: has(c.Expected, questions.QWorthAgentBudget),
			NextStep:         has(c.Expected, questions.QNextStep),
			AssigneeKind:     has(c.Expected, questions.QAssigneeKind),
		}
		if p.Questions, err = questions.GenericQuestions(s, ask); err != nil {
			return nil, err
		}
		p.State = s
		p.Validate = func(ds map[string]decisions.Decision) error {
			_, perr := questions.ParseGeneric(s, ask, ds)
			return perr
		}
	default:
		return nil, fmt.Errorf("%w: unknown group %q", ErrInvalidCase, c.Group)
	}
	for id, want := range c.Expected {
		q, ok := p.Questions[id]
		if !ok {
			return nil, fmt.Errorf("%w: expected answer for unknown question %q", ErrInvalidCase, id)
		}
		if !legal(q, want) {
			return nil, fmt.Errorf("%w: %q is not a legal answer of question %q", ErrInvalidCase, want, id)
		}
	}
	return &p, nil
}

func has(m map[string]string, k string) bool {
	_, ok := m[k]
	return ok
}

func legal(q decisions.Question, v string) bool {
	switch c := q.Criteria.(type) {
	case map[string]string:
		_, ok := c[v]
		return ok
	case []string:
		for _, l := range c {
			if l == v {
				return true
			}
		}
		return false
	default:
		return v == "true" || v == "false"
	}
}

// DatasetStats counts the cases per group and how many labels are still drafts.
func DatasetStats(cases []Case) (perGroup map[Group]int, unreviewed int) {
	perGroup = map[Group]int{}
	for _, c := range cases {
		perGroup[c.Group]++
		if !c.Reviewed {
			unreviewed++
		}
	}
	return perGroup, unreviewed
}

// SortedGroups returns the groups of a count map in a stable order.
func SortedGroups(m map[Group]int) []Group {
	out := make([]Group, 0, len(m))
	for g := range m {
		out = append(out, g)
	}
	sort.Slice(out, func(i, j int) bool { return strings.Compare(string(out[i]), string(out[j])) < 0 })
	return out
}
