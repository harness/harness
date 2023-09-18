// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// FileViewList lists all files of the PR marked as viewed for the user.
func (c *Controller) FileViewList(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) ([]*types.PullReqFileView, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	fileViews, err := c.fileViewStore.List(ctx, pr.ID, session.Principal.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to read file view entries for user from db: %w", err)
	}

	return fileViews, nil
}
