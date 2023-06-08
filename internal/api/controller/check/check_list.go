// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListChecks return an array of status check results for a commit in a repository.
func (c *Controller) ListChecks(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	commitSHA string,
) ([]*types.Check, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access access to repo: %w", err)
	}

	list, err := c.checkStore.List(ctx, repo.ID, commitSHA)
	if err != nil {
		return nil, fmt.Errorf("failed to list status check results for repo=%s: %w", repo.UID, err)
	}

	return list, nil
}
