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
	"fmt"

	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Delete deletes a protection rule by identifier.
func (s *Service) Delete(ctx context.Context,
	principal *types.Principal,
	parentType enum.RuleParent,
	parentID int64,
	scopeIdentifier string,
	path string,
	identifier string,
) error {
	rule, err := s.ruleStore.FindByIdentifier(ctx, parentType, parentID, identifier)
	if err != nil {
		return fmt.Errorf("failed to find protection rule by identifier: %w", err)
	}

	err = s.ruleStore.Delete(ctx, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to delete protection rule: %w", err)
	}

	nameKey := audit.RepoName
	if parentType == enum.RuleParentSpace {
		nameKey = audit.SpaceName
	}
	err = s.auditService.Log(ctx,
		*principal,
		audit.NewResource(
			audit.ResourceTypeBranchRule,
			rule.Identifier,
			nameKey,
			scopeIdentifier,
		),
		audit.ActionDeleted,
		paths.Parent(path),
		audit.WithOldObject(rule),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete branch rule operation: %s", err)
	}

	return nil
}
