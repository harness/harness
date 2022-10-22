// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
* ListCommits lists the commits of a repo.
 */
func (c *Controller) ListCommits(ctx context.Context, session *auth.Session,
	repoRef string, gitRef string, filter *types.CommitFilter) ([]Commit, int64, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, 0, err
	}

	// set gitRef to default branch in case an empty reference was provided
	if gitRef == "" {
		gitRef = repo.DefaultBranch
	}

	rpcOut, err := c.gitRPCClient.ListCommits(ctx, &gitrpc.ListCommitsParams{
		RepoUID:  repo.GitUID,
		GitREF:   gitRef,
		Page:     int32(filter.Page),
		PageSize: int32(filter.Size),
	})
	if err != nil {
		return nil, 0, err
	}

	commits := make([]Commit, len(rpcOut.Commits))
	for i := range rpcOut.Commits {
		var commit *Commit
		commit, err = mapCommit(&rpcOut.Commits[i])
		if err != nil {
			return nil, 0, fmt.Errorf("failed to map commit: %w", err)
		}
		commits[i] = *commit
	}

	return commits, rpcOut.TotalCount, nil
}
