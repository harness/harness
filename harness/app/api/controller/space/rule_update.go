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
	"github.com/harness/gitness/app/services/rules"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// RuleUpdate updates an existing protection rule for a space.
func (c *Controller) RuleUpdate(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	in *rules.UpdateInput,
) (*types.Rule, error) {
	space, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	rule, err := c.rulesSvc.Update(
		ctx,
		&session.Principal,
		enum.RuleParentSpace,
		space.ID,
		space.Identifier,
		space.Path,
		identifier,
		in,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update space-level protection rule by identifier: %w", err)
	}

	return rule, nil
}
