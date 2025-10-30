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
	"time"

	"github.com/harness/gitness/app/api/usererror"
	ruleevents "github.com/harness/gitness/app/events/rule"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const ruleScopeRepo = int64(0)

type CreateInput struct {
	Type  enum.RuleType  `json:"type"`
	State enum.RuleState `json:"state"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID         string                `json:"uid" deprecated:"true"`
	Identifier  string                `json:"identifier"`
	Description string                `json:"description"`
	Pattern     protection.Pattern    `json:"pattern"`
	RepoTarget  protection.RepoTarget `json:"repo_target"`
	Definition  json.RawMessage       `json:"definition"`
}

// sanitize validates and sanitizes the create rule input data.
func (in *CreateInput) sanitize() error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	if err := in.Pattern.Validate(); err != nil {
		return usererror.BadRequestf("Invalid pattern: %s", err)
	}

	if err := in.RepoTarget.Validate(); err != nil {
		return usererror.BadRequestf("Invalid repo target: %s", err)
	}

	var ok bool
	in.State, ok = in.State.Sanitize()
	if !ok {
		return usererror.BadRequest("Rule state is invalid")
	}

	if in.Type == "" {
		in.Type = protection.TypeBranch
	}

	if len(in.Definition) == 0 {
		return usererror.BadRequest("Rule definition missing")
	}

	return nil
}

// Create creates a new protection rule for a scope.
func (s *Service) Create(ctx context.Context,
	principal *types.Principal,
	parentType enum.RuleParent,
	parentID int64,
	scopeIdentifier string,
	path string,
	in *CreateInput,
) (*types.Rule, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	var err error
	in.Definition, err = s.protectionManager.SanitizeJSON(
		in.Type, in.Definition,
	)
	if err != nil {
		return nil, usererror.BadRequestf("Invalid rule definition: %s", err.Error())
	}

	scope := ruleScopeRepo
	if parentType == enum.RuleParentSpace {
		scope, err = s.spaceStore.GetTreeLevel(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get parent tree level: %w", err)
		}
	}
	now := time.Now().UnixMilli()
	rule := &types.Rule{
		CreatedBy:     principal.ID,
		Created:       now,
		Updated:       now,
		Type:          in.Type,
		State:         in.State,
		Identifier:    in.Identifier,
		Description:   in.Description,
		Pattern:       in.Pattern.JSON(),
		RepoTarget:    in.RepoTarget.JSON(),
		Definition:    in.Definition,
		Scope:         scope,
		CreatedByInfo: types.PrincipalInfo{},
	}

	spacePath := path
	nameKey := audit.RepoName
	if parentType == enum.RuleParentRepo {
		spacePath = paths.Parent(path)
		rule.RepoID = &parentID
	} else if parentType == enum.RuleParentSpace {
		nameKey = audit.SpaceName
		rule.SpaceID = &parentID
	}

	userMap, ruleUserIDs, userGroupMap, _, err := s.getRuleUserAndUserGroups(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule users and user groups: %w", err)
	}

	if err := s.ruleValidator.Validate(ctx, ruleUserIDs, userMap); err != nil {
		return nil, fmt.Errorf("failed to validate users: %w", err)
	}

	err = s.ruleStore.Create(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to create protection rule: %w", err)
	}

	err = s.auditService.Log(ctx,
		*principal,
		audit.NewResource(ruleTypeToResourceType(rule.Type), rule.Identifier, nameKey, scopeIdentifier),
		audit.ActionCreated,
		spacePath,
		audit.WithNewObject(rule),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for create rule operation: %s", err)
	}

	rule.Users = userMap
	rule.UserGroups = userGroupMap

	err = s.backfillRuleRepositories(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("failed to backfill rule repositories: %w", err)
	}

	var event instrument.Event
	if parentType == enum.RuleParentRepo {
		event = instrumentEventRepo(
			rule.ID, principal.ToPrincipalInfo(), parentID, scopeIdentifier, path,
		)
	} else if parentType == enum.RuleParentSpace {
		event = instrumentEventSpace(
			rule.ID, principal.ToPrincipalInfo(), parentID, scopeIdentifier, path,
		)
	}
	err = s.instrumentation.Track(ctx, event)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert instrumentation record for create branch rule operation: %s", err)
	}

	s.sendSSE(ctx, parentID, parentType, enum.SSETypeRuleCreated, rule)

	s.eventReporter.Created(ctx, &ruleevents.CreatedPayload{
		Base: ruleevents.Base{
			RuleID:      rule.ID,
			SpaceID:     rule.SpaceID,
			RepoID:      rule.RepoID,
			PrincipalID: rule.CreatedBy,
		},
	})

	return rule, nil
}

func instrumentEventRepo(
	ruleID int64,
	principalInfo *types.PrincipalInfo,
	scopeID int64,
	scopeIdentifier string,
	path string,
) instrument.Event {
	return instrument.Event{
		Type:      instrument.EventTypeCreateBranchRule,
		Principal: principalInfo,
		Path:      path,
		Properties: map[instrument.Property]any{
			instrument.PropertyRepositoryID:   scopeID,
			instrument.PropertyRepositoryName: scopeIdentifier,
			instrument.PropertyRuleID:         ruleID,
		},
	}
}

func instrumentEventSpace(
	ruleID int64,
	principalInfo *types.PrincipalInfo,
	scopeID int64,
	scopeIdentifier string,
	path string,
) instrument.Event {
	return instrument.Event{
		Type:      instrument.EventTypeCreateBranchRule,
		Principal: principalInfo,
		Path:      path,
		Properties: map[instrument.Property]any{
			instrument.PropertySpaceID:   scopeID,
			instrument.PropertySpaceName: scopeIdentifier,
			instrument.PropertyRuleID:    ruleID,
		},
	}
}
