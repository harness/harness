// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

// MergeBranchParams is input structure object for merging operation.
type MergeBranchParams struct {
	WriteParams
	BaseBranch       string
	HeadRepoUID      string
	HeadBranch       string
	Title            string
	Message          string
	Force            bool
	DeleteHeadBranch bool
}

// MergeBranchResult is result object from merging and returns
// base, head and commit sha.
type MergeBranchOutput struct {
	MergedSHA string
	BaseSHA   string
	HeadSHA   string
}

// MergeBranch merge head branch to base branch.
func (c *Client) MergeBranch(ctx context.Context, params *MergeBranchParams) (MergeBranchOutput, error) {
	if params == nil {
		return MergeBranchOutput{}, ErrNoParamsProvided
	}

	resp, err := c.mergeService.MergeBranch(ctx, &rpc.MergeBranchRequest{
		Base:             mapToRPCWriteRequest(params.WriteParams),
		BaseBranch:       params.BaseBranch,
		HeadBranch:       params.HeadBranch,
		Title:            params.Title,
		Message:          params.Message,
		Force:            params.Force,
		DeleteHeadBranch: params.DeleteHeadBranch,
	})
	if err != nil {
		return MergeBranchOutput{}, err
	}
	return MergeBranchOutput{
		MergedSHA: resp.GetMergeSha(),
		BaseSHA:   resp.GetBaseSha(),
		HeadSHA:   resp.GetHeadSha(),
	}, nil
}
