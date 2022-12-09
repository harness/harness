// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// List returns a list of pull requests from the provided repository.
func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.PullReqFilter,
) ([]*types.PullReq, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	if filter.SourceRepoRef == repoRef {
		filter.SourceRepoID = repo.ID
	} else if filter.SourceRepoRef != "" {
		var sourceRepo *types.Repository
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, filter.SourceRepoRef, enum.PermissionRepoView)
		if err != nil {
			return nil, err
		}
		filter.SourceRepoID = sourceRepo.ID
	}

	return c.pullreqStore.List(ctx, repo.ID, filter)
}
