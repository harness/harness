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

package protection

import (
	"encoding/json"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func TestIsCritical(t *testing.T) {
	tests := []struct {
		name  string
		input []types.RuleViolations
		exp   bool
	}{
		{
			name:  "empty",
			input: []types.RuleViolations{},
			exp:   false,
		},
		{
			name: "non-critical",
			input: []types.RuleViolations{
				{
					Rule:       types.RuleInfo{State: enum.RuleStateMonitor},
					Bypassed:   false,
					Violations: []types.Violation{{Code: "x"}, {Code: "x"}},
				},
				{
					Rule:       types.RuleInfo{State: enum.RuleStateActive},
					Bypassed:   true,
					Violations: []types.Violation{{Code: "x"}, {Code: "x"}},
				},
				{
					Rule:       types.RuleInfo{State: enum.RuleStateActive},
					Bypassed:   false,
					Violations: []types.Violation{},
				},
			},
			exp: false,
		},
		{
			name: "critical",
			input: []types.RuleViolations{
				{
					Rule:       types.RuleInfo{State: enum.RuleStateActive},
					Bypassed:   false,
					Violations: []types.Violation{{Code: "x"}, {Code: "x"}},
				},
			},
			exp: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if want, got := test.exp, IsCritical(test.input); want != got {
				t.Errorf("want=%t got=%t", want, got)
			}
		})
	}
}

func TestManager_SanitizeJSON(t *testing.T) {
	tests := []struct {
		name      string
		ruleTypes []types.RuleType
		ruleType  types.RuleType
		errReg    error
		errSan    error
	}{
		{
			name:      "success",
			ruleTypes: []types.RuleType{TypeBranch},
			ruleType:  TypeBranch,
		},
		{
			name:      "duplicate",
			ruleTypes: []types.RuleType{TypeBranch, TypeBranch},
			ruleType:  TypeBranch,
			errReg:    ErrAlreadyRegistered,
		},
		{
			name:      "unregistered",
			ruleTypes: []types.RuleType{},
			ruleType:  TypeBranch,
			errSan:    ErrUnrecognizedType,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := NewManager(nil)

			err := func() error {
				for _, ruleType := range test.ruleTypes {
					err := m.Register(ruleType, func() Definition { return &Branch{} })
					if err != nil {
						return err
					}
				}
				return nil
			}()
			// nolint:errorlint // deliberately comparing errors with ==
			if test.errReg != err {
				t.Errorf("register type error mismatch: want=%v got=%v", test.errReg, err)
				return
			}

			_, err = m.SanitizeJSON(test.ruleType, json.RawMessage("{}"))
			// nolint:errorlint // deliberately comparing errors with ==
			if test.errSan != err {
				t.Errorf("register type error mismatch: want=%v got=%v", test.errSan, err)
				return
			}
		})
	}
}

func TestGenerateErrorMessageForBlockingViolations(t *testing.T) {
	type testCase struct {
		name       string
		violations []types.RuleViolations
		expected   string
	}

	tests := []testCase{
		{
			name:       "no violations",
			violations: nil,
			expected:   "No blocking rule violations found.",
		},
		{
			name: "no blocking violations",
			violations: []types.RuleViolations{
				{
					Bypassed: true,
				},
				{
					Rule: types.RuleInfo{
						State: enum.RuleStateDisabled,
					},
				},
				{
					Rule: types.RuleInfo{
						State: enum.RuleStateMonitor,
					},
				},
			},
			expected: "No blocking rule violations found.",
		},
		{
			name: "single violation without details",
			violations: []types.RuleViolations{
				{
					Rule: types.RuleInfo{
						Identifier: "rule1",
						State:      enum.RuleStateActive,
						Type:       "branch",
						SpacePath:  "space/path1",
					},
				},
			},
			expected: `Operation violates branch protection rule "rule1" in scope "space/path1"`,
		},
		{
			name: "multiple violations without details",
			violations: []types.RuleViolations{
				{
					Rule: types.RuleInfo{
						Identifier: "rule1",
						State:      enum.RuleStateActive,
						Type:       "branch",
						SpacePath:  "space/path1",
					},
				},
				{
					Rule: types.RuleInfo{
						Identifier: "rule2",
						State:      enum.RuleStateActive,
						Type:       "other",
						SpacePath:  "space/path2",
					},
				},
			},
			expected: `Operation violates 2 protection rules, including branch protection rule "rule1" ` +
				`in scope "space/path1"`,
		}, {
			name: "single violation with details",
			violations: []types.RuleViolations{
				{
					Rule: types.RuleInfo{
						Identifier: "rule1",
						State:      enum.RuleStateActive,
						Type:       "branch",
						RepoPath:   "repo/path1",
					},
					Violations: []types.Violation{
						{
							Message: "violation1.1",
						},
						{
							Message: "violation1.2",
						},
					},
				},
			},
			expected: `Operation violates branch protection rule "rule1" ` +
				`in repository "repo/path1" with violation: violation1.1`,
		},
		{
			name: "multiple violations with details",
			violations: []types.RuleViolations{
				{
					Rule: types.RuleInfo{
						Identifier: "rule1",
						State:      enum.RuleStateActive,
						Type:       "other",
						RepoPath:   "repo/path1",
					},
				},
				{
					Rule: types.RuleInfo{
						Identifier: "rule2",
						State:      enum.RuleStateActive,
						Type:       "branch",
						RepoPath:   "repo/path2",
					},
					Violations: []types.Violation{
						{
							Message: "violation2.1",
						},
						{
							Message: "violation2.2",
						},
					},
				},
				{
					Rule: types.RuleInfo{
						Identifier: "rule3",
						State:      enum.RuleStateActive,
						Type:       "other",
						RepoPath:   "repo/path3",
					},
					Violations: []types.Violation{
						{
							Message: "violation3.1",
						},
					},
				},
			},
			expected: `Operation violates 3 protection rules, including branch protection rule "rule2" ` +
				`in repository "repo/path2" with violation: violation2.1`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateErrorMessageForBlockingViolations(tt.violations)
			if got != tt.expected {
				t.Errorf("Want error message %q, got %q", tt.expected, got)
			}
		})
	}
}
