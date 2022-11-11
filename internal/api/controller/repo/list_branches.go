// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Branch struct {
	Name   string  `json:"name"`
	SHA    string  `json:"sha"`
	Commit *Commit `json:"commit,omitempty"`
}

/*
* ListBranches lists the branches of a repo.
 */
func (c *Controller) ListBranches(ctx context.Context, session *auth.Session,
	repoRef string, includeCommit bool, filter *types.BranchFilter) ([]Branch, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoView, false); err != nil {
		return nil, err
	}

	rpcOut, err := c.gitRPCClient.ListBranches(ctx, &gitrpc.ListBranchesParams{
		RepoUID:       repo.GitUID,
		IncludeCommit: includeCommit,
		Query:         filter.Query,
		Sort:          mapToRPCBranchSortOption(filter.Sort),
		Order:         mapToRPCSortOrder(filter.Order),
		Page:          int32(filter.Page),
		PageSize:      int32(filter.Size),
	})
	if err != nil {
		return nil, err
	}

	branches := make([]Branch, len(rpcOut.Branches))
	for i := range rpcOut.Branches {
		branches[i], err = mapBranch(rpcOut.Branches[i])
		if err != nil {
			return nil, fmt.Errorf("failed to map branch: %w", err)
		}
	}

	return branches, nil
}

func mapToRPCBranchSortOption(o enum.BranchSortOption) gitrpc.BranchSortOption {
	switch o {
	case enum.BranchSortOptionDate:
		return gitrpc.BranchSortOptionDate
	case enum.BranchSortOptionName:
		return gitrpc.BranchSortOptionName
	case enum.BranchSortOptionDefault:
		return gitrpc.BranchSortOptionDefault
	default:
		// no need to error out - just use default for sorting
		return gitrpc.BranchSortOptionDefault
	}
}

func mapToRPCSortOrder(o enum.Order) gitrpc.SortOrder {
	switch o {
	case enum.OrderAsc:
		return gitrpc.SortOrderAsc
	case enum.OrderDesc:
		return gitrpc.SortOrderDesc
	case enum.OrderDefault:
		return gitrpc.SortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return gitrpc.SortOrderDefault
	}
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
		SHA:    b.SHA,
		Commit: commit,
	}, nil
}
