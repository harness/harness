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
	"reflect"
	"testing"

	"github.com/harness/gitness/types"
)

// nolint:gocognit // it's a unit test
func TestDefLifecycle_RefChangeVerify(t *testing.T) {
	const refName = "a"
	tests := []struct {
		name      string
		def       DefLifecycle
		action    RefAction
		expCodes  []string
		expParams [][]any
	}{
		{
			name: "empty",
		},
		{
			name:      "lifecycle.create-fail",
			def:       DefLifecycle{CreateForbidden: true},
			action:    RefActionCreate,
			expCodes:  []string{"lifecycle.create"},
			expParams: [][]any{{refName}},
		},
		{
			name:      "lifecycle.delete-fail",
			def:       DefLifecycle{DeleteForbidden: true},
			action:    RefActionDelete,
			expCodes:  []string{"lifecycle.delete"},
			expParams: [][]any{{refName}},
		},
		{
			name:      "lifecycle.update-fail",
			def:       DefLifecycle{UpdateForbidden: true},
			action:    RefActionUpdate,
			expCodes:  []string{"lifecycle.update"},
			expParams: [][]any{{refName}},
		},
		{
			name:      "lifecycle.update.force-fail",
			def:       DefLifecycle{UpdateForceForbidden: true},
			action:    RefActionUpdateForce,
			expCodes:  []string{"lifecycle.update.force"},
			expParams: [][]any{{refName}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			in := RefChangeVerifyInput{
				RefNames:  []string{refName},
				RefAction: test.action,
				RefType:   RefTypeBranch,
			}

			if err := test.def.Sanitize(); err != nil {
				t.Errorf("def invalid: %s", err.Error())
				return
			}

			violations, err := test.def.RefChangeVerify(context.Background(), in)
			if err != nil {
				t.Errorf("got an error: %s", err.Error())
				return
			}

			inspectBranchViolations(t, test.expCodes, test.expParams, violations)
		})
	}
}

func inspectBranchViolations(t *testing.T,
	expCodes []string,
	expParams [][]any,
	violations []types.RuleViolations,
) {
	if len(expCodes) == 0 &&
		(len(violations) == 0 || len(violations) == 1 && len(violations[0].Violations) == 0) {
		// no violations expected and no violations received
		return
	}

	if len(violations) != 1 {
		t.Error("expected size of violation should always be one")
		return
	}

	if want, got := len(expCodes), len(violations[0].Violations); want != got {
		t.Errorf("violation count: want=%d got=%d", want, got)
		return
	}

	for i, violation := range violations[0].Violations {
		if want, got := expCodes[i], violation.Code; want != got {
			t.Errorf("violation %d code mismatch: want=%s got=%s", i, want, got)
		}
		if want, got := expParams[i], violation.Params; !reflect.DeepEqual(want, got) {
			t.Errorf("violation %d params mismatch: want=%v got=%v", i, want, got)
		}
	}
}
