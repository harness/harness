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
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// RestoreBranchInput used for branch restoration apis.
type RestoreBranchInput struct {
	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

// RestoreBranch restores branch for the merged/closed PR.
func (c *Controller) RestoreBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	pullreqNum int64,
	in *RestoreBranchInput,
) (types.CreateBranchOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, pullreqNum)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}
	if pr.State == enum.PullReqStateOpen {
		return types.CreateBranchOutput{}, nil, errors.Conflict("source branch %q already exists", pr.SourceBranch)
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to fetch rules: %w", err)
	}
	violations, err := rules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		Actor:       &session.Principal,
		AllowBypass: in.BypassRules,
		IsRepoOwner: isRepoOwner,
		Repo:        repo,
		RefAction:   protection.RefActionCreate,
		RefType:     protection.RefTypeBranch,
		RefNames:    []string{pr.SourceBranch},
	})
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		return types.CreateBranchOutput{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return types.CreateBranchOutput{}, violations, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	rpcOut, err := c.git.CreateBranch(ctx, &git.CreateBranchParams{
		WriteParams: writeParams,
		BranchName:  pr.SourceBranch,
		Target:      pr.SourceSHA,
	})
	if err != nil {
		return types.CreateBranchOutput{}, nil, err
	}
	branch, err := controller.MapBranch(rpcOut.Branch)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to map branch: %w", err)
	}

	err = func() error {
		if pr, err = c.pullreqStore.UpdateActivitySeq(ctx, pr); err != nil {
			return fmt.Errorf("failed to update pull request activity sequence: %w", err)
		}

		_, err := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID,
			&types.PullRequestActivityPayloadBranchRestore{SHA: pr.SourceSHA}, nil)
		return err
	}()
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity for successful branch restore")
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeCreateBranch,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   repo.ID,
			instrument.PropertyRepositoryName: repo.Identifier,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create branch operation: %s", err)
	}

	return types.CreateBranchOutput{
		Branch: branch,
	}, nil, nil
}
