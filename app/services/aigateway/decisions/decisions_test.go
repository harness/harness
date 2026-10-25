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

package decisions

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

// contractResponse is the response example of the Decisions API reference.
const contractResponse = `{
  "id": "gen-dec-1790265859-EaXKST7hul1Wcqots1ZK",
  "model": "baseten/mammoth-1-20261021",
  "provider": "Baseten",
  "answers": {
    "is_bug": { "type": "noul", "noul": 0.96 },
    "team": {
      "type": "choice",
      "choice": "payments",
      "probabilities": { "payments": 0.78, "frontend": 0.22, "account": 0, "none": 0 },
      "confidence": 0.67
    },
    "urgency": {
      "type": "score",
      "score": 1.99,
      "legend": { "0": "Can wait", "1": "This week", "2": "Blocking revenue" },
      "probabilities": { "0": 0, "1": 0, "2": 1 },
      "confidence": 0.99
    }
  },
  "usage": { "input_tokens": 476, "output_tokens": 70, "cost": 0 }
}`

func contractRequest() *Request {
	return &Request{
		Model: "baseten/mammoth-1:free",
		State: map[string]any{"ticket": "blank screen after clicking Pay"},
		Questions: map[string]Question{
			"is_bug": Noul("Is the customer reporting a software defect?", "broken behavior", "a question"),
			"team": Choice("Which team should own this ticket?", map[string]string{
				"payments": "Checkout issues",
				"frontend": "Rendering issues",
				"account":  "Login issues",
				"none":     "None of the above",
			}),
			"urgency": Score("How urgent is this ticket?", "Can wait", "This week", "Blocking revenue"),
		},
	}
}

func decode(t *testing.T, raw string) *Response {
	t.Helper()
	var resp Response
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	return &resp
}

func TestRequestWireFormat(t *testing.T) {
	raw, err := json.Marshal(contractRequest())
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}

	questions := got["questions"].(map[string]any)
	if _, ok := questions["urgency"].(map[string]any)["criteria"].([]any); !ok {
		t.Errorf("score criteria must be an array, got %v", questions["urgency"])
	}
	if _, ok := questions["team"].(map[string]any)["criteria"].(map[string]any); !ok {
		t.Errorf("choice criteria must be an object, got %v", questions["team"])
	}
	if _, ok := got["session_id"]; ok {
		t.Errorf("empty session_id must be omitted")
	}
}

func TestValidate(t *testing.T) {
	if err := contractRequest().Validate(); err != nil {
		t.Fatalf("contract request must be valid: %v", err)
	}

	tests := map[string]func(r *Request){
		"no model":          func(r *Request) { r.Model = "" },
		"no state":          func(r *Request) { r.State = nil },
		"no questions":      func(r *Request) { r.Questions = nil },
		"bad id":            func(r *Request) { r.Questions["Bad-ID"] = Noul("x", "", "") },
		"one option":        func(r *Request) { r.Questions["team"] = Choice("x", map[string]string{"a": "a"}) },
		"one level":         func(r *Request) { r.Questions["urgency"] = Score("x", "only") },
		"partial noul":      func(r *Request) { r.Questions["is_bug"] = Noul("x", "yes", "") },
		"unknown type":      func(r *Request) { r.Questions["x"] = Question{Type: "rank", Instructions: "x"} },
		"no instructions":   func(r *Request) { r.Questions["x"] = Noul("", "", "") },
		"wrong criteria go": func(r *Request) { r.Questions["x"] = Question{Type: TypeScore, Instructions: "x", Criteria: "a"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			r := contractRequest()
			mutate(r)
			if err := r.Validate(); !errors.Is(err, ErrInvalidRequest) {
				t.Errorf("expected ErrInvalidRequest, got %v", err)
			}
		})
	}
}

func TestParseContract(t *testing.T) {
	got, err := Parse(contractRequest(), decode(t, contractResponse))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bug := got["is_bug"]
	if bug.Value != "true" || bug.Probability != 0.96 || math.Abs(bug.Confidence-0.96) > 1e-9 {
		t.Errorf("unexpected noul decision: %+v", bug)
	}

	team := got["team"]
	if team.Value != "payments" || team.Confidence != 0.67 {
		t.Errorf("unexpected choice decision: %+v", team)
	}

	urgency := got["urgency"]
	if urgency.Value != "Blocking revenue" || urgency.Score != 1.99 || urgency.Confidence != 0.99 {
		t.Errorf("unexpected score decision: %+v", urgency)
	}
}

func TestParseNoulConfidenceIsSymmetric(t *testing.T) {
	req := &Request{Model: "m", State: "s", Questions: map[string]Question{"q": Noul("x", "", "")}}
	got, err := Parse(req, decode(t, `{"answers":{"q":{"type":"noul","noul":0.1}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got["q"].Value != "false" || math.Abs(got["q"].Confidence-0.9) > 1e-9 {
		t.Errorf("unexpected decision: %+v", got["q"])
	}
}

func TestParseStrict(t *testing.T) {
	tests := map[string]string{
		"missing answer": `{"answers":{
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
		"wrong type": `{"answers":{
			"is_bug":{"type":"choice","choice":"payments"},
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
		"missing noul value": `{"answers":{
			"is_bug":{"type":"noul"},
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
		"noul out of range": `{"answers":{
			"is_bug":{"type":"noul","noul":1.5},
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
		"unknown option": `{"answers":{
			"is_bug":{"type":"noul","noul":0.5},
			"team":{"type":"choice","choice":"billing","confidence":0.5},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
		"score out of range": `{"answers":{
			"is_bug":{"type":"noul","noul":0.5},
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":3,"confidence":0.5}}}`,
		"unknown level": `{"answers":{
			"is_bug":{"type":"noul","noul":0.5},
			"team":{"type":"choice","choice":"payments","confidence":0.5},
			"urgency":{"type":"score","score":1,"probabilities":{"7":1},"confidence":0.5}}}`,
		"confidence out of range": `{"answers":{
			"is_bug":{"type":"noul","noul":0.5},
			"team":{"type":"choice","choice":"payments","confidence":2},
			"urgency":{"type":"score","score":1,"confidence":0.5}}}`,
	}
	for name, raw := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Parse(contractRequest(), decode(t, raw)); !errors.Is(err, ErrInvalidResponse) {
				t.Errorf("expected ErrInvalidResponse, got %v", err)
			}
		})
	}
}

func TestParseConfidenceFallsBackToProbabilities(t *testing.T) {
	req := &Request{Model: "m", State: "s", Questions: map[string]Question{
		"q": Choice("x", map[string]string{"a": "a", "b": "b"}),
	}}
	got, err := Parse(req, decode(t, `{"answers":{"q":{"type":"choice","choice":"b","probabilities":{"a":0.3,"b":0.7}}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if got["q"].Confidence != 0.7 {
		t.Errorf("expected confidence 0.7, got %v", got["q"].Confidence)
	}
}
