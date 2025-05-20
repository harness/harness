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
	"context"
	"testing"

	"github.com/harness/gitness/types"
)

func TestTagRuleSet_SetRefChangeVerify(t *testing.T) {
	tests := []struct {
		name    string
		rules   []types.RuleInfoInternal
		input   RefChangeVerifyInput
		expViol []types.RuleViolations
	}{
		{
			name:  "empty-with-action-create",
			rules: []types.RuleInfoInternal{},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionCreate,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-a"},
			},
			expViol: []types.RuleViolations{},
		},
		{
			name:  "empty-with-action-delete",
			rules: []types.RuleInfoInternal{},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionDelete,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-a"},
			},
			expViol: []types.RuleViolations{},
		},
		{
			name: "create-forbidden-with-pattern-and-matching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"create_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionCreate,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-a"},
			},
			expViol: []types.RuleViolations{
				{
					Rule: types.RuleInfo{Type: "tag"},
					Violations: []types.Violation{
						{Code: codeLifecycleCreate},
					},
				},
			},
		},
		{
			name: "create-forbidden-with-pattern-and-mismatching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"create_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionCreate,
				RefType:   RefTypeTag,
				RefNames:  []string{"dev-a"},
			},
			expViol: []types.RuleViolations{},
		},
		{
			name: "delete-forbidden-with-pattern-and-matching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"delete_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionDelete,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-a"},
			},
			expViol: []types.RuleViolations{
				{
					Rule: types.RuleInfo{Type: "tag"},
					Violations: []types.Violation{
						{Code: codeLifecycleDelete},
					},
				},
			},
		},
		{
			name: "delete-forbidden-with-pattern-and-mismatching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo: types.RuleInfo{
						Type: TypeTag,
					},
					Definition: []byte(`{"lifecycle": {"delete_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionDelete,
				RefType:   RefTypeTag,
				RefNames:  []string{"dev-a"},
			},
			expViol: []types.RuleViolations{},
		},
		{
			name: "create-forbidden-with-two-rules-and-pattern-and-matching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"create_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"create_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["*-experimental"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionCreate,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-experimental"},
			},
			expViol: []types.RuleViolations{
				{
					Rule: types.RuleInfo{Type: "tag"},
					Violations: []types.Violation{
						{Code: codeLifecycleCreate},
					},
				},
				{
					Rule: types.RuleInfo{Type: "tag"},
					Violations: []types.Violation{
						{Code: codeLifecycleCreate},
					},
				},
			},
		},
		{
			name: "delete-forbidden-with-two-rules-and-pattern-and-matching-ref",
			rules: []types.RuleInfoInternal{
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"create_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["feat-*"]}`),
				},
				{
					RuleInfo:   types.RuleInfo{Type: TypeTag},
					Definition: []byte(`{"lifecycle": {"delete_forbidden": true}}`),
					Pattern:    []byte(`{"include": ["*-experimental"]}`),
				},
			},
			input: RefChangeVerifyInput{
				Actor:     &types.Principal{ID: 1},
				RefAction: RefActionDelete,
				RefType:   RefTypeTag,
				RefNames:  []string{"feat-experimental"},
			},
			expViol: []types.RuleViolations{
				{
					Rule: types.RuleInfo{Type: "tag"},
					Violations: []types.Violation{
						{Code: codeLifecycleDelete},
					},
				},
			},
		},
	}

	ctx := context.Background()

	m := NewManager(nil)
	_ = m.Register(TypeTag, func() Definition {
		return &Tag{}
	})

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			set := tagRuleSet{
				rules:   test.rules,
				manager: m,
			}

			violations, err := set.RefChangeVerify(ctx, test.input)
			if err != nil {
				t.Errorf("got error: %s", err.Error())
			}

			if want, got := len(test.expViol), len(violations); want != got {
				t.Errorf("violations count: want=%d got=%d", want, got)
				return
			}

			for i := range test.expViol {
				if want, got := test.expViol[i].Rule, violations[i].Rule; want != got {
					t.Errorf("violation %d rule: want=%+v got=%+v", i, want, got)
				}

				if want, got := test.expViol[i].Bypassed, violations[i].Bypassed; want != got {
					t.Errorf("violation %d bypassed: want=%t got=%t", i, want, got)
				}

				if want, got := len(test.expViol[i].Violations), len(violations[i].Violations); want != got {
					t.Errorf("violation %d violations count: want=%d got=%d", i, want, got)
					continue
				}

				for j := range test.expViol[i].Violations {
					if want, got := test.expViol[i].Violations[j].Code, violations[i].Violations[j].Code; want != got {
						t.Errorf("violation %d violation %d code: want=%s got=%s", i, j, want, got)
					}
				}
			}
		})
	}
}
