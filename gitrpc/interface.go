// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitrpc

import (
	"context"
	"io"
)

type Interface interface {
	CreateRepository(ctx context.Context, params *CreateRepositoryParams) (*CreateRepositoryOutput, error)
	GetTreeNode(ctx context.Context, params *GetTreeNodeParams) (*GetTreeNodeOutput, error)
	ListTreeNodes(ctx context.Context, params *ListTreeNodeParams) (*ListTreeNodeOutput, error)
	GetSubmodule(ctx context.Context, params *GetSubmoduleParams) (*GetSubmoduleOutput, error)
	GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error)
	CreateBranch(ctx context.Context, params *CreateBranchParams) (*CreateBranchOutput, error)
	DeleteBranch(ctx context.Context, params *DeleteBranchParams) error
	ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error)
	GetRef(ctx context.Context, params *GetRefParams) (*GetRefResponse, error)

	/*
	 * Commits service
	 */
	GetCommit(ctx context.Context, params *GetCommitParams) (*GetCommitOutput, error)
	ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error)
	ListCommitTags(ctx context.Context, params *ListCommitTagsParams) (*ListCommitTagsOutput, error)
	GetCommitDivergences(ctx context.Context, params *GetCommitDivergencesParams) (*GetCommitDivergencesOutput, error)
	CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error)

	/*
	 * Git Cli Service
	 */
	GetInfoRefs(ctx context.Context, w io.Writer, params *InfoRefsParams) error
	ServicePack(ctx context.Context, w io.Writer, params *ServicePackParams) error

	/*
	 * Diff services
	 */
	RawDiff(ctx context.Context, in *RawDiffParams, w io.Writer) error

	/*
	 * Merge services
	 */
	MergeBranch(ctx context.Context, in *MergeBranchParams) (MergeBranchOutput, error)
}
