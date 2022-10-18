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

type Branch struct {
	Name    string `json:"name"`
	Default bool   `json:"default"`
	Commit  Commit `json:"commit"`
}

/*
* ListBranches lists the branches of a repo.
 */
func (c *Controller) ListBranches(ctx context.Context, session *auth.Session,
	repoRef string, branchFilter *types.BranchFilter) ([]Branch, int64, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, 0, err
	}

	rpcOut, err := c.gitRPCClient.ListBranches(ctx, &gitrpc.ListBranchesParams{
		RepoUID:  repo.GitUID,
		Page:     int32(branchFilter.Page),
		PageSize: int32(branchFilter.Size),
	})
	if err != nil {
		return nil, 0, err
	}

	branches := make([]Branch, len(rpcOut.Branches))
	for i := range rpcOut.Branches {
		branches[i] = mapBranch(rpcOut.Branches[i], repo.DefaultBranch)
	}

	return branches, rpcOut.TotalCount, nil
}

func mapBranch(b gitrpc.Branch, defaultBranch string) Branch {
	return Branch{
		Name:    b.Name,
		Default: b.Name == defaultBranch,
		Commit:  mapCommit(b.Commit),
	}
}
