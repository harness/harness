// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// DeletePath deletes a space path.
func (c *Controller) DeletePath(ctx context.Context, session *auth.Session, spaceRef string, pathID int64) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return err
	}

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		var path *types.Path
		path, err = c.pathStore.FindWithLock(ctx, pathID)
		if err != nil {
			return fmt.Errorf("failed to find path: %w", err)
		}

		if path.TargetType != enum.PathTargetTypeSpace || path.TargetID != space.ID {
			// return not found in case the path doesn't belong to this space!
			return fmt.Errorf("path doesn't belong to space - %w", usererror.ErrNotFound)
		}

		if path.IsPrimary {
			return usererror.ErrPrimaryPathCantBeDeleted
		}

		err = c.pathStore.Delete(ctx, pathID)
		if err != nil {
			return fmt.Errorf("failed to delete path: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
