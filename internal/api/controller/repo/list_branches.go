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

type Branch struct {
	Name   string  `json:"name"`
	Commit *Commit `json:"commit,omitempty"`
}

/*
* ListBranches lists the branches of a repo.
 */
func (c *Controller) ListBranches(ctx context.Context, session *auth.Session,
	repoRef string, includeCommit bool, branchFilter *types.BranchFilter) ([]Branch, int64, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, 0, err
	}

	rpcOut, err := c.gitRPCClient.ListBranches(ctx, &gitrpc.ListBranchesParams{
		RepoUID:       repo.GitUID,
		IncludeCommit: includeCommit,
		Page:          int32(branchFilter.Page),
		PageSize:      int32(branchFilter.Size),
	})
	if err != nil {
		return nil, 0, err
	}

	branches := make([]Branch, len(rpcOut.Branches))
	for i := range rpcOut.Branches {
		branches[i], err = mapBranch(rpcOut.Branches[i])
		if err != nil {
			return nil, 0, fmt.Errorf("failed to map branch: %w", err)
		}
	}

	return branches, rpcOut.TotalCount, nil
}

func mapBranch(b gitrpc.Branch) (Branch, error) {
	var commit *Commit
	if b.Commit != nil {
		var err error
		commit, err = mapCommit(b.Commit)
		if err != nil {
			return Branch{}, err
		}
	}
	return Branch{
		Name:   b.Name,
		Commit: commit,
	}, nil
}
