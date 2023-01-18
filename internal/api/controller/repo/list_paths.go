// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

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
* ListPaths lists all paths of a repo.
 */
func (c *Controller) ListPaths(ctx context.Context, session *auth.Session,
	repoRef string, filter *types.PathFilter) ([]*types.Path, int64, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, 0, err
	}

	var (
		paths []*types.Path
		count int64
	)
	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		paths, err = c.pathStore.List(ctx, enum.PathTargetTypeRepo, repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list paths: %w", err)
		}

		if filter.Page == 1 && len(paths) < filter.Size {
			count = int64(len(paths))
			return nil
		}

		count, err = c.pathStore.Count(ctx, enum.PathTargetTypeRepo, repo.ID, filter)
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
