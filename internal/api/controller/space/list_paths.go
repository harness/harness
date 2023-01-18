// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListPaths lists all paths of a space.
 */
func (c *Controller) ListPaths(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.PathFilter) ([]*types.Path, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, false); err != nil {
		return nil, 0, err
	}

	var (
		paths []*types.Path
		count int64
	)
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		paths, err = c.pathStore.List(ctx, enum.PathTargetTypeSpace, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list paths: %w", err)
		}

		if filter.Page == 1 && len(paths) < filter.Size {
			count = int64(len(paths))
			return nil
		}

		count, err = c.pathStore.Count(ctx, enum.PathTargetTypeSpace, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count paths: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return paths, count, nil
}
