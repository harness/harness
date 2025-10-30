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
	"github.com/harness/gitness/types/enum"
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

	err = c.rulesSvc.Delete(
		ctx,
		&session.Principal,
		enum.RuleParentRepo, repo.ID,
		repo.Identifier, repo.Path,
		identifier,
	)
	if err != nil {
		return fmt.Errorf("failed to delete repo-level protection rule: %w", err)
	}

	return nil
}
