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

package merge

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/services/checkreq"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CheckRulesInput struct {
	PullReq          *types.PullReq
	TargetRepo       *types.RepositoryCore
	SourceRepo       *types.RepositoryCore
	Actor            *types.Principal
	IsRepoOwner      bool
	IsAncestor       bool // is the current target branch commit an ancestor of the latest commit on the source branch
	MergeMethod      enum.MergeMethod
	AllowBypassRules bool
}

func (s *Service) CheckRules(
	ctx context.Context,
	protectionRules protection.BranchProtection,
	in CheckRulesInput,
) (protection.MergeVerifyOutput, []types.RuleViolations, error) {
	return s.checkRules(ctx, protectionRules, false, in)
}

func (s *Service) CheckRulesForMergeQueue(
	ctx context.Context,
	protectionRules protection.BranchProtection,
	in CheckRulesInput,
) (protection.MergeVerifyOutput, []types.RuleViolations, error) {
	return s.checkRules(ctx, protectionRules, true, in)
}

// RequiredChecks returns the checks required for the pull request by an owner
// outside of gitness, independently of the branch protection rules.
func (s *Service) RequiredChecks(
	ctx context.Context,
	repo *types.RepositoryCore,
	pr *types.PullReq,
) (checkreq.Resolved, error) {
	return checkreq.Resolve(ctx, s.checkRequirements, repo, pr)
}

func (s *Service) checkRules(
	ctx context.Context,
	protectionRules protection.BranchProtection,
	isMergeQueue bool,
	in CheckRulesInput,
) (protection.MergeVerifyOutput, []types.RuleViolations, error) {
	reviewers, err := s.reviewerStore.List(ctx, in.PullReq.ID)
	if err != nil {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("failed to load list of reviwers: %w", err)
	}

	checkResults, err := s.checkStore.ListResults(ctx, in.PullReq.TargetRepoID, in.PullReq.SourceSHA)
	if err != nil {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("failed to list status checks: %w", err)
	}

	codeOwnerWithApproval, err := s.codeOwners.Evaluate(ctx, in.TargetRepo, in.PullReq, reviewers)
	if err != nil && !errors.Is(err, codeowners.ErrNotFound) {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("CODEOWNERS evaluation failed: %w", err)
	}

	ruleOut, violations, err := protectionRules.MergeVerify(ctx, protection.MergeVerifyInput{
		ResolveUserGroupIDs: s.userGroupService.ListUserIDsByGroupIDs,
		MapUserGroupIDs:     s.userGroupService.MapGroupIDsToPrincipals,
		Actor:               in.Actor,
		AllowBypass:         in.AllowBypassRules,
		IsRepoOwner:         in.IsRepoOwner,
		TargetRepo:          in.TargetRepo,
		SourceRepo:          in.SourceRepo,
		PullReq:             in.PullReq,
		Reviewers:           reviewers,
		Method:              in.MergeMethod,
		TargetIsAncestor:    in.IsAncestor,
		CheckResults:        checkResults,
		CodeOwners:          codeOwnerWithApproval,
		OmitMQViolations:    isMergeQueue,
	})
	if err != nil {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	// Checks required by an external owner are verified separately from the
	// protection rules, so that a rule's bypass list cannot waive them.
	requirements, err := checkreq.Resolve(ctx, s.checkRequirements, in.TargetRepo, in.PullReq)
	if err != nil {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("failed to resolve external checks: %w", err)
	}
	violations = append(violations, checkreq.Violations(requirements.Unsatisfied(checkResults))...)

	return ruleOut, violations, nil
}
