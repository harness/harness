// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
