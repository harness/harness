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

func (s ruleSet) MergeVerify(
	ctx context.Context,
	in MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	var out MergeVerifyOutput
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

		rOut, rVs, err := protection.MergeVerify(ctx, in)
		if err != nil {
			return out, nil, err
		}

		violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
		out.DeleteSourceBranch = out.DeleteSourceBranch || rOut.DeleteSourceBranch
	}

	return out, violations, nil
}

func (s ruleSet) RefChangeVerify(ctx context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error) {
	var violations []types.RuleViolations

	for _, r := range s.rules {
		matched, err := matchedNames(r.Pattern, in.Repo.DefaultBranch, in.RefNames...)
		if err != nil {
			return nil, err
		}
		if len(matched) == 0 {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return nil,
				fmt.Errorf("failed to parse protection definition ID=%d Type=%s: %w", r.ID, r.Type, err)
		}

		ruleIn := in
		ruleIn.RefNames = matched

		rVs, err := protection.RefChangeVerify(ctx, ruleIn)
		if err != nil {
			return nil, err
		}

		violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
	}

	return violations, nil
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
