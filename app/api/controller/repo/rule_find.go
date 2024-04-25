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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RuleFind returns the protection rule by identifier.
func (c *Controller) RuleFind(ctx context.Context,
	session *auth.Session,
	repoRef string,
	identifier string,
) (*types.Rule, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	r, err := c.ruleStore.FindByIdentifier(ctx, nil, &repo.ID, identifier)
	if err != nil {
		return nil, fmt.Errorf("failed to find repository-level protection rule by identifier: %w", err)
	}

	r.Users, err = c.getRuleUsers(ctx, r)
	if err != nil {
		return nil, err
	}

	return r, nil
}
