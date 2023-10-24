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
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Branch struct {
	Name   string        `json:"name"`
	SHA    string        `json:"sha"`
	Commit *types.Commit `json:"commit,omitempty"`
}

// ListBranches lists the branches of a repo.
func (c *Controller) ListBranches(ctx context.Context,
	session *auth.Session,
	repoRef string,
	includeCommit bool,
	filter *types.BranchFilter,
) ([]Branch, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, true)
	if err != nil {
		return nil, err
	}

	rpcOut, err := c.gitRPCClient.ListBranches(ctx, &gitrpc.ListBranchesParams{
		ReadParams:    gitrpc.CreateRPCReadParams(repo),
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
	var commit *types.Commit
	if b.Commit != nil {
		var err error
		commit, err = controller.MapCommit(b.Commit)
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
