// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListCommits lists the commits of a repo.
 */
func (c *Controller) ListCommits(ctx context.Context, session *auth.Session,
	repoRef string, gitRef string, filter *types.CommitFilter, path string,
	since int64, until int64) ([]types.Commit, *types.RenameDetails, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, nil, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	rpcOut, err := c.gitRPCClient.ListCommits(ctx, &gitrpc.ListCommitsParams{
		ReadParams: CreateRPCReadParams(repo),
		GitREF:     gitRef,
		After:      filter.After,
		Page:       int32(filter.Page),
		Limit:      int32(filter.Limit),
		Path:       path,
		Since:      since,
		Until:      until,
	})
	if err != nil {
		return nil, nil, err
	}

	commits := make([]types.Commit, len(rpcOut.Commits))
	for i := range rpcOut.Commits {
		var commit *types.Commit
		commit, err = controller.MapCommit(&rpcOut.Commits[i])
		if err != nil {
			return nil, nil, fmt.Errorf("failed to map commit: %w", err)
		}
		commits[i] = *commit
	}

	var renameDetails = types.RenameDetails{
		IsRenamed: false,
	}
	if rpcOut.RenameDetails != nil {
		renameDetails.IsRenamed = rpcOut.RenameDetails.IsRenamed
		renameDetails.OldPath = rpcOut.RenameDetails.OldPath
		renameDetails.NewPath = rpcOut.RenameDetails.NewPath
	}
	return commits, &renameDetails, nil
}
