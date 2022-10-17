// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

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
	repoRef string, gitRef string, commitFilter *types.CommitFilter) ([]Commit, int64, error) {
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
		Page:     int32(commitFilter.Page),
		PageSize: int32(commitFilter.Size),
	})
	if err != nil {
		return nil, 0, err
	}

	commits := make([]Commit, 0, len(rpcOut.Commits))
	for _, rpcCommit := range rpcOut.Commits {
		commit := mapCommit(rpcCommit)
		commits = append(commits, commit)
	}

	return commits, rpcOut.TotalCount, nil
}
