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

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// DeleteBranch deletes the source branch of a PR.
func (c *Controller) DeleteBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	bypassRules,
	dryRunRules bool,
) (types.DeleteBranchOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}
	branchName := pr.SourceBranch

	// make sure user isn't deleting the default branch
	if branchName == repo.DefaultBranch {
		return types.DeleteBranchOutput{}, nil, usererror.ErrDefaultBranchCantBeDeleted
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return types.DeleteBranchOutput{}, nil, fmt.Errorf("failed to fetch rules: %w", err)
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

	branch, err := func() (types.Branch, error) {
		rpcOut, err := c.git.GetBranch(ctx, &git.GetBranchParams{
			ReadParams: git.CreateReadParams(repo),
			BranchName: branchName,
		})
		if err != nil {
			return types.Branch{}, fmt.Errorf("failed to fetch source branch: %w", err)
		}

		mappedBranch, err := controller.MapBranch(rpcOut.Branch)
		if err != nil {
			return types.Branch{}, fmt.Errorf("failed to map source branch: %w", err)
		}
		return mappedBranch, nil
	}()
	if err != nil {
		return types.DeleteBranchOutput{}, nil, err
	}
	if pr.SourceSHA != branch.SHA.String() {
		return types.DeleteBranchOutput{}, nil, errors.Conflict("source branch SHA does not match pull request source SHA")
	}

	err = c.git.DeleteBranch(ctx, &git.DeleteBranchParams{
		WriteParams: writeParams,
		BranchName:  branchName,
		SHA:         branch.SHA.String(),
	})
	if err != nil {
		return types.DeleteBranchOutput{}, nil, err
	}

	err = func() error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to update pull request activity sequence: %w", err)
		}

		_, err := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID,
			&types.PullRequestActivityPayloadBranchDelete{SHA: branch.SHA.String()}, nil)
		return err
	}()
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity for successful branch delete")
	}

	return types.DeleteBranchOutput{
		DryRunRulesOutput: types.DryRunRulesOutput{
			RuleViolations: violations,
		}}, nil, nil
}
