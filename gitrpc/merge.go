// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/rpc"
)

// MergeParams is input structure object for merging operation.
type MergeParams struct {
	WriteParams
	BaseBranch string
	// HeadRepoUID specifies the UID of the repo that contains the head branch (required for forking).
	// WARNING: This field is currently not supported yet!
	HeadRepoUID string
	HeadBranch  string
	Title       string
	Message     string

	// Committer overwrites the git committer used for committing the files (optional, default: actor)
	Committer *Identity
	// Author overwrites the git author used for committing the files (optional, default: committer)
	Author *Identity

	RefType enum.RefType
	RefName string

	// HeadExpectedSHA is commit sha on the head branch, if HeadExpectedSHA is older
	// than the HeadBranch latest sha then merge will fail.
	HeadExpectedSHA string

	Force            bool
	DeleteHeadBranch bool
}

// MergeOutput is result object from merging and returns
// base, head and commit sha.
type MergeOutput struct {
	// BaseSHA is the sha of the latest commit on the base branch that was used for merging.
	BaseSHA string
	// HeadSHA is the sha of the latest commit on the head branch that was used for merging.
	HeadSHA string
	// MergeBaseSHA is the sha of the merge base of the HeadSHA and BaseSHA
	MergeBaseSHA string
	// MergeSHA is the sha of the commit after merging HeadSHA with BaseSHA.
	MergeSHA string
}

// Merge method executes git merge operation. Refs can be sha, branch or tag.
// Based on input params.RefType merge can do checking or final merging of two refs.
// some examples:
//
//	params.RefType = Undefined -> discard merge commit (only performs a merge check).
//	params.RefType = Raw and params.RefName = refs/pull/1/ref will push to refs/pullreq/1/ref
//	params.RefType = RefTypeBranch and params.RefName = "somebranch" -> merge and push to refs/heads/somebranch
//	params.RefType = RefTypePullReqHead and params.RefName = "1" -> merge and push to refs/pullreq/1/head
//	params.RefType = RefTypePullReqMerge and params.RefName = "1" -> merge and push to refs/pullreq/1/merge
//
// There are cases when you want to block merging and for that you will need to provide
// params.HeadExpectedSHA which will be compared with the latest sha from head branch
// if they are not the same error will be returned.
func (c *Client) Merge(ctx context.Context, params *MergeParams) (MergeOutput, error) {
	if params == nil {
		return MergeOutput{}, ErrNoParamsProvided
	}

	resp, err := c.mergeService.Merge(ctx, &rpc.MergeRequest{
		Base:             mapToRPCWriteRequest(params.WriteParams),
		BaseBranch:       params.BaseBranch,
		HeadBranch:       params.HeadBranch,
		Title:            params.Title,
		Message:          params.Message,
		Author:           mapToRPCIdentityOptional(params.Author),
		Committer:        mapToRPCIdentityOptional(params.Committer),
		RefType:          rpc.RefType(params.RefType),
		RefName:          params.RefName,
		HeadExpectedSha:  params.HeadExpectedSHA,
		Force:            params.Force,
		DeleteHeadBranch: params.DeleteHeadBranch,
	})
	if err != nil {
		return MergeOutput{}, processRPCErrorf(err, "merging failed")
	}

	return MergeOutput{
		BaseSHA:      resp.GetBaseSha(),
		HeadSHA:      resp.GetHeadSha(),
		MergeBaseSHA: resp.GetMergeBaseSha(),
		MergeSHA:     resp.GetMergeSha(),
	}, nil
}
