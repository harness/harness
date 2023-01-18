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
* ListRepositories lists the repositories of a space.
 */
func (c *Controller) ListRepositories(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.RepoFilter) ([]*types.Repository, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionRepoView, true); err != nil {
		return nil, 0, err
	}

	count, err := c.repoStore.Count(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count child repos: %w", err)
	}

	repos, err := c.repoStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list child repos: %w", err)
	}

	// backfill URLs
	for _, repo := range repos {
		repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)
	}

	/*
	 * TODO: needs access control? Might want to avoid that (makes paging and performance hard)
	 */
	return repos, count, nil
}
