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
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/types"
)

type ruleSet struct {
	rules   []types.RuleInfoInternal
	manager *Manager
}

var _ Protection = ruleSet{} // ensure that ruleSet implements the Protection interface.

func (s ruleSet) CanMerge(ctx context.Context, in CanMergeInput) (CanMergeOutput, []types.RuleViolations, error) {
	var out CanMergeOutput
	var violations []types.RuleViolations

	for _, r := range s.rules {
		bp := Pattern{}

		if err := json.Unmarshal(r.Pattern, &bp); err != nil {
			return out, nil, fmt.Errorf("failed to parse branch pattern: %w", err)
		}

		if !bp.Matches(in.PullReq.TargetBranch, in.TargetRepo.DefaultBranch) {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return out, nil,
				fmt.Errorf("failed to parse protection definition ID=%d Type=%s: %w", r.ID, r.Type, err)
		}

		rOut, rVs, err := protection.CanMerge(ctx, in)
		if err != nil {
			return out, nil, err
		}

		violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
		out.DeleteSourceBranch = out.DeleteSourceBranch || rOut.DeleteSourceBranch
	}

	return out, violations, nil
}

func backFillRule(vs []types.RuleViolations, rule types.RuleInfo) []types.RuleViolations {
	violations := make([]types.RuleViolations, 0, len(vs))

	for i := range vs {
		if len(vs[i].Violations) == 0 {
			continue
		}

		vs[i].Rule = rule
		violations = append(violations, vs[i])
	}

	return violations
}
