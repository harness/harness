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

package space

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RuleList returns protection rules for a repository.
func (c *Controller) RuleList(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	inherited bool,
	filter *types.RuleFilter,
) ([]types.Rule, int64, error) {
	space, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	list, count, err := c.rulesSvc.List(ctx, space.ID, enum.RuleParentSpace, inherited, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list space-level protection rules: %w", err)
	}

	return list, count, nil
}
