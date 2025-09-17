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
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ForkSyncInput struct {
	Branch          string  `json:"branch"`
	BranchCommitSHA sha.SHA `json:"branch_commit_sha"`

	BranchUpstream string `json:"branch_upstream"` // Can be omitted, defaults to the value of Branch

	DryRun      bool `json:"dry_run"`
	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (in *ForkSyncInput) validate() error {
	if in.Branch == "" {
		return errors.InvalidArgument("Branch name must be provided")
	}

	if in.BranchCommitSHA.IsEmpty() {
		return errors.InvalidArgument("Branch commit SHA must be provided")
	}

	return nil
}

//nolint:gocognit
func (c *Controller) ForkSync(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *ForkSyncInput,
) (*types.ForkSyncOutput, *types.MergeViolations, error) {
	if err := in.validate(); err != nil {
		return nil, nil, err
	}

	repoForkCore, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, err
	}

	branchUpstreamName := in.BranchUpstream
	if branchUpstreamName == "" {
		branchUpstreamName = in.Branch
	}

	branchForkInfo, err := c.git.GetRef(ctx, git.GetRefParams{
		ReadParams: git.CreateReadParams(repoForkCore),
		Name:       in.Branch,
		Type:       gitenum.RefTypeBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get repo branch: %w", err)
	}

	if !branchForkInfo.SHA.Equal(in.BranchCommitSHA) {
		return nil, nil, errors.InvalidArgument("The commit %s isn't the latest commit on the branch %s",
			in.BranchCommitSHA, in.Branch)
	}

	branchUpstreamSHA, repoUpstreamCore, err := c.fetchUpstreamBranch(
		ctx,
		session,
		repoForkCore,
		branchUpstreamName,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch upstream branch: %w", err)
	}

	ancestorResult, err := c.git.IsAncestor(ctx, git.IsAncestorParams{
		ReadParams:          git.CreateReadParams(repoForkCore),
		AncestorCommitSHA:   branchUpstreamSHA,
		DescendantCommitSHA: branchForkInfo.SHA,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check if the upstream commit is ancestor: %w", err)
	}

	if ancestorResult.Ancestor {
		// The branch already contains the latest commit from the upstream repository branch - nothing to do.
		return &types.ForkSyncOutput{
			AlreadyAncestor: true,
		}, nil, nil
	}

	mergeBase, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: git.CreateReadParams(repoForkCore),
		Ref1:       branchUpstreamSHA.String(),
		Ref2:       branchForkInfo.SHA.String(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	var (
		message   string
		author    *git.Identity
		committer *git.Identity
	)

	mergeMethod := gitenum.MergeMethodFastForward
	if !branchForkInfo.SHA.Equal(mergeBase.MergeBaseSHA) {
		mergeMethod = gitenum.MergeMethodMerge

		message = fmt.Sprintf("Merge upstream branch '%s' of %s",
			branchUpstreamName, repoUpstreamCore.Path)
		committer = controller.SystemServicePrincipalInfo()
		author = controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	}

	protectionRules, isRepoOwner, err := c.fetchBranchRules(ctx, session, repoForkCore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch rules: %w", err)
	}

	violations, err := protectionRules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		Repo:               repoForkCore,
		RefAction:          protection.RefActionUpdate,
		RefType:            protection.RefTypeBranch,
		RefNames:           []string{in.Branch},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		// DryRunRules is true: Just return rule violations and don't attempt to rebase.
		return &types.ForkSyncOutput{
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

	var refs []git.RefUpdate
	if !in.DryRun {
		headBranchRef, err := git.GetRefPath(in.Branch, gitenum.RefTypeBranch)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate ref name: %w", err)
		}

		refs = append(refs, git.RefUpdate{
			Name: headBranchRef,
			Old:  branchForkInfo.SHA,
			New:  sha.SHA{}, // update to the result of the merge
		})
	}

	now := time.Now()

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, c.urlProvider, session, repoForkCore)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams: writeParams,
		//HeadRepoUID:     repoUpstreamCore.GitUID, // TODO: Remove HeadRepoUID!
		BaseSHA:       branchForkInfo.SHA,
		HeadSHA:       branchUpstreamSHA,
		Message:       message,
		Committer:     committer,
		CommitterDate: &now,
		Author:        author,
		AuthorDate:    &now,
		Refs:          refs,
		Method:        mergeMethod,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("fork branch sync merge failed: %w", err)
	}

	if in.DryRun {
		// DryRun is true: Just return rule violations and list of conflicted files.
		// No reference is updated, so don't return the resulting commit SHA.
		return &types.ForkSyncOutput{
			RuleViolations: violations,
			DryRun:         true,
			ConflictFiles:  mergeOutput.ConflictFiles,
		}, nil, nil
	}

	if mergeOutput.MergeSHA.IsEmpty() || len(mergeOutput.ConflictFiles) > 0 {
		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
			Message:        fmt.Sprintf("Fork sync blocked by conflicting files: %v", mergeOutput.ConflictFiles),
		}, nil
	}

	return &types.ForkSyncOutput{
		NewCommitSHA:   mergeOutput.MergeSHA,
		RuleViolations: violations,
	}, nil, nil
}
