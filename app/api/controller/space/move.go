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
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MoveInput is used for moving a space.
type MoveInput struct {
	// TODO [CODE-1363]: remove after identifier migration.
	UID        *string `json:"uid" deprecated:"true"`
	Identifier *string `json:"identifier"`
}

func (i *MoveInput) hasChanges(space *types.Space) bool {
	if i.Identifier != nil && *i.Identifier != space.Identifier {
		return true
	}

	return false
}

// Move moves a space to a new identifier.
// TODO: Add support for moving to other parents and alias.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	in *MoveInput,
) (*SpaceOutput, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit); err != nil {
		return nil, err
	}

	if err = c.sanitizeMoveInput(in, space.ParentID == 0); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	// exit early if there are no changes
	if !in.hasChanges(space) {
		return GetSpaceOutput(ctx, c.publicAccess, space)
	}

	if err = c.moveInner(
		ctx,
		session,
		space,
		in.Identifier,
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

func (c *Controller) moveInner(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
	inIdentifier *string,
) error {
	return c.tx.WithTx(ctx, func(ctx context.Context) error {
		// delete old primary segment
		err := c.spacePathStore.DeletePrimarySegment(ctx, space.ID)
		if err != nil {
			return fmt.Errorf("failed to delete primary path segment: %w", err)
		}

		// update space with move inputs
		if inIdentifier != nil {
			space.Identifier = *inIdentifier
		}

		// add new primary segment using updated space data
		now := time.Now().UnixMilli()
		newPrimarySegment := &types.SpacePathSegment{
			ParentID:   space.ParentID,
			Identifier: space.Identifier,
			SpaceID:    space.ID,
			IsPrimary:  true,
			CreatedBy:  session.Principal.ID,
			Created:    now,
			Updated:    now,
		}
		err = c.spacePathStore.InsertSegment(ctx, newPrimarySegment)
		if err != nil {
			return fmt.Errorf("failed to create new primary path segment: %w", err)
		}

		// update space itself
		err = c.spaceStore.Update(ctx, space)
		if err != nil {
			return fmt.Errorf("failed to update the space in the db: %w", err)
		}

		return nil
	})
}
