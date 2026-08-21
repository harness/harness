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

package checkreq

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/google/go-cmp/cmp"
)

type stubProvider Requirements

func (s stubProvider) RequiredChecks(
	context.Context,
	*types.RepositoryCore,
	*types.PullReq,
) (Requirements, error) {
	return Requirements(s), nil
}

func resolve(t *testing.T, requirements Requirements) Resolved {
	t.Helper()
	resolved, err := Resolve(context.Background(), stubProvider(requirements), &types.RepositoryCore{}, &types.PullReq{})
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}
	return resolved
}

func bypassedBy(id int64) *int64 { return &id }

func TestResolvedUnsatisfied(t *testing.T) {
	tests := []struct {
		name         string
		requirements Requirements
		results      []types.CheckResult
		want         []string
	}{
		{
			name:         "no requirements",
			requirements: Requirements{},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusFailure}},
			want:         nil,
		},
		{
			name:         "required and successful",
			requirements: Requirements{Required: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusSuccess}},
			want:         nil,
		},
		{
			name:         "required and not reported",
			requirements: Requirements{Required: []string{"a"}},
			results:      nil,
			want:         []string{"a"},
		},
		{
			name:         "required and running",
			requirements: Requirements{Required: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusRunning}},
			want:         []string{"a"},
		},
		{
			name:         "non-bypassable is not satisfied by an explicit bypass",
			requirements: Requirements{Required: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusFailure, BypassedBy: bypassedBy(7)}},
			want:         []string{"a"},
		},
		{
			name:         "bypassable is satisfied by an explicit bypass",
			requirements: Requirements{Bypassable: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusFailure, BypassedBy: bypassedBy(7)}},
			want:         nil,
		},
		{
			name:         "bypassable and failed without a bypass",
			requirements: Requirements{Bypassable: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusFailure}},
			want:         []string{"a"},
		},
		{
			name:         "declared both ways is treated as non-bypassable",
			requirements: Requirements{Required: []string{"a"}, Bypassable: []string{"a"}},
			results:      []types.CheckResult{{Identifier: "a", Status: enum.CheckStatusFailure, BypassedBy: bypassedBy(7)}},
			want:         []string{"a"},
		},
		{
			name:         "unsatisfied are sorted",
			requirements: Requirements{Required: []string{"b"}, Bypassable: []string{"a", "c"}},
			results:      []types.CheckResult{{Identifier: "c", Status: enum.CheckStatusSuccess}},
			want:         []string{"a", "b"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolve(t, test.requirements).Unsatisfied(test.results)
			if diff := cmp.Diff(test.want, got); diff != "" {
				t.Errorf("Unsatisfied() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestResolvedRequirement(t *testing.T) {
	resolved := resolve(t, Requirements{Required: []string{"a"}, Bypassable: []string{"b"}})

	tests := []struct {
		identifier     string
		wantRequired   bool
		wantBypassable bool
	}{
		{identifier: "a", wantRequired: true, wantBypassable: false},
		{identifier: "b", wantRequired: true, wantBypassable: true},
		{identifier: "c", wantRequired: false, wantBypassable: false},
	}

	for _, test := range tests {
		t.Run(test.identifier, func(t *testing.T) {
			required, bypassable := resolved.Requirement(test.identifier)
			if required != test.wantRequired || bypassable != test.wantBypassable {
				t.Errorf("Requirement(%q) = (%t, %t), want (%t, %t)",
					test.identifier, required, bypassable, test.wantRequired, test.wantBypassable)
			}
		})
	}
}

func TestViolationsBlockEverybody(t *testing.T) {
	if got := Violations(nil); got != nil {
		t.Errorf("Violations(nil) = %v, want nil", got)
	}

	got := Violations([]string{"a", "b"})
	if len(got) != 1 {
		t.Fatalf("Violations() returned %d rule violations, want 1", len(got))
	}
	if got[0].Bypassable || got[0].Bypassed {
		t.Errorf("Violations() must not be bypassable: %+v", got[0])
	}
	if !got[0].IsCritical() {
		t.Error("Violations() must be critical so that the merge is blocked")
	}
	if len(got[0].Violations) != 1 || got[0].Violations[0].Code != CodeStatusChecksRequired {
		t.Errorf("unexpected violations: %+v", got[0].Violations)
	}
}

func TestResolveNilProvider(t *testing.T) {
	resolved, err := Resolve(context.Background(), nil, &types.RepositoryCore{}, &types.PullReq{})
	if err != nil {
		t.Fatalf("Resolve() returned error: %v", err)
	}
	if got := resolved.Unsatisfied([]types.CheckResult{{Identifier: "a"}}); got != nil {
		t.Errorf("Unsatisfied() = %v, want nil", got)
	}
	if got := resolved.Identifiers(); len(got) != 0 {
		t.Errorf("Identifiers() = %v, want empty", got)
	}
}

// The payload of a requirement describes the check the provider would report,
// so that a pending placeholder is not mistaken for a payload-less check.
func TestResolvedPayload(t *testing.T) {
	resolved := resolve(t, Requirements{
		Required:    []string{"a", "b"},
		PayloadKind: "agent_report",
		PayloadData: map[string]json.RawMessage{"a": json.RawMessage(`{"criterion_id":7}`)},
	})

	got := resolved.Payload("a")
	if got.Kind != "agent_report" || string(got.Data) != `{"criterion_id":7}` {
		t.Errorf("Payload(\"a\") = %+v", got)
	}

	// A requirement without payload data keeps the kind and an empty object.
	got = resolved.Payload("b")
	if got.Kind != "agent_report" || string(got.Data) != "{}" {
		t.Errorf("Payload(\"b\") = %+v", got)
	}
}
