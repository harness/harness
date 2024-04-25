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

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListCommits lists the commits of a repo.
func (c *Controller) ListCommits(ctx context.Context,
	session *auth.Session,
	repoRef string,
	gitRef string,
	filter *types.CommitFilter,
) (types.ListCommitResponse, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return types.ListCommitResponse{}, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	rpcOut, err := c.git.ListCommits(ctx, &git.ListCommitsParams{
		ReadParams:   git.CreateReadParams(repo),
		GitREF:       gitRef,
		After:        filter.After,
		Page:         int32(filter.Page),
		Limit:        int32(filter.Limit),
		Path:         filter.Path,
		Since:        filter.Since,
		Until:        filter.Until,
		Committer:    filter.Committer,
		IncludeStats: filter.IncludeStats,
	})
	if err != nil {
		return types.ListCommitResponse{}, err
	}

	commits := make([]types.Commit, len(rpcOut.Commits))
	for i := range rpcOut.Commits {
		var commit *types.Commit
		commit, err = controller.MapCommit(&rpcOut.Commits[i])
		if err != nil {
			return types.ListCommitResponse{}, fmt.Errorf("failed to map commit: %w", err)
		}
		commits[i] = *commit
	}

	renameDetailList := make([]types.RenameDetails, len(rpcOut.RenameDetails))
	for i := range rpcOut.RenameDetails {
		renameDetails := controller.MapRenameDetails(rpcOut.RenameDetails[i])
		if renameDetails == nil {
			return types.ListCommitResponse{}, fmt.Errorf("rename details was nil")
		}
		renameDetailList[i] = *renameDetails
	}
	return types.ListCommitResponse{
		Commits:       commits,
		RenameDetails: renameDetailList,
		TotalCommits:  rpcOut.TotalCommits,
	}, nil
}
