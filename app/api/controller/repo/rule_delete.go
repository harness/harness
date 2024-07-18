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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// RuleDelete deletes a protection rule by identifier.
func (c *Controller) RuleDelete(ctx context.Context,
	session *auth.Session,
	repoRef string,
	identifier string,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return err
	}

	r, err := c.ruleStore.FindByIdentifier(ctx, nil, &repo.ID, identifier)
	if err != nil {
		return fmt.Errorf("failed to find repository-level protection rule by identifier: %w", err)
	}

	err = c.ruleStore.Delete(ctx, r.ID)
	if err != nil {
		return fmt.Errorf("failed to delete repository-level protection rule: %w", err)
	}

	err = c.auditService.Log(ctx,
		session.Principal,
		audit.NewResource(audit.ResourceTypeBranchRule, r.Identifier, audit.RepoName, repo.Identifier),
		audit.ActionDeleted,
		paths.Parent(repo.Path),
		audit.WithOldObject(r),
	)
	if err != nil {
		log.Ctx(ctx).Warn().Msgf("failed to insert audit log for delete branch rule operation: %s", err)
	}

	return nil
}
