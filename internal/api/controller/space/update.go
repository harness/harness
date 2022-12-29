// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a space.
type UpdateInput struct {
	Description *string `json:"description"`
	IsPublic    *bool   `json:"is_public"`
}

// Update updates a space.
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	spaceRef string, in *UpdateInput) (*types.Space, error) {
	space, err := c.spaceStore.FindSpaceFromRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return nil, err
	}

	// update values only if provided
	if in.Description != nil {
		space.Description = *in.Description
	}
	if in.IsPublic != nil {
		space.IsPublic = *in.IsPublic
	}

	// always update time
	space.Updated = time.Now().UnixMilli()

	// ensure provided values are valid
	if err = c.spaceCheck(space); err != nil {
		return nil, err
	}

	err = c.spaceStore.Update(ctx, space)
	if err != nil {
		return nil, err
	}

	return space, nil
}
