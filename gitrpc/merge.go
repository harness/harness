// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/rpc"
)

type MergeBranchParams struct {
	WriteParams
	BaseBranch   string
	HeadRepoUID  string
	HeadBranch   string
	Force        bool
	DeleteBranch bool
}

func (c *Client) MergeBranch(ctx context.Context, params *MergeBranchParams) (string, error) {
	if params == nil {
		return "", ErrNoParamsProvided
	}

	resp, err := c.mergeService.MergeBranch(ctx, &rpc.MergeBranchRequest{
		Base:       mapToRPCWriteRequest(params.WriteParams),
		Branch:     params.BaseBranch,
		HeadBranch: params.HeadBranch,
		Force:      params.Force,
		Delete:     params.DeleteBranch,
	})
	if err != nil {
		return "", err
	}
	return resp.CommitId, nil
}
