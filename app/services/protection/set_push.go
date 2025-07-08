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
	"fmt"

	"github.com/harness/gitness/types"
)

type pushRuleSet struct {
	rules   []types.RuleInfoInternal
	manager *Manager
}

var _ PushProtection = pushRuleSet{}

func (s pushRuleSet) PushVerify(
	ctx context.Context,
	in PushVerifyInput,
) (PushVerifyOutput, []types.RuleViolations, error) {
	var violations []types.RuleViolations
	var out PushVerifyOutput
	out.Protections = make(map[int64]PushProtection, len(s.rules))

	for _, r := range s.rules {
		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return out, nil, fmt.Errorf(
				"failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err,
			)
		}

		pushProtection, ok := protection.(PushProtection)
		if !ok {
			return out, nil, fmt.Errorf(
				"unexpected type for protection: got %T, expected PushProtection",
				protection,
			)
		}

		out.Protections[r.ID] = pushProtection

		rOut, rViolations, err := pushProtection.PushVerify(ctx, in)
		if err != nil {
			return out, nil, fmt.Errorf("failed to process push rule in push rule set: %w", err)
		}

		if out.FileSizeLimit == 0 ||
			(rOut.FileSizeLimit > 0 && out.FileSizeLimit > rOut.FileSizeLimit) {
			out.FileSizeLimit = rOut.FileSizeLimit
		}

		out.PrincipalCommitterMatch = out.PrincipalCommitterMatch || rOut.PrincipalCommitterMatch

		out.SecretScanningEnabled = out.SecretScanningEnabled || rOut.SecretScanningEnabled

		violations = append(violations, rViolations...)
	}

	return out, violations, nil
}

func (s pushRuleSet) Violations(in *PushViolationsInput) (PushViolationsOutput, error) {
	output := PushViolationsOutput{}

	for _, r := range s.rules {
		out, err := in.Protections[r.ID].Violations(in)
		if err != nil {
			return PushViolationsOutput{}, fmt.Errorf(
				"failed to backfill violations: %w", err,
			)
		}

		output.Violations = append(output.Violations, backFillRule(out.Violations, r.RuleInfo)...)
	}

	return output, nil
}

func (s pushRuleSet) UserIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserIDs)
}

func (s pushRuleSet) UserGroupIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserGroupIDs)
}
