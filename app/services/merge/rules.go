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
	MergeMethod      enum.MergeMethod
	AllowBypassRules bool
}

func (s *Service) CheckRules(
	ctx context.Context,
	protectionRules protection.BranchProtection,
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
		CheckResults:        checkResults,
		CodeOwners:          codeOwnerWithApproval,
	})
	if err != nil {
		return protection.MergeVerifyOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	return ruleOut, violations, nil
}
