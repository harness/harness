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
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// DeleteBranch deletes a repo branch.
func (c *Controller) DeleteBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	branchName string,
	bypassRules,
	dryRunRules bool,
) (types.DeleteBranchOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, err
	}

	// make sure user isn't deleting the default branch
	// ASSUMPTION: lower layer calls explicit branch api
	// and 'refs/heads/branch1' would fail if 'branch1' exists.
	// TODO: Add functional test to ensure the scenario is covered!
	if branchName == repo.DefaultBranch {
		return types.DeleteBranchOutput{}, nil, usererror.ErrDefaultBranchCantBeDeleted
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, err
	}

	violations, err := rules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		Actor:       &session.Principal,
		AllowBypass: bypassRules,
		IsRepoOwner: isRepoOwner,
		Repo:        repo,
		RefAction:   protection.RefActionDelete,
		RefType:     protection.RefTypeBranch,
		RefNames:    []string{branchName},
	})
	if err != nil {
		return types.DeleteBranchOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if dryRunRules {
		return types.DeleteBranchOutput{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return types.DeleteBranchOutput{}, violations, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	err = c.git.DeleteBranch(ctx, &git.DeleteBranchParams{
		WriteParams: writeParams,
		BranchName:  branchName,
	})
	if err != nil {
		return types.DeleteBranchOutput{}, nil, err
	}

	if protection.IsBypassed(violations) {
		err = c.auditService.Log(ctx,
			session.Principal,
			audit.NewResource(
				audit.ResourceTypeRepository,
				repo.Identifier,
				audit.RepoPath,
				repo.Path,
				audit.BypassedResourceType,
				audit.BypassedResourceTypeBranch,
				audit.BypassedResourceName,
				branchName,
				audit.BypassAction,
				audit.BypassActionDeleted,
				audit.ResourceName,
				fmt.Sprintf(
					audit.BypassSHALabelFormat,
					repo.Identifier,
					branchName,
				),
			),
			audit.ActionBypassed,
			paths.Parent(repo.Path),
			audit.WithNewObject(audit.BranchObject{
				BranchName:     branchName,
				RepoPath:       repo.Path,
				RuleViolations: violations,
			}),
		)
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete branch operation: %s", err)
		}
	}

	return types.DeleteBranchOutput{
		DryRunRulesOutput: types.DryRunRulesOutput{
			RuleViolations: violations,
		}}, nil, nil
}
