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
* ListSpaces lists the child spaces of a space.
 */
func (c *Controller) ListSpaces(ctx context.Context, session *auth.Session,
	spaceRef string, spaceFilter *types.SpaceFilter) (int64, []*types.Space, error) {
	space, err := findSpaceFromRef(ctx, c.spaceStore, spaceRef)
	if err != nil {
		return 0, nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, true); err != nil {
		return 0, nil, err
	}

	count, err := c.spaceStore.Count(ctx, space.ID)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to count child spaces: %w", err)
	}

	spaces, err := c.spaceStore.List(ctx, space.ID, spaceFilter)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to list child spaces: %w", err)
	}

	/*
	 * TODO: needs access control? Might want to avoid that (makes paging and performance hard)
	 */
	return count, spaces, nil
}
