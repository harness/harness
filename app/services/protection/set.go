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
	"github.com/harness/gitness/types/enum"

	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
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

	if in.Method == "" {
		out.AllowedMethods = slices.Clone(enum.MergeMethods)
	}

	err := s.forEachRuleMatchBranch(in.TargetRepo.DefaultBranch, in.PullReq.TargetBranch,
		func(r *types.RuleInfoInternal, p Protection) error {
			rOut, rVs, err := p.MergeVerify(ctx, in)
			if err != nil {
				return err
			}

			// combine output across rules
			violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
			out.AllowedMethods = intersectSorted(out.AllowedMethods, rOut.AllowedMethods)
			out.DeleteSourceBranch = out.DeleteSourceBranch || rOut.DeleteSourceBranch
			out.MinimumRequiredApprovalsCount = maxInt(out.MinimumRequiredApprovalsCount, rOut.MinimumRequiredApprovalsCount)
			out.MinimumRequiredApprovalsCountLatest = maxInt(out.MinimumRequiredApprovalsCountLatest, rOut.MinimumRequiredApprovalsCountLatest) //nolint:lll
			out.RequiresCodeOwnersApproval = out.RequiresCodeOwnersApproval || rOut.RequiresCodeOwnersApproval
			out.RequiresCodeOwnersApprovalLatest = out.RequiresCodeOwnersApprovalLatest || rOut.RequiresCodeOwnersApprovalLatest
			out.RequiresCommentResolution = out.RequiresCommentResolution || rOut.RequiresCommentResolution
			out.RequiresNoChangeRequests = out.RequiresNoChangeRequests || rOut.RequiresNoChangeRequests

			return nil
		})
	if err != nil {
		return out, nil, fmt.Errorf("failed to merge verify: %w", err)
	}

	return out, violations, nil
}

func (s ruleSet) RequiredChecks(
	ctx context.Context,
	in RequiredChecksInput,
) (RequiredChecksOutput, error) {
	requiredIDMap := map[string]struct{}{}
	bypassableIDMap := map[string]struct{}{}
	err := s.forEachRuleMatchBranch(in.Repo.DefaultBranch, in.PullReq.TargetBranch,
		func(_ *types.RuleInfoInternal, p Protection) error {
			out, err := p.RequiredChecks(ctx, in)
			if err != nil {
				return err
			}

			for reqCheckID := range out.RequiredIdentifiers {
				requiredIDMap[reqCheckID] = struct{}{}
				delete(bypassableIDMap, reqCheckID)
			}
			for reqCheckID := range out.BypassableIdentifiers {
				if _, ok := requiredIDMap[reqCheckID]; ok {
					continue
				}
				bypassableIDMap[reqCheckID] = struct{}{}
			}

			return nil
		})
	if err != nil {
		return RequiredChecksOutput{}, err
	}

	return RequiredChecksOutput{
		RequiredIdentifiers:   requiredIDMap,
		BypassableIdentifiers: bypassableIDMap,
	}, nil
}

func (s ruleSet) RefChangeVerify(ctx context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error) {
	var violations []types.RuleViolations

	err := s.forEachRuleMatchRefs(in.Repo.DefaultBranch, in.RefNames,
		func(r *types.RuleInfoInternal, p Protection, matched []string) error {
			ruleIn := in
			ruleIn.RefNames = matched

			rVs, err := p.RefChangeVerify(ctx, ruleIn)
			if err != nil {
				return err
			}

			violations = append(violations, backFillRule(rVs, r.RuleInfo)...)

			return nil
		})
	if err != nil {
		return nil, err
	}

	return violations, nil
}

func (s ruleSet) UserIDs() ([]int64, error) {
	mapIDs := make(map[int64]struct{})
	err := s.forEachRule(func(_ *types.RuleInfoInternal, p Protection) error {
		userIDs, err := p.UserIDs()
		if err != nil {
			return err
		}

		for _, userID := range userIDs {
			mapIDs[userID] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]int64, 0, len(mapIDs))
	for userID := range mapIDs {
		result = append(result, userID)
	}

	return result, nil
}

func (s ruleSet) UserGroupIDs() ([]int64, error) {
	mapIDs := make(map[int64]struct{})
	err := s.forEachRule(func(_ *types.RuleInfoInternal, p Protection) error {
		userGroupIDs, err := p.UserGroupIDs()
		if err != nil {
			return err
		}

		for _, userGroupID := range userGroupIDs {
			mapIDs[userGroupID] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	result := make([]int64, 0, len(mapIDs))
	for userGroupID := range mapIDs {
		result = append(result, userGroupID)
	}

	return result, nil
}

func (s ruleSet) forEachRule(
	fn func(r *types.RuleInfoInternal, p Protection) error,
) error {
	for i := range s.rules {
		r := s.rules[i]

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRule: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		err = fn(&r, protection)
		if err != nil {
			return fmt.Errorf("forEachRule: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}
	}

	return nil
}

func (s ruleSet) forEachRuleMatchBranch(
	defaultBranch string,
	branchName string,
	fn func(r *types.RuleInfoInternal, p Protection) error,
) error {
	for i := range s.rules {
		r := s.rules[i]

		matches, err := matchesName(r.Pattern, defaultBranch, branchName)
		if err != nil {
			return err
		}
		if !matches {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchBranch: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		err = fn(&r, protection)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchBranch: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}
	}

	return nil
}

func (s ruleSet) forEachRuleMatchRefs(
	defaultBranch string,
	refNames []string,
	fn func(r *types.RuleInfoInternal, p Protection, matched []string) error,
) error {
	for i := range s.rules {
		r := s.rules[i]

		matched, err := matchedNames(r.Pattern, defaultBranch, refNames...)
		if err != nil {
			return err
		}
		if len(matched) == 0 {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchRefs: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		err = fn(&r, protection, matched)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchRefs: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}
	}

	return nil
}

func backFillRule(vs []types.RuleViolations, rule types.RuleInfo) []types.RuleViolations {
	for i := range vs {
		vs[i].Rule = rule
	}
	return vs
}

// intersectSorted removed all elements of the "sliceA" that are not also in the "sliceB" slice.
// Assumes both slices are sorted.
func intersectSorted[T constraints.Ordered](sliceA, sliceB []T) []T {
	var idxA, idxB int
	for idxA < len(sliceA) && idxB < len(sliceB) {
		a, b := sliceA[idxA], sliceB[idxB]

		if a == b {
			idxA++
			continue
		}

		if a < b {
			sliceA = append(sliceA[:idxA], sliceA[idxA+1:]...)
			continue
		}

		idxB++
	}
	sliceA = sliceA[:idxA]

	return sliceA
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
