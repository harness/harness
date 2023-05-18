// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListSpaces lists the child spaces of a space.
func (c *Controller) ListSpaces(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.SpaceFilter) ([]*types.Space, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, true); err != nil {
		return nil, 0, err
	}
	return c.ListSpacesNoAuth(ctx, space.ID, filter)
}

// List spaces WITHOUT checking PermissionSpaceView.
func (c *Controller) ListSpacesNoAuth(
	ctx context.Context,
	spaceID int64,
	filter *types.SpaceFilter,
) ([]*types.Space, int64, error) {
	count, err := c.spaceStore.Count(ctx, spaceID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count child spaces: %w", err)
	}

	spaces, err := c.spaceStore.List(ctx, spaceID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list child spaces: %w", err)
	}

	/*
	 * TODO: needs access control? Might want to avoid that (makes paging and performance hard)
	 */
	return spaces, count, nil
}
