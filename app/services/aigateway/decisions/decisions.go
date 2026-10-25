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

// Package decisions holds the contract of the OpenRouter Decisions API (alpha),
// used by decision models such as Mammoth 1. The API is alpha, so everything that
// depends on its wire format lives here and is covered by contract tests.
package decisions

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
)

// Path is relative to the OpenRouter API base URL (https://openrouter.ai/api).
const Path = "/alpha/decisions"

type QuestionType string

const (
	TypeNoul   QuestionType = "noul"
	TypeChoice QuestionType = "choice"
	TypeScore  QuestionType = "score"
)

var (
	ErrInvalidRequest  = errors.New("invalid decisions request")
	ErrInvalidResponse = errors.New("invalid decisions response")
)

var questionIDRegex = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// Question is one decision. Criteria is map[string]string for noul and choice,
// and []string (lowest level first) for score.
type Question struct {
	Type         QuestionType `json:"type"`
	Instructions string       `json:"instructions"`
	Criteria     any          `json:"criteria,omitempty"`
}

func Noul(instructions, whenTrue, whenFalse string) Question {
	q := Question{Type: TypeNoul, Instructions: instructions}
	if whenTrue != "" || whenFalse != "" {
		q.Criteria = map[string]string{"true": whenTrue, "false": whenFalse}
	}
	return q
}

func Choice(instructions string, options map[string]string) Question {
	return Question{Type: TypeChoice, Instructions: instructions, Criteria: options}
}

func Score(instructions string, levels ...string) Question {
	return Question{Type: TypeScore, Instructions: instructions, Criteria: levels}
}

type Request struct {
	Model     string              `json:"model"`
	State     any                 `json:"state"`
	Questions map[string]Question `json:"questions"`
	SessionID string              `json:"session_id,omitempty"`
	User      string              `json:"user,omitempty"`
	Provider  any                 `json:"provider,omitempty"`
}

func (r *Request) Validate() error {
	if r.Model == "" {
		return fmt.Errorf("%w: model is required", ErrInvalidRequest)
	}
	if r.State == nil {
		return fmt.Errorf("%w: state is required", ErrInvalidRequest)
	}
	if len(r.Questions) == 0 {
		return fmt.Errorf("%w: at least one question is required", ErrInvalidRequest)
	}
	for id, q := range r.Questions {
		if !questionIDRegex.MatchString(id) {
			return fmt.Errorf("%w: invalid question id %q", ErrInvalidRequest, id)
		}
		if q.Instructions == "" {
			return fmt.Errorf("%w: question %q has no instructions", ErrInvalidRequest, id)
		}
		if err := validateCriteria(id, q); err != nil {
			return err
		}
	}
	return nil
}

func validateCriteria(id string, q Question) error {
	switch q.Type {
	case TypeNoul:
		if q.Criteria == nil {
			return nil
		}
		c, ok := q.Criteria.(map[string]string)
		if !ok || len(c) != 2 || c["true"] == "" || c["false"] == "" {
			return fmt.Errorf("%w: noul question %q needs exactly true and false criteria", ErrInvalidRequest, id)
		}
	case TypeChoice:
		c, ok := q.Criteria.(map[string]string)
		if !ok || len(c) < 2 {
			return fmt.Errorf("%w: choice question %q needs at least two options", ErrInvalidRequest, id)
		}
	case TypeScore:
		c, ok := q.Criteria.([]string)
		if !ok || len(c) < 2 {
			return fmt.Errorf("%w: score question %q needs at least two levels", ErrInvalidRequest, id)
		}
	default:
		return fmt.Errorf("%w: question %q has unknown type %q", ErrInvalidRequest, id, q.Type)
	}
	return nil
}

// Answer is the raw answer. Fields other than Type are pointers so that a missing key
// can be told apart from a zero value.
type Answer struct {
	Type          QuestionType       `json:"type"`
	Noul          *float64           `json:"noul,omitempty"`
	Choice        *string            `json:"choice,omitempty"`
	Score         *float64           `json:"score,omitempty"`
	Legend        map[string]any     `json:"legend,omitempty"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    *float64           `json:"confidence,omitempty"`
}

type Usage struct {
	InputTokens  int     `json:"input_tokens"`
	OutputTokens int     `json:"output_tokens"`
	Cost         float64 `json:"cost"`
}

type Response struct {
	ID       string            `json:"id"`
	Model    string            `json:"model"`
	Provider string            `json:"provider"`
	Answers  map[string]Answer `json:"answers"`
	Usage    Usage             `json:"usage"`
}

// Decision is a validated answer, normalized so policies can compare confidence against thresholds.
type Decision struct {
	QuestionID string
	Type       QuestionType
	// Value is "true"/"false" for noul, the chosen option for choice,
	// and the most likely level (as written in the criteria) for score.
	Value string
	// Probability is P(true) for noul.
	Probability float64
	// Score is the expected level index for score questions.
	Score         float64
	Probabilities map[string]float64
	Confidence    float64
}

// Parse validates the response against the request strictly: every question must be
// answered with the expected type and in-range values. It never fills in defaults.
func Parse(req *Request, resp *Response) (map[string]Decision, error) {
	if resp == nil {
		return nil, fmt.Errorf("%w: empty response", ErrInvalidResponse)
	}

	out := make(map[string]Decision, len(req.Questions))
	for id, q := range req.Questions {
		a, ok := resp.Answers[id]
		if !ok {
			return nil, fmt.Errorf("%w: missing answer for %q", ErrInvalidResponse, id)
		}
		if a.Type != q.Type {
			return nil, fmt.Errorf("%w: answer %q has type %q, expected %q", ErrInvalidResponse, id, a.Type, q.Type)
		}

		var (
			d   Decision
			err error
		)
		switch q.Type {
		case TypeNoul:
			d, err = parseNoul(id, a)
		case TypeChoice:
			d, err = parseChoice(id, q, a)
		case TypeScore:
			d, err = parseScore(id, q, a)
		}
		if err != nil {
			return nil, err
		}
		out[id] = d
	}
	return out, nil
}

func parseNoul(id string, a Answer) (Decision, error) {
	if a.Noul == nil || !isProbability(*a.Noul) {
		return Decision{}, fmt.Errorf("%w: noul answer %q is missing or out of range", ErrInvalidResponse, id)
	}
	p := *a.Noul
	value := "false"
	if p >= 0.5 {
		value = "true"
	}
	return Decision{
		QuestionID:    id,
		Type:          TypeNoul,
		Value:         value,
		Probability:   p,
		Probabilities: map[string]float64{"true": p, "false": 1 - p},
		Confidence:    math.Max(p, 1-p),
	}, nil
}

func parseChoice(id string, q Question, a Answer) (Decision, error) {
	options, _ := q.Criteria.(map[string]string)
	if a.Choice == nil {
		return Decision{}, fmt.Errorf("%w: choice answer %q is missing", ErrInvalidResponse, id)
	}
	if _, ok := options[*a.Choice]; !ok {
		return Decision{}, fmt.Errorf("%w: choice answer %q picked unknown option %q", ErrInvalidResponse, id, *a.Choice)
	}

	confidence, err := confidenceOf(id, a)
	if err != nil {
		return Decision{}, err
	}
	return Decision{
		QuestionID:    id,
		Type:          TypeChoice,
		Value:         *a.Choice,
		Probabilities: a.Probabilities,
		Confidence:    confidence,
	}, nil
}

func parseScore(id string, q Question, a Answer) (Decision, error) {
	levels, _ := q.Criteria.([]string)
	if a.Score == nil || *a.Score < 0 || *a.Score > float64(len(levels)-1) || math.IsNaN(*a.Score) {
		return Decision{}, fmt.Errorf("%w: score answer %q is missing or out of range", ErrInvalidResponse, id)
	}

	confidence, err := confidenceOf(id, a)
	if err != nil {
		return Decision{}, err
	}

	// most likely level: by probabilities when present, otherwise the rounded expected score
	idx := int(math.Round(*a.Score))
	if len(a.Probabilities) > 0 {
		keys := make([]string, 0, len(a.Probabilities))
		for k := range a.Probabilities {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		best := -1.0
		for _, k := range keys {
			i, err := strconv.Atoi(k)
			if err != nil || i < 0 || i >= len(levels) {
				return Decision{}, fmt.Errorf("%w: score answer %q has unknown level %q", ErrInvalidResponse, id, k)
			}
			if p := a.Probabilities[k]; p > best {
				best, idx = p, i
			}
		}
	}

	return Decision{
		QuestionID:    id,
		Type:          TypeScore,
		Value:         levels[idx],
		Score:         *a.Score,
		Probabilities: a.Probabilities,
		Confidence:    confidence,
	}, nil
}

// confidenceOf reads the optional confidence field, falling back to the highest probability.
func confidenceOf(id string, a Answer) (float64, error) {
	if a.Confidence != nil {
		if !isProbability(*a.Confidence) {
			return 0, fmt.Errorf("%w: answer %q has confidence out of range", ErrInvalidResponse, id)
		}
		return *a.Confidence, nil
	}
	best := 0.0
	for _, p := range a.Probabilities {
		if !isProbability(p) {
			return 0, fmt.Errorf("%w: answer %q has probability out of range", ErrInvalidResponse, id)
		}
		best = math.Max(best, p)
	}
	return best, nil
}

func isProbability(p float64) bool {
	return !math.IsNaN(p) && p >= 0 && p <= 1
}
