// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// Recheck re-checks all system PR checks (mergeability check, ...).
func (c *Controller) Recheck(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
) error {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	err = c.pullreqService.UpdateMergeDataIfRequired(ctx, repo.ID, prNum)
	if err != nil {
		return fmt.Errorf("failed to refresh merge data: %w", err)
	}

	return nil
}
