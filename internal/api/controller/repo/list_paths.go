// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListPaths lists all paths of a repo.
 */
func (c *Controller) ListPaths(ctx context.Context, session *auth.Session,
	repoRef string, filter *types.PathFilter) ([]*types.Path, int64, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, 0, err
	}

	count, err := c.repoStore.CountPaths(ctx, repo.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count paths: %w", err)
	}

	paths, err := c.repoStore.ListPaths(ctx, repo.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list paths: %w", err)
	}

	return paths, count, nil
}
