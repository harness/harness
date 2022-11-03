// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/rpc"
	"github.com/rs/zerolog/log"
)

type BranchSortOption int

const (
	BranchSortOptionDefault BranchSortOption = iota
	BranchSortOptionName
	BranchSortOptionDate
)

type ListBranchesParams struct {
	// RepoUID is the uid of the git repository
	RepoUID       string
	IncludeCommit bool
	Query         string
	Sort          BranchSortOption
	Order         SortOrder
	Page          int32
	PageSize      int32
}

type ListBranchesOutput struct {
	Branches []Branch
}

type Branch struct {
	Name   string
	SHA    string
	Commit *Commit
}

func (c *Client) ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	stream, err := c.repoService.ListBranches(ctx, &rpc.ListBranchesRequest{
		RepoUid:       params.RepoUID,
		IncludeCommit: params.IncludeCommit,
		Query:         params.Query,
		Sort:          mapToRPCListBranchesSortOption(params.Sort),
		Order:         mapToRPCSortOrder(params.Order),
		Page:          params.Page,
		PageSize:      params.PageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start stream for branches: %w", err)
	}

	// NOTE: don't use PageSize as initial slice capacity - as that theoretically could be MaxInt
	output := &ListBranchesOutput{
		Branches: make([]Branch, 0, 16),
	}
	for {
		var next *rpc.ListBranchesResponse
		next, err = stream.Recv()
		if errors.Is(err, io.EOF) {
			log.Ctx(ctx).Debug().Msg("received end of stream")
			break
		}
		if err != nil {
			return nil, fmt.Errorf("received unexpected error from rpc: %w", err)
		}
		if next.GetBranch() == nil {
			return nil, fmt.Errorf("expected branch message")
		}

		var branch *Branch
		branch, err = mapRPCBranch(next.GetBranch())
		if err != nil {
			return nil, fmt.Errorf("failed to map rpc branch: %w", err)
		}

		output.Branches = append(output.Branches, *branch)
	}

	return output, nil
}
