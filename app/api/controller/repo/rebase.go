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
	BaseCommitSHA sha.SHA `json:"base_commit_sha"`

	HeadBranch    string  `json:"head_branch"`
	HeadCommitSHA sha.SHA `json:"head_commit_sha"`

	DryRun      bool `json:"dry_run"`
	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (in *RebaseInput) validate() error {
	if in.BaseBranch == "" && in.BaseCommitSHA.IsEmpty() {
		return usererror.BadRequest("Either base branch or base commit SHA name must be provided")
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

	protectionRules, isRepoOwner, err := c.fetchBranchRules(ctx, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	violations, err := protectionRules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		Repo:               repo,
		RefAction:          protection.RefActionUpdateForce,
		RefType:            protection.RefTypeBranch,
		RefNames:           []string{in.HeadBranch},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		// DryRunRules is true: Just return rule violations and don't attempt to rebase.
		return &types.RebaseResponse{
			RuleViolations: violations,
			DryRunRules:    true,
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return nil, &types.MergeViolations{
			RuleViolations: violations,
			Message:        protection.GenerateErrorMessageForBlockingViolations(violations),
		}, nil
	}

	readParams := git.CreateReadParams(repo)

	headBranch, err := c.git.GetBranch(ctx, &git.GetBranchParams{
		ReadParams: readParams,
		BranchName: in.HeadBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get head branch: %w", err)
	}

	if !headBranch.Branch.SHA.Equal(in.HeadCommitSHA) {
		return nil, nil, usererror.BadRequestf("The commit %s isn't the latest commit on the branch %s",
			in.HeadCommitSHA, headBranch.Branch.Name)
	}

	baseCommitSHA := in.BaseCommitSHA
	if baseCommitSHA.IsEmpty() {
		baseBranch, err := c.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: readParams,
			BranchName: in.BaseBranch,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get base branch: %w", err)
		}

		baseCommitSHA = baseBranch.Branch.SHA
	}

	isAncestor, err := c.git.IsAncestor(ctx, git.IsAncestorParams{
		ReadParams:          readParams,
		AncestorCommitSHA:   baseCommitSHA,
		DescendantCommitSHA: headBranch.Branch.SHA,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed check ancestor: %w", err)
	}

	if isAncestor.Ancestor {
		// The head branch already contains the latest commit from the base branch - nothing to do.
		return &types.RebaseResponse{
			AlreadyAncestor: true,
			RuleViolations:  violations,
		}, nil, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	var refs []git.RefUpdate
	if !in.DryRun {
		headBranchRef, err := git.GetRefPath(in.HeadBranch, gitenum.RefTypeBranch)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to gerenere ref name: %w", err)
		}

		refs = append(refs, git.RefUpdate{
			Name: headBranchRef,
			Old:  headBranch.Branch.SHA,
			New:  sha.SHA{}, // update to the result of the merge
		})
	}

	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     writeParams,
		BaseSHA:         baseCommitSHA,
		HeadRepoUID:     repo.GitUID,
		HeadBranch:      in.HeadBranch,
		Refs:            refs,
		HeadExpectedSHA: in.HeadCommitSHA,
		Method:          gitenum.MergeMethodRebase,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("rebase execution failed: %w", err)
	}

	if in.DryRun {
		// DryRun is true: Just return rule violations and list of conflicted files.
		// No reference is updated, so don't return the resulting commit SHA.
		return &types.RebaseResponse{
			RuleViolations: violations,
			DryRun:         true,
			ConflictFiles:  mergeOutput.ConflictFiles,
		}, nil, nil
	}

	if mergeOutput.MergeSHA.IsEmpty() || len(mergeOutput.ConflictFiles) > 0 {
		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
			Message:        fmt.Sprintf("Rebase blocked by conflicting files: %v", mergeOutput.ConflictFiles),
		}, nil
	}

	return &types.RebaseResponse{
		NewHeadBranchSHA: mergeOutput.MergeSHA,
		RuleViolations:   violations,
	}, nil, nil
}
