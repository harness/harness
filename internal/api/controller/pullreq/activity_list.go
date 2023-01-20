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

// ActivityList returns a list of pull request activities
// from the provided repository and pull request number.
func (c *Controller) ActivityList(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	prNum int64,
	filter *types.PullReqActivityFilter,
) ([]*types.PullReqActivity, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, repo.ID, prNum)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pull request by number: %w", err)
	}

	list, err := c.activityStore.List(ctx, pr.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list pull requests activities: %w", err)
	}

	// the function returns deleted comments, but it removes their content
	for _, act := range list {
		if act.Deleted != nil {
			act.Text = ""
			_ = act.SetPayload(nil)
		}
	}

	if filter.Limit == 0 {
		return list, int64(len(list)), nil
	}

	count, err := c.activityStore.Count(ctx, pr.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count pull request activities: %w", err)
	}

	return list, count, nil
}
