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

type ruleSet struct {
	rules   []types.RuleInfoInternal
	manager *Manager
}

var _ Protection = ruleSet{} // ensure that ruleSet implements the Protection interface.

func (s ruleSet) CanMerge(ctx context.Context, in CanMergeInput) (CanMergeOutput, []types.RuleViolations, error) {
	var out CanMergeOutput
	var violations []types.RuleViolations

	for _, r := range s.rules {
		matches, err := matchesName(r.Pattern, in.TargetRepo.DefaultBranch, in.PullReq.TargetBranch)
		if err != nil {
			return out, nil, err
		}
		if !matches {
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

func (s ruleSet) CanPush(ctx context.Context, in CanPushInput) (CanPushOutput, []types.RuleViolations, error) {
	var out CanPushOutput
	var violations []types.RuleViolations

	for _, r := range s.rules {
		matched, err := matchedNames(r.Pattern, in.Repo.DefaultBranch, in.BranchNames...)
		if err != nil {
			return out, nil, err
		}
		if len(matched) == 0 {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return out, nil,
				fmt.Errorf("failed to parse protection definition ID=%d Type=%s: %w", r.ID, r.Type, err)
		}

		ruleIn := in
		in.BranchNames = matched

		_, rVs, err := protection.CanPush(ctx, ruleIn)
		if err != nil {
			return out, nil, err
		}

		violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
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
