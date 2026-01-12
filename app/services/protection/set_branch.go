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
	"errors"
	"fmt"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
	"golang.org/x/exp/constraints"
	"golang.org/x/exp/slices"
)

type branchRuleSet struct {
	rules   []types.RuleInfoInternal
	manager *Manager
}

var _ Protection = branchRuleSet{} // ensure that ruleSet implements the Protection interface.

func (s branchRuleSet) MergeVerify(
	ctx context.Context,
	in MergeVerifyInput,
) (MergeVerifyOutput, []types.RuleViolations, error) {
	var out MergeVerifyOutput
	var violations []types.RuleViolations

	out.AllowedMethods = slices.Clone(enum.MergeMethods)

	err := s.forEachRuleMatchBranch(
		in.TargetRepo.ID,
		in.TargetRepo.Identifier,
		in.TargetRepo.DefaultBranch,
		in.PullReq.TargetBranch,
		func(r *types.RuleInfoInternal, p BranchProtection) error {
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
			out.RequiresBypassMessage = out.RequiresBypassMessage || rOut.RequiresBypassMessage
			out.DefaultReviewerApprovals = append(out.DefaultReviewerApprovals, rOut.DefaultReviewerApprovals...)

			return nil
		})
	if err != nil {
		return out, nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return out, violations, nil
}

func (s branchRuleSet) RequiredChecks(
	ctx context.Context,
	in RequiredChecksInput,
) (RequiredChecksOutput, error) {
	requiredIDMap := map[string]struct{}{}
	bypassableIDMap := map[string]struct{}{}

	err := s.forEachRuleMatchBranch(
		in.Repo.ID,
		in.Repo.Identifier,
		in.Repo.DefaultBranch,
		in.PullReq.TargetBranch,
		func(_ *types.RuleInfoInternal, p BranchProtection) error {
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
		return RequiredChecksOutput{}, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return RequiredChecksOutput{
		RequiredIdentifiers:   requiredIDMap,
		BypassableIdentifiers: bypassableIDMap,
	}, nil
}

func (s branchRuleSet) CreatePullReqVerify(
	ctx context.Context,
	in CreatePullReqVerifyInput,
) (CreatePullReqVerifyOutput, []types.RuleViolations, error) {
	var out CreatePullReqVerifyOutput
	var violations []types.RuleViolations

	err := s.forEachRuleMatchBranch(
		in.RepoID,
		in.RepoIdentifier,
		in.DefaultBranch,
		in.TargetBranch,
		func(r *types.RuleInfoInternal, p BranchProtection) error {
			rOut, rVs, err := p.CreatePullReqVerify(ctx, in)
			if err != nil {
				return err
			}

			// combine output across rules
			violations = append(violations, backFillRule(rVs, r.RuleInfo)...)
			out.RequestCodeOwners = out.RequestCodeOwners || rOut.RequestCodeOwners
			out.DefaultReviewerIDs = append(out.DefaultReviewerIDs, rOut.DefaultReviewerIDs...)
			out.DefaultGroupReviewerIDs = append(out.DefaultGroupReviewerIDs, rOut.DefaultGroupReviewerIDs...)

			return nil
		})
	if err != nil {
		return out, nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	out.DefaultReviewerIDs = deduplicateInt64Slice(out.DefaultReviewerIDs)
	out.DefaultGroupReviewerIDs = deduplicateInt64Slice(out.DefaultGroupReviewerIDs)

	return out, violations, nil
}

func (s branchRuleSet) RefChangeVerify(ctx context.Context, in RefChangeVerifyInput) ([]types.RuleViolations, error) {
	var violations []types.RuleViolations

	err := forEachRuleMatchRefs(
		s.manager,
		s.rules,
		in.Repo.ID,
		in.Repo.Identifier,
		in.Repo.DefaultBranch,
		in.RefNames,
		refChangeVerifyFunc(ctx, in, &violations),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to process each rule in ruleSet: %w", err)
	}

	return violations, nil
}

func (s branchRuleSet) UserIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserIDs)
}

func (s branchRuleSet) UserGroupIDs() ([]int64, error) {
	return collectIDs(s.manager, s.rules, Protection.UserGroupIDs)
}

func (s branchRuleSet) forEachRuleMatchBranch(
	repoID int64,
	repoIdentifier string,
	defaultBranch string,
	branchName string,
	fn func(r *types.RuleInfoInternal, p BranchProtection) error,
) error {
	for i := range s.rules {
		r := s.rules[i]

		matchedRepo, err := matchesRepo(r.RepoTarget, repoID, repoIdentifier)
		if err != nil {
			return err
		}
		if !matchedRepo {
			continue
		}

		matchedRef, err := matchesRef(r.Pattern, defaultBranch, branchName)
		if err != nil {
			return err
		}
		if !matchedRef {
			continue
		}

		protection, err := s.manager.FromJSON(r.Type, r.Definition, false)
		if err != nil {
			return fmt.Errorf("forEachRuleMatchBranch: failed to parse protection definition ID=%d Type=%s: %w",
				r.ID, r.Type, err)
		}

		branchProtection, ok := protection.(BranchProtection)
		if !ok { // theoretically, should never happen
			log.Warn().Err(errors.New("failed to type assert Protection to BranchProtection"))
			return nil
		}

		err = fn(&r, branchProtection)
		if err != nil {
			return fmt.Errorf(
				"forEachRuleMatchBranch: failed to process rule ID=%d Type=%s: %w",
				r.ID, r.Type, err,
			)
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

func deduplicateInt64Slice(slice []int64) []int64 {
	seen := make(map[int64]bool)
	result := []int64{}

	for _, val := range slice {
		if _, ok := seen[val]; ok {
			continue
		}

		seen[val] = true
		result = append(result, val)
	}

	return result
}
