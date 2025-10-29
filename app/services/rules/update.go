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

package rules

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type UpdateInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID         *string                `json:"uid" deprecated:"true"`
	Identifier  *string                `json:"identifier"`
	State       *enum.RuleState        `json:"state"`
	Description *string                `json:"description"`
	Pattern     *protection.Pattern    `json:"pattern"`
	RepoTarget  *protection.RepoTarget `json:"repo_target"`
	Definition  *json.RawMessage       `json:"definition"`
}

// sanitize validates and sanitizes the update rule input data.
func (in *UpdateInput) sanitize() error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := check.Identifier(*in.Identifier); err != nil {
			return err
		}
	}

	if in.State != nil {
		state, ok := in.State.Sanitize()
		if !ok {
			return usererror.BadRequest("Rule state is invalid")
		}

		in.State = &state
	}

	if in.Pattern != nil {
		if err := in.Pattern.Validate(); err != nil {
			return usererror.BadRequestf("Invalid pattern: %s", err)
		}
	}

	if in.RepoTarget != nil {
		if err := in.RepoTarget.Validate(); err != nil {
			return usererror.BadRequestf("Invalid repo target: %s", err)
		}
	}

	if in.Definition != nil && len(*in.Definition) == 0 {
		return usererror.BadRequest("Rule definition missing")
	}

	return nil
}

func (in *UpdateInput) isEmpty() bool {
	return in.Identifier == nil && in.State == nil && in.Description == nil && in.Pattern == nil && in.Definition == nil
}

// Update updates an existing protection rule for a repository.
func (s *Service) Update(ctx context.Context,
	principal *types.Principal,
	parentType enum.RuleParent,
	parentID int64,
	scopeIdentifier string,
	path string,
	identifier string,
	in *UpdateInput,
) (*types.Rule, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	rule, err := s.ruleStore.FindByIdentifier(ctx, parentType, parentID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to get a repository rule by its identifier: %w", err)
	}
	oldRule := rule.Clone()

	if in.isEmpty() {
		userMap, _, userGroupMap, _, err := s.getRuleUserAndUserGroups(ctx, rule)
		if err != nil {
			return nil, fmt.Errorf("failed to get rule users and user groups: %w", err)
		}

		rule.Users = userMap
		rule.UserGroups = userGroupMap

		return rule, nil
	}

	if in.Identifier != nil {
		rule.Identifier = *in.Identifier
	}
	if in.State != nil {
		rule.State = *in.State
	}
	if in.Description != nil {
		rule.Description = *in.Description
	}
	if in.Pattern != nil {
		rule.Pattern = in.Pattern.JSON()
	}
	if in.RepoTarget != nil {
		rule.RepoTarget = in.RepoTarget.JSON()
	}
	if in.Definition != nil {
		rule.Definition, err = s.protectionManager.SanitizeJSON(rule.Type, *in.Definition)
		if err != nil {
			return nil, fmt.Errorf("invalid rule definition: %w", err)
		}
	}

	userMap, ruleUserIDs, userGroupMap, _, err := s.getRuleUserAndUserGroups(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule users and user groups: %w", err)
	}

	if err := s.ruleValidator.Validate(ctx, ruleUserIDs, userMap); err != nil {
		return nil, fmt.Errorf("failed to validate users: %w", err)
	}

	rule.Users = userMap
	rule.UserGroups = userGroupMap

	err = s.backfillRuleRepositories(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to backfill rule repositories: %w", err)
	}

	if rule.IsEqual(&oldRule) {
		return rule, nil
	}

	err = s.ruleStore.Update(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to update repository-level protection rule: %w", err)
	}

	nameKey := audit.RepoName
	if parentType == enum.RuleParentSpace {
		nameKey = audit.SpaceName
	}
	err = s.auditService.Log(ctx,
		*principal,
		audit.NewResource(ruleTypeToResourceType(rule.Type), rule.Identifier, nameKey, scopeIdentifier),
		audit.ActionUpdated,
		paths.Parent(path),
		audit.WithOldObject(oldRule),
		audit.WithNewObject(rule),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for update rule operation: %s", err)
	}

	s.sendSSE(ctx, parentID, parentType, enum.SSETypeRuleUpdated, rule)

	return rule, nil
}
