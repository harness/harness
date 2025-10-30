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
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// CreateCommitTagInput used for tag creation apis.
type CreateCommitTagInput struct {
	Name string `json:"name"`
	// Target is the commit (or points to the commit) the new tag will be pointing to.
	// If no target is provided, the tag points to the same commit as the default branch of the repo.
	Target string `json:"target"`

	// Message is the optional message the tag will be created with - if the message is empty
	// the tag will be lightweight, otherwise it'll be annotated.
	Message string `json:"message"`

	DryRunRules bool `json:"dry_run_rules"`
	BypassRules bool `json:"bypass_rules"`
}

// CreateCommitTag creates a new tag for a repo.
func (c *Controller) CreateCommitTag(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateCommitTagInput,
) (types.CreateCommitTagOutput, []types.RuleViolations, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return types.CreateCommitTagOutput{}, nil, err
	}

	// set target to default branch in case no branch or commit was provided
	if in.Target == "" {
		in.Target = repo.DefaultBranch
	}

	rules, isRepoOwner, err := c.fetchTagRules(ctx, session, repo)
	if err != nil {
		return types.CreateCommitTagOutput{}, nil, err
	}

	violations, err := rules.RefChangeVerify(ctx, protection.RefChangeVerifyInput{
		ResolveUserGroupID: c.userGroupService.ListUserIDsByGroupIDs,
		Actor:              &session.Principal,
		AllowBypass:        in.BypassRules,
		IsRepoOwner:        isRepoOwner,
		Repo:               repo,
		RefAction:          protection.RefActionCreate,
		RefType:            protection.RefTypeTag,
		RefNames:           []string{in.Name},
	})
	if err != nil {
		return types.CreateCommitTagOutput{}, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if in.DryRunRules {
		return types.CreateCommitTagOutput{
			DryRunRulesOutput: types.DryRunRulesOutput{
				DryRunRules:    true,
				RuleViolations: violations,
			},
		}, nil, nil
	}

	if protection.IsCritical(violations) {
		return types.CreateCommitTagOutput{}, violations, nil
	}

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return types.CreateCommitTagOutput{}, nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	now := time.Now()
	rpcOut, err := c.git.CreateCommitTag(ctx, &git.CreateCommitTagParams{
		WriteParams: writeParams,
		Name:        in.Name,
		Target:      in.Target,
		Message:     in.Message,
		Tagger:      identityFromPrincipal(session.Principal),
		TaggerDate:  &now,
	})
	if err != nil {
		return types.CreateCommitTagOutput{}, nil, err
	}

	commitTag := controller.MapCommitTag(rpcOut.CommitTag)

	if protection.IsBypassed(violations) {
		err = c.auditService.Log(ctx,
			session.Principal,
			audit.NewResource(
				audit.ResourceTypeRepository,
				repo.Identifier,
				audit.RepoPath,
				repo.Path,
				audit.BypassedResourceType,
				audit.BypassedResourceTypeTag,
				audit.BypassedResourceName,
				commitTag.Name,
				audit.BypassAction,
				audit.BypassActionCreated,
				audit.ResourceName,
				fmt.Sprintf(
					audit.BypassSHALabelFormat,
					repo.Identifier,
					commitTag.Name,
				),
			),
			audit.ActionBypassed,
			paths.Parent(repo.Path),
			audit.WithNewObject(audit.CommitTagObject{
				TagName:        commitTag.Name,
				RepoPath:       repo.Path,
				RuleViolations: violations,
			}),
		)
		if err != nil {
			log.Ctx(ctx).Warn().Msgf("failed to insert audit log for create tag operation: %s", err)
		}
	}

	err = c.instrumentation.Track(ctx, instrument.Event{
		Type:      instrument.EventTypeCreateTag,
		Principal: session.Principal.ToPrincipalInfo(),
		Path:      repo.Path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   repo.ID,
			instrument.PropertyRepositoryName: repo.Identifier,
		},
	})
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create tag operation: %s", err)
	}

	return types.CreateCommitTagOutput{
		CommitTag: commitTag,
		DryRunRulesOutput: types.DryRunRulesOutput{
			RuleViolations: violations,
		},
	}, nil, nil
}
