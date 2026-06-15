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

package repo

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MergeQueueFullInfo describes the merge queue setup for a branch
// together with the ordered list of queued pull requests.
type MergeQueueFullInfo struct {
	Active  bool                       `json:"active"`
	Setup   protection.MergeQueueSetup `json:"setup"`
	Entries []types.MergeQueueListItem `json:"entries"`
}

// ListMergeQueueEntries returns the entire merge queue for the given branch in queue order.
// Each item pairs the queued pull request with its merge queue info (state, position, checks).
func (c *Controller) ListMergeQueueEntries(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	branch string,
) (*MergeQueueFullInfo, error) {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return nil, usererror.BadRequest("Branch must be provided.")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to repo: %w", err)
	}

	repoLevelBranchRules, err := c.protectionManager.ListOnlyRepoBranchRules(ctx, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo-level rules for the repository: %w", err)
	}

	setup, err := repoLevelBranchRules.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         repo,
		TargetBranch: branch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	entries, err := c.mergeQueueService.ListMergeQueue(ctx, repo, branch, setup)
	if err != nil {
		return nil, fmt.Errorf("failed to list merge queue: %w", err)
	}

	return &MergeQueueFullInfo{
		Active:  setup.IsActive(),
		Setup:   setup,
		Entries: entries,
	}, nil
}
