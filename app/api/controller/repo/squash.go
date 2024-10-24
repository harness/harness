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
	"strings"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type SquashInput struct {
	BaseBranch    string  `json:"base_branch"`
	BaseCommitSHA sha.SHA `json:"base_commit_sha"`

	HeadBranch    string  `json:"head_branch"`
	HeadCommitSHA sha.SHA `json:"head_commit_sha"`

	Title   string `json:"title"`
	Message string `json:"message"`

	DryRun      bool `json:"dry_run"`
	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

func (in *SquashInput) validate() error {
	if in.BaseBranch == "" && in.BaseCommitSHA.IsEmpty() {
		return usererror.BadRequest("Either base branch or base commit SHA name must be provided")
	}

	if in.HeadBranch == "" {
		return usererror.BadRequest("Head branch name must be provided")
	}

	if in.HeadCommitSHA.IsEmpty() {
		return usererror.BadRequest("Head branch commit SHA must be provided")
	}

	// cleanup title / message (NOTE: git doesn't support white space only)
	in.Title = strings.TrimSpace(in.Title)
	in.Message = strings.TrimSpace(in.Message)

	return nil
}

// Squash squashes all commits since merge-base to the latest commit of the base branch.
// This operation alters history of the base branch and therefore is considered a force push.
//
//nolint:gocognit
func (c *Controller) Squash(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *SquashInput,
) (*types.SquashResponse, *types.MergeViolations, error) {
	if err := in.validate(); err != nil {
		return nil, nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	protectionRules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
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
		// DryRunRules is true: Just return rule violations and don't squash commits.
		return &types.SquashResponse{
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

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	refType := gitenum.RefTypeBranch
	refName := in.HeadBranch
	if in.DryRun {
		refType = gitenum.RefTypeUndefined
		refName = ""
	}

	mergeBase, err := c.git.MergeBase(ctx, git.MergeBaseParams{
		ReadParams: readParams,
		Ref1:       baseCommitSHA.String(),
		Ref2:       headBranch.Branch.SHA.String(),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	const commitBulletLimit = 50

	commits, err := c.git.ListCommits(ctx, &git.ListCommitsParams{
		ReadParams: readParams,
		GitREF:     in.HeadCommitSHA.String(),
		After:      mergeBase.MergeBaseSHA.String(),
		Limit:      commitBulletLimit,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read list of commits: %w", err)
	}

	commitCount := len(commits.Commits)

	if commitCount == 0 {
		return nil, nil, usererror.BadRequest("Failed to find commits between head and base")
	}

	commit0Subject, commit0Body := parser.SplitMessage(commits.Commits[0].Message)

	if in.Title == "" {
		in.Title = commit0Subject
		if commitCount > 1 {
			in.Title = fmt.Sprintf("Squashed %d commits", commits.TotalCommits)
		}
	}

	if in.Message == "" {
		in.Message = commit0Body
		if commitCount > 1 {
			sb := strings.Builder{}
			for i := range min(commitBulletLimit, len(commits.Commits)) {
				sb.WriteString("* ")
				sb.WriteString(commits.Commits[i].Title)
				sb.WriteByte('\n')
			}
			if otherCommitCount := commits.TotalCommits - len(commits.Commits); otherCommitCount > 0 {
				sb.WriteString(fmt.Sprintf("* and %d more commits\n", otherCommitCount))
			}

			in.Message = sb.String()
		}
	}

	author := controller.IdentityFromPrincipalInfo(*session.Principal.ToPrincipalInfo())
	committer := controller.SystemServicePrincipalInfo()
	now := time.Now()

	mergeOutput, err := c.git.Merge(ctx, &git.MergeParams{
		WriteParams:     writeParams,
		BaseSHA:         mergeBase.MergeBaseSHA,
		HeadRepoUID:     repo.GitUID,
		HeadBranch:      in.HeadBranch,
		Message:         git.CommitMessage(in.Title, in.Message),
		RefType:         refType,
		RefName:         refName,
		Committer:       committer,
		CommitterDate:   &now,
		Author:          author,
		AuthorDate:      &now,
		HeadExpectedSHA: in.HeadCommitSHA,
		Method:          gitenum.MergeMethodSquash,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("squash execution failed: %w", err)
	}

	if in.DryRun {
		// DryRun is true: Just return rule violations and list of conflicted files.
		// No reference is updated, so don't return the resulting commit SHA.
		return &types.SquashResponse{
			RuleViolations: violations,
			DryRun:         true,
			ConflictFiles:  mergeOutput.ConflictFiles,
		}, nil, nil
	}

	if mergeOutput.MergeSHA.IsEmpty() || len(mergeOutput.ConflictFiles) > 0 {
		return nil, &types.MergeViolations{
			ConflictFiles:  mergeOutput.ConflictFiles,
			RuleViolations: violations,
			Message:        fmt.Sprintf("Squash blocked by conflicting files: %v", mergeOutput.ConflictFiles),
		}, nil
	}

	return &types.SquashResponse{
		NewHeadBranchSHA: mergeOutput.MergeSHA,
		RuleViolations:   violations,
	}, nil, nil
}
