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

package types

import (
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types/enum"

	"gopkg.in/yaml.v3"
)

type Rule struct {
	ID      int64 `json:"-"`
	Version int64 `json:"-"`

	CreatedBy int64 `json:"-"`
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`

	RepoID  *int64 `json:"-"`
	SpaceID *int64 `json:"-"`

	Identifier  string `json:"identifier"`
	Description string `json:"description"`

	Type  RuleType       `json:"type"`
	State enum.RuleState `json:"state"`

	Pattern    json.RawMessage `json:"pattern"`
	Definition json.RawMessage `json:"definition"`

	CreatedByInfo PrincipalInfo `json:"created_by"`

	Users map[int64]*PrincipalInfo `json:"users"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (r Rule) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias Rule
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(r),
		UID:   r.Identifier,
	})
}

func (r Rule) MarshalYAML() (interface{}, error) {
	// yaml cannot marshal json.RawMessage
	pattern := make(map[string]any)
	err := yaml.Unmarshal(r.Pattern, pattern)
	if err != nil {
		return nil, err
	}
	definition := make(map[string]any)
	err = yaml.Unmarshal(r.Definition, definition)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":          r.ID,
		"created":     r.Created,
		"updated":     r.Updated,
		"created_by":  r.CreatedBy,
		"identifier":  r.Identifier,
		"description": r.Description,
		"type":        r.Type,
		"state":       r.State,
		"pattern":     pattern,
		"definition":  definition,
	}, nil
}

// Clone makes deep copy of the rule object.
func (r Rule) Clone() Rule {
	var repoID *int64
	var spaceID *int64

	if r.RepoID != nil {
		id := *r.RepoID
		repoID = &id
	}

	if r.SpaceID != nil {
		id := *r.SpaceID
		spaceID = &id
	}

	r.RepoID = repoID
	r.SpaceID = spaceID

	pattern := make(json.RawMessage, len(r.Pattern))
	copy(pattern, r.Pattern)
	r.Pattern = pattern

	definition := make(json.RawMessage, len(r.Definition))
	copy(definition, r.Definition)
	r.Definition = definition

	users := make(map[int64]*PrincipalInfo, len(r.Users))
	for key, value := range r.Users {
		cloned := *value
		users[key] = &cloned
	}
	r.Users = users

	return r
}

type RuleType string

type RuleFilter struct {
	ListQueryFilter
	States []enum.RuleState
	Sort   enum.RuleSort `json:"sort"`
	Order  enum.Order    `json:"order"`
}

// Violation represents a single violation.
type Violation struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Params  []any  `json:"params"`
}

// RuleViolations holds several violations of a rule.
type RuleViolations struct {
	Rule       RuleInfo    `json:"rule"`
	Bypassable bool        `json:"bypassable"`
	Bypassed   bool        `json:"bypassed"`
	Violations []Violation `json:"violations"`
}

func (violations *RuleViolations) Add(code, message string) {
	violations.Violations = append(violations.Violations, Violation{
		Code:    code,
		Message: message,
		Params:  nil,
	})
}

func (violations *RuleViolations) Addf(code, format string, params ...any) {
	violations.Violations = append(violations.Violations, Violation{
		Code:    code,
		Message: fmt.Sprintf(format, params...),
		Params:  params,
	})
}

func (violations *RuleViolations) IsCritical() bool {
	return violations.Rule.State == enum.RuleStateActive && len(violations.Violations) > 0 && !violations.Bypassed
}

func (violations *RuleViolations) IsBypassed() bool {
	return violations.Rule.State == enum.RuleStateActive && len(violations.Violations) > 0 && violations.Bypassed
}

// RuleInfo holds basic info about a rule that is used to describe the rule in RuleViolations.
type RuleInfo struct {
	SpacePath string `json:"space_path,omitempty"`
	RepoPath  string `json:"repo_path,omitempty"`

	ID         int64          `json:"-"`
	Identifier string         `json:"identifier"`
	Type       RuleType       `json:"type"`
	State      enum.RuleState `json:"state"`
}

// TODO [CODE-1363]: remove after identifier migration.
func (r RuleInfo) MarshalJSON() ([]byte, error) {
	// alias allows us to embed the original object while avoiding an infinite loop of marshaling.
	type alias RuleInfo
	return json.Marshal(&struct {
		alias
		UID string `json:"uid"`
	}{
		alias: (alias)(r),
		UID:   r.Identifier,
	})
}

type RuleInfoInternal struct {
	RuleInfo
	Pattern    json.RawMessage
	Definition json.RawMessage
}

type RulesViolations struct {
	Message    string           `json:"message"`
	Violations []RuleViolations `json:"violations"`
}

type DryRunRulesOutput struct {
	DryRunRules    bool             `json:"dry_run_rules,omitempty"`
	RuleViolations []RuleViolations `json:"rule_violations,omitempty"`
}
