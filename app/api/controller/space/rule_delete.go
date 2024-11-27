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
	"github.com/harness/gitness/types/enum"
)

// RuleDelete deletes a protection rule by identifier.
func (c *Controller) RuleDelete(ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
) error {
	space, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return fmt.Errorf("failed to acquire access to space: %w", err)
	}

	err = c.rulesSvc.Delete(
		ctx,
		&session.Principal,
		enum.RuleParentSpace,
		space.ID,
		space.Identifier,
		space.Path,
		identifier,
	)
	if err != nil {
		return fmt.Errorf("failed to delete space-level protection rule: %w", err)
	}

	return nil
}
