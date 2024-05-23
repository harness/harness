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
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// CreateBranchInput used for branch creation apis.
type CreateBranchInput struct {
	Name string `json:"name"`

	// Target is the commit (or points to the commit) the new branch will be pointing to.
	// If no target is provided, the branch points to the same commit as the default branch of the repo.
	Target string `json:"target"`

	BypassRules bool `json:"bypass_rules"`
}

// CreateBranch creates a new branch for a repo.
func (c *Controller) CreateBranch(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateBranchInput,
) (*Branch, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, nil, err
	}

	// set target to default branch in case no target was provided
	if in.Target == "" {
		in.Target = repo.DefaultBranch
	}

	rules, isRepoOwner, err := c.fetchRules(ctx, session, repo)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}
	if protection.IsCritical(violations) {
		return nil, violations, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	rpcOut, err := c.git.CreateBranch(ctx, &git.CreateBranchParams{
		WriteParams: writeParams,
		BranchName:  in.Name,
		Target:      in.Target,
	})
	if err != nil {
		return nil, nil, err
	}

	branch, err := mapBranch(rpcOut.Branch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to map branch: %w", err)
	}

	return &branch, nil, nil
}
