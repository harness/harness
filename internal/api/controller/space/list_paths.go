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

/*
* ListPaths lists all paths of a space.
 */
func (c *Controller) ListPaths(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.PathFilter) ([]*types.Path, int64, error) {
	space, err := c.spaceStore.FindSpaceFromRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, 0, err
	}

	count, err := c.spaceStore.CountPaths(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count paths: %w", err)
	}

	paths, err := c.spaceStore.ListPaths(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list paths: %w", err)
	}

	return paths, count, nil
}
