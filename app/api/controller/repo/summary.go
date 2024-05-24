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

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Summary returns commit, branch, tag and pull req count for a repo.
func (c *Controller) Summary(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) (*types.RepositorySummary, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, fmt.Errorf("access check failed: %w", err)
	}

	summary, err := c.git.Summary(ctx, git.SummaryParams{ReadParams: git.CreateReadParams(repo)})
	if err != nil {
		return nil, fmt.Errorf("failed to get repo summary: %w", err)
	}

	return &types.RepositorySummary{
		DefaultBranchCommitCount: summary.CommitCount,
		BranchCount:              summary.BranchCount,
		TagCount:                 summary.TagCount,
		PullReqSummary: types.RepositoryPullReqSummary{
			OpenCount:   repo.NumOpenPulls,
			ClosedCount: repo.NumClosedPulls,
			MergedCount: repo.NumMergedPulls,
		},
	}, nil
}
