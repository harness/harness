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
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// CreateBranchInput used for branch creation apis.
type CreateBranchInput struct {
	Name string `json:"name"`

	// Target is the commit (or points to the commit) the new branch will be pointing to.
	// If no target is provided, the branch points to the same commit as the default branch of the repo.
	Target string `json:"target"`

	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

// CreateBranch creates a new branch for a repo.
func (c *Controller) CreateBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateBranchInput,
) (types.CreateBranchOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.CreateBranchOutput{}, nil, err
	}

	// set target to default branch in case no target was provided
	if in.Target == "" {
		in.Target = repo.DefaultBranch
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return types.CreateBranchOutput{}, nil, err
	}

	violations, err := rules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		Actor:       &session.Principal,
		AllowBypass: in.BypassRules,
		IsRepoOwner: isRepoOwner,
		Repo:        repo,
		RefAction:   protection.RefActionCreate,
		RefType:     protection.RefTypeBranch,
		RefNames:    []string{in.Name},
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
		BranchName:  in.Name,
		Target:      in.Target,
	})
	if err != nil {
		return types.CreateBranchOutput{}, nil, err
	}

	branch, err := controller.MapBranch(rpcOut.Branch)
	if err != nil {
		return types.CreateBranchOutput{}, nil, fmt.Errorf("failed to map branch: %w", err)
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
				branch.Name,
				audit.BypassAction,
				audit.BypassActionCreated,
			),
			audit.ActionBypassed,
			paths.Parent(repo.Path),
			audit.WithNewObject(audit.BranchObject{
				BranchName:     branch.Name,
				RepoPath:       repo.Path,
				RuleViolations: violations,
			}),
		)
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert audit log for create branch operation: %s", err)
		}
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
