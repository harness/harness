// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListPaths lists all paths of a space.
 */
func (c *Controller) ListPaths(ctx context.Context, session *auth.Session,
	spaceRef string, pathFilter *types.PathFilter) ([]*types.Path, error) {
	space, err := findSpaceFromRef(ctx, c.spaceStore, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, err
	}

	return c.spaceStore.ListAllPaths(ctx, space.ID, pathFilter)
}
