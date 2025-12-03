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
	"strings"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a space.
type UpdateInput struct {
	Description *string `json:"description"`
}

func (in *UpdateInput) hasChanges(space *types.Space) bool {
	return in.Description != nil && *in.Description != space.Description
}

// Update updates a space.
func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *UpdateInput,
) (*SpaceOutput, error) {
	spaceCore, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	space, err := c.spaceStore.Find(ctx, spaceCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by ID: %w", err)
	}

	if !in.hasChanges(space) {
		return GetSpaceOutput(ctx, c.publicAccess, space)
	}

	if err = c.sanitizeUpdateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	space, err = c.spaceStore.UpdateOptLock(ctx, space, func(space *types.Space) error {
		// update values only if provided
		if in.Description != nil {
			space.Description = *in.Description
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return GetSpaceOutput(ctx, c.publicAccess, space)
}

func (c *Controller) sanitizeUpdateInput(in *UpdateInput) error {
	if in.Description != nil {
		*in.Description = strings.TrimSpace(*in.Description)
		if err := check.Description(*in.Description); err != nil {
			return err
		}
	}

	return nil
}
