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

// Package sast parses SARIF 2.1.0 output of static analysis tools (Opengrep, Semgrep)
// and maps the results to Shield findings of kind sast.
//
// The package has no database, HTTP or wire dependencies.
package sast

import (
	"encoding/json"
	"fmt"
	"io"
)

// Report is the lenient subset of a SARIF 2.1.0 log used by Shield.
type Report struct {
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

// Run is a single tool invocation.
type Run struct {
	Tool    Tool     `json:"tool"`
	Results []Result `json:"results"`
}

// Tool describes the analysis tool.
type Tool struct {
	Driver Driver `json:"driver"`
}

// Driver is the tool component with its rule metadata.
type Driver struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Rules   []Rule `json:"rules"`
}

// Text is a SARIF message-like object.
type Text struct {
	Text     string `json:"text"`
	Markdown string `json:"markdown"`
}

// Rule is a SARIF reportingDescriptor.
type Rule struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	ShortDescription Text       `json:"shortDescription"`
	FullDescription  Text       `json:"fullDescription"`
	Help             Text       `json:"help"`
	Properties       Properties `json:"properties"`
}

// Properties is the property bag of rules and results. All fields are optional.
type Properties struct {
	Tags             []string
	Precision        string
	SecuritySeverity *float64
	CWE              []string
}

// UnmarshalJSON tolerates security-severity as string or number and cwe as string or list.
// A malformed property bag never fails the whole report.
func (p *Properties) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil //nolint:nilerr
	}
	if v, ok := raw["tags"]; ok {
		p.Tags = stringOrList(v)
	}
	if v, ok := raw["precision"]; ok {
		var s string
		_ = json.Unmarshal(v, &s)
		p.Precision = s
	}
	if v, ok := raw["cwe"]; ok {
		p.CWE = stringOrList(v)
	}
	if v, ok := raw["security-severity"]; ok {
		p.SecuritySeverity = numberOrString(v)
	}
	return nil
}

func stringOrList(raw json.RawMessage) []string {
	var l []string
	if json.Unmarshal(raw, &l) == nil {
		return l
	}
	var s string
	if json.Unmarshal(raw, &s) == nil && s != "" {
		return []string{s}
	}
	return nil
}

func numberOrString(raw json.RawMessage) *float64 {
	var f float64
	if json.Unmarshal(raw, &f) == nil {
		return &f
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		if _, err := fmt.Sscanf(s, "%g", &f); err == nil {
			return &f
		}
	}
	return nil
}

// Result is a single SARIF result.
type Result struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           *int              `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             Text              `json:"message"`
	Locations           []Location        `json:"locations"`
	Fingerprints        map[string]string `json:"fingerprints"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Properties          Properties        `json:"properties"`
}

// Location is a SARIF location.
type Location struct {
	PhysicalLocation PhysicalLocation `json:"physicalLocation"`
}

// PhysicalLocation points to a file region.
type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           Region           `json:"region"`
}

// ArtifactLocation holds the file URI.
type ArtifactLocation struct {
	URI string `json:"uri"`
}

// Region is a text region; Snippet is optional.
type Region struct {
	StartLine   int  `json:"startLine"`
	StartColumn int  `json:"startColumn"`
	EndLine     int  `json:"endLine"`
	EndColumn   int  `json:"endColumn"`
	Snippet     Text `json:"snippet"`
}

// ParseSARIF decodes a SARIF log. Unknown and missing optional fields are ignored.
func ParseSARIF(r io.Reader) (*Report, error) {
	var rep Report
	if err := json.NewDecoder(r).Decode(&rep); err != nil {
		return nil, fmt.Errorf("failed to decode SARIF: %w", err)
	}
	return &rep, nil
}
