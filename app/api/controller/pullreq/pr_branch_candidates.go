// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"time"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	PRBannerDuration            = 2 * time.Hour
	PRBannerDefaultLimitForPage = 3
	PRBannerMaxLimitForPage     = 10
)

// PRBranchCandidates finds branch names updated by the current user that don't have PRs.
func (c *Controller) PRBranchCandidates(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	limit uint64,
) ([]types.BranchTable, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	cutOffTime := time.Now().Add(-PRBannerDuration)

	// Request extra branch to account for potentially filtering out default branch
	branches, err := c.branchStore.FindBranchesWithoutPRs(
		ctx,
		repo.ID,
		session.Principal.ID,
		cutOffTime.UnixMilli(),
		limit+1,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Filter out default branch if present
	result := make([]types.BranchTable, 0, len(branches))
	for _, branch := range branches {
		if branch.Name != repo.DefaultBranch {
			result = append(result, branch)
		}
		// Stop once we've reached the limit
		if uint64(len(result)) >= limit {
			break
		}
	}

	return result, nil
}
