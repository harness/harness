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
	"encoding/json"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type RuleCreateInput struct {
	Type  types.RuleType `json:"type"`
	State enum.RuleState `json:"state"`
	// TODO [CODE-1363]: remove after identifier migration.
	UID         string             `json:"uid" deprecated:"true"`
	Identifier  string             `json:"identifier"`
	Description string             `json:"description"`
	Pattern     protection.Pattern `json:"pattern"`
	Definition  json.RawMessage    `json:"definition"`
}

// sanitize validates and sanitizes the create rule input data.
func (in *RuleCreateInput) sanitize() error {
	// TODO [CODE-1363]: remove after identifier migration.
	if in.Identifier == "" {
		in.Identifier = in.UID
	}

	if err := check.Identifier(in.Identifier); err != nil {
		return err
	}

	if err := in.Pattern.Validate(); err != nil {
		return usererror.BadRequestf("invalid pattern: %s", err)
	}

	var ok bool
	in.State, ok = in.State.Sanitize()
	if !ok {
		return usererror.BadRequest("rule state is invalid")
	}

	if in.Type == "" {
		in.Type = protection.TypeBranch
	}

	if len(in.Definition) == 0 {
		return usererror.BadRequest("rule definition missing")
	}

	return nil
}

// RuleCreate creates a new protection rule for a repo.
func (c *Controller) RuleCreate(ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *RuleCreateInput,
) (*types.Rule, error) {
	if err := in.sanitize(); err != nil {
		return nil, err
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	in.Definition, err = c.protectionManager.SanitizeJSON(in.Type, in.Definition)
	if err != nil {
		return nil, usererror.BadRequestf("invalid rule definition: %s", err.Error())
	}

	now := time.Now().UnixMilli()
	r := &types.Rule{
		CreatedBy:     session.Principal.ID,
		Created:       now,
		Updated:       now,
		RepoID:        &repo.ID,
		SpaceID:       nil,
		Type:          in.Type,
		State:         in.State,
		Identifier:    in.Identifier,
		Description:   in.Description,
		Pattern:       in.Pattern.JSON(),
		Definition:    in.Definition,
		CreatedByInfo: types.PrincipalInfo{},
	}

	err = c.ruleStore.Create(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository-level protection rule: %w", err)
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeBranchRule, r.Identifier, audit.RepoName, repo.Identifier),
		audit.ActionCreated,
		paths.Parent(repo.Path),
		audit.WithNewObject(r),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for create branch rule operation: %s", err)
	}

	r.Users, err = c.getRuleUsers(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
