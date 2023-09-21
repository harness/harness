// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gitrpc

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/check"
	"github.com/harness/gitness/gitrpc/rpc"

	"github.com/rs/zerolog/log"
)

type BranchSortOption int

const (
	BranchSortOptionDefault BranchSortOption = iota
	BranchSortOptionName
	BranchSortOptionDate
)

type CreateBranchParams struct {
	WriteParams
	// BranchName is the name of the branch
	BranchName string
	// Target is a git reference (branch / tag / commit SHA)
	Target string
}

type CreateBranchOutput struct {
	Branch Branch
}

type GetBranchParams struct {
	ReadParams
	// BranchName is the name of the branch
	BranchName string
}

type GetBranchOutput struct {
	Branch Branch
}

type DeleteBranchParams struct {
	WriteParams
	// Name is the name of the branch
	BranchName string
}

type ListBranchesParams struct {
	ReadParams
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

func (c *Client) CreateBranch(ctx context.Context, params *CreateBranchParams) (*CreateBranchOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	if err := check.BranchName(params.BranchName); err != nil {
		return nil, ErrInvalidArgumentf(err.Error())
	}

	resp, err := c.refService.CreateBranch(ctx, &rpc.CreateBranchRequest{
		Base:       mapToRPCWriteRequest(params.WriteParams),
		Target:     params.Target,
		BranchName: params.BranchName,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to create '%s' branch on server", params.BranchName)
	}

	var branch *Branch
	branch, err = mapRPCBranch(resp.Branch)
	if err != nil {
		return nil, processRPCErrorf(err, "failed to map rpc branch %v", resp.Branch)
	}

	return &CreateBranchOutput{
		Branch: *branch,
	}, nil
}

func (c *Client) GetBranch(ctx context.Context, params *GetBranchParams) (*GetBranchOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}
	resp, err := c.refService.GetBranch(ctx, &rpc.GetBranchRequest{
		Base:       mapToRPCReadRequest(params.ReadParams),
		BranchName: params.BranchName,
	})
	if err != nil {
		return nil, processRPCErrorf(err, "failed to get branch from server")
	}

	var branch *Branch
	branch, err = mapRPCBranch(resp.GetBranch())
	if err != nil {
		return nil, fmt.Errorf("failed to map rpc branch: %w", err)
	}

	return &GetBranchOutput{
		Branch: *branch,
	}, nil
}

func (c *Client) DeleteBranch(ctx context.Context, params *DeleteBranchParams) error {
	if params == nil {
		return ErrNoParamsProvided
	}
	_, err := c.refService.DeleteBranch(ctx, &rpc.DeleteBranchRequest{
		Base:       mapToRPCWriteRequest(params.WriteParams),
		BranchName: params.BranchName,
		// TODO: what are scenarios where we wouldn't want to force delete?
		// Branch protection is a different story, and build on top application layer.
		Force: true,
	})
	if err != nil {
		return processRPCErrorf(err, "failed to delete branch on server")
	}

	return nil
}

func (c *Client) ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error) {
	if params == nil {
		return nil, ErrNoParamsProvided
	}

	stream, err := c.refService.ListBranches(ctx, &rpc.ListBranchesRequest{
		Base:          mapToRPCReadRequest(params.ReadParams),
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
			return nil, processRPCErrorf(err, "received unexpected error from rpc")
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
