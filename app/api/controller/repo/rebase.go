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

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type RebaseInput struct {
	BaseBranch    string  `json:"base_branch"`
	HeadBranch    string  `json:"head_branch"`
	HeadCommitSHA sha.SHA `json:"head_commit_sha"`

	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (in *RebaseInput) validate() error {
	if in.BaseBranch == "" {
		return usererror.BadRequest("Base branch name must be provided")
	}

	if in.HeadBranch == "" {
		return usererror.BadRequest("Head branch name must be provided")
	}

	if in.HeadCommitSHA.IsEmpty() {
		return usererror.BadRequest("Head branch commit SHA must be provided")
	}

	return nil
}

// Rebase rebases a branch against (the latest commit from) a different branch.
func (c *Controller) Rebase(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *RebaseInput,
) (*types.RebaseResponse, *types.MergeViolations, error) {
	if err := in.validate(); err != nil {
		return nil, nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	isRepoOwner, err := apiauth.IsRepoOwner(ctx, c.authorizer, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to determine if user is repo owner: %w", err)
	}

	protectionRules, err := c.protectionManager.ForRepository(ctx, repo.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch protection rules for the repository: %w", err)
	}

	violations, err := protectionRules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		Repo:               repo,
		RefAction:          protection.RefActionUpdate,
		RefType:            protection.RefTypeBranch,
		RefNames:           []string{in.HeadBranch},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		return &types.RebaseResponse{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{RuleViolations: violations}, nil
	}

	readParams := git.CreateReadParams(repo)

	headBranch, err := c.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: readParams,
		BranchName: in.HeadBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get head branch: %w", err)
	}

	baseBranch, err := c.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: readParams,
		BranchName: in.BaseBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base branch: %w", err)
	}

	if !headBranch.Branch.SHA.Equal(in.HeadCommitSHA) {
		return nil, nil, usererror.BadRequestf("The commit %s isn't the latest commit on the branch %s",
			in.HeadCommitSHA, headBranch.Branch.SHA.String())
	}

	isAncestor, err := c.git.IsAncestor(ctx, git.IsAncestorParams{
		ReadParams:          readParams,
		AncestorCommitSHA:   baseBranch.Branch.SHA,
		DescendantCommitSHA: headBranch.Branch.SHA,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed check ancestor: %w", err)
	}

	if isAncestor.Ancestor {
		// The head branch already contains the latest commit from the base branch - nothing to do.
		return &types.RebaseResponse{
			DryRunRulesOutput: types.DryRunRulesOutput{
				RuleViolations: violations,
			},
		}, nil, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     writeParams,
		BaseBranch:      in.BaseBranch,
		HeadRepoUID:     repo.GitUID,
		HeadBranch:      in.HeadBranch,
		RefType:         gitenum.RefTypeBranch,
		RefName:         in.HeadBranch,
		HeadExpectedSHA: in.HeadCommitSHA,
		Method:          gitenum.MergeMethodRebase,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("rebase execution failed: %w", err)
	}

	if mergeOutput.MergeSHA.String() == "" || len(mergeOutput.ConflictFiles) > 0 {
		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
		}, nil
	}

	return &types.RebaseResponse{
		DryRunRulesOutput: types.DryRunRulesOutput{
			RuleViolations: violations,
		},
		NewHeadBranchSHA: mergeOutput.MergeSHA,
	}, nil, nil
}
