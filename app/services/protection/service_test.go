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
