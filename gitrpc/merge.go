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

package gitrpc

import (
	"context"
	"time"

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

	// Committer overwrites the git committer used for committing the files
	// (optional, default: actor)
	Committer *Identity
	// CommitterDate overwrites the git committer date used for committing the files
	// (optional, default: current time on server)
	CommitterDate *time.Time
	// Author overwrites the git author used for committing the files
	// (optional, default: committer)
	Author *Identity
	// AuthorDate overwrites the git author date used for committing the files
	// (optional, default: committer date)
	AuthorDate *time.Time

	RefType enum.RefType
	RefName string

	// HeadExpectedSHA is commit sha on the head branch, if HeadExpectedSHA is older
	// than the HeadBranch latest sha then merge will fail.
	HeadExpectedSHA string

	Force            bool
	DeleteHeadBranch bool

	Method enum.MergeMethod
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
		AuthorDate:       mapToRPCTimeOptional(params.AuthorDate),
		Committer:        mapToRPCIdentityOptional(params.Committer),
		CommitterDate:    mapToRPCTimeOptional(params.CommitterDate),
		RefType:          rpc.RefType(params.RefType),
		RefName:          params.RefName,
		HeadExpectedSha:  params.HeadExpectedSHA,
		Force:            params.Force,
		DeleteHeadBranch: params.DeleteHeadBranch,
		Method:           params.Method.ToRPC(),
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
