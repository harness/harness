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

// MoveInput is used for moving a space.
type MoveInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
	// ParentRef can be either a space ID or space path
	ParentRef *string `json:"parent_ref"`
}

func (i *MoveInput) hasChanges(
	space *types.Space,
	currentParentSpace *types.SpaceCore,
	targetParentSpace *types.SpaceCore,
) bool {
	if i.Identifier != nil && *i.Identifier != space.Identifier {
		return true
	}

	if i.ParentRef != nil && targetParentSpace.ID != currentParentSpace.ID {
		return true
	}

	return false
}

// Move moves a space to a new identifier.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *MoveInput,
) (*SpaceOutput, error) {
	spaceCore, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionSpaceEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	if err = c.sanitizeMoveInput(in, spaceCore.ParentID == 0); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	space, err := c.spaceStore.Find(ctx, spaceCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find space by ID: %w", err)
	}

	currentParentSpace, err := c.spaceFinder.FindByID(ctx, space.ParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to find current parent space: %w", err)
	}

	targetParentSpace := currentParentSpace
	if in.ParentRef != nil {
		targetParentSpace, err = c.getSpaceCheckAuthSpaceCreation(ctx, session, *in.ParentRef)
		if err != nil {
			return nil, fmt.Errorf("failed to access target parent space: %w", err)
		}
	}

	// exit early if there are no changes
	if !in.hasChanges(space, currentParentSpace, targetParentSpace) {
		return GetSpaceOutput(ctx, c.publicAccess, space)
	}

	if err = c.spaceSvc.MoveNoAuth(
		ctx,
		session,
		space,
		in.Identifier,
		targetParentSpace.Path,
	); err != nil {
		return nil, err
	}

	return GetSpaceOutput(ctx, c.publicAccess, space)
}

func (c *Controller) sanitizeMoveInput(in *MoveInput, isRoot bool) error {
	if in.Identifier == nil {
		in.Identifier = in.UID
	}

	if in.Identifier != nil {
		if err := c.identifierCheck(*in.Identifier, isRoot); err != nil {
			return err
		}
	}

	return nil
}
