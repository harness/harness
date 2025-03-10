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

package git

import (
	"context"
	"io"

	"github.com/harness/gitness/git/api"
)

type Interface interface {
	CreateRepository(ctx context.Context, params *CreateRepositoryParams) (*CreateRepositoryOutput, error)
	DeleteRepository(ctx context.Context, params *DeleteRepositoryParams) error
	GetTreeNode(ctx context.Context, params *GetTreeNodeParams) (*GetTreeNodeOutput, error)
	ListTreeNodes(ctx context.Context, params *ListTreeNodeParams) (*ListTreeNodeOutput, error)
	ListPaths(ctx context.Context, params *ListPathsParams) (*ListPathsOutput, error)
	GetSubmodule(ctx context.Context, params *GetSubmoduleParams) (*GetSubmoduleOutput, error)
	GetBlob(ctx context.Context, params *GetBlobParams) (*GetBlobOutput, error)
	CreateBranch(ctx context.Context, params *CreateBranchParams) (*CreateBranchOutput, error)
	CreateCommitTag(ctx context.Context, params *CreateCommitTagParams) (*CreateCommitTagOutput, error)
	DeleteTag(ctx context.Context, params *DeleteTagParams) error
	GetBranch(ctx context.Context, params *GetBranchParams) (*GetBranchOutput, error)
	DeleteBranch(ctx context.Context, params *DeleteBranchParams) error
	ListBranches(ctx context.Context, params *ListBranchesParams) (*ListBranchesOutput, error)
	UpdateDefaultBranch(ctx context.Context, params *UpdateDefaultBranchParams) error
	GetRef(ctx context.Context, params GetRefParams) (GetRefResponse, error)
	PathsDetails(ctx context.Context, params PathsDetailsParams) (PathsDetailsOutput, error)
	Summary(ctx context.Context, params SummaryParams) (SummaryOutput, error)

	// GetRepositorySize calculates the size of a repo in KiB.
	GetRepositorySize(ctx context.Context, params *GetRepositorySizeParams) (*GetRepositorySizeOutput, error)
	// UpdateRef creates, updates or deletes a git ref. If the OldValue is defined it must match the reference value
	// prior to the call. To remove a ref use the zero ref as the NewValue. To require the creation of a new one and
	// not update of an exiting one, set the zero ref as the OldValue.
	UpdateRef(ctx context.Context, params UpdateRefParams) error

	SyncRepository(ctx context.Context, params *SyncRepositoryParams) (*SyncRepositoryOutput, error)

	MatchFiles(ctx context.Context, params *MatchFilesParams) (*MatchFilesOutput, error)

	/*
	 * Commits service
	 */
	GetCommit(ctx context.Context, params *GetCommitParams) (*GetCommitOutput, error)
	ListCommits(ctx context.Context, params *ListCommitsParams) (*ListCommitsOutput, error)
	ListCommitTags(ctx context.Context, params *ListCommitTagsParams) (*ListCommitTagsOutput, error)
	GetCommitDivergences(ctx context.Context, params *GetCommitDivergencesParams) (*GetCommitDivergencesOutput, error)
	CommitFiles(ctx context.Context, params *CommitFilesParams) (CommitFilesResponse, error)
	MergeBase(ctx context.Context, params MergeBaseParams) (MergeBaseOutput, error)
	IsAncestor(ctx context.Context, params IsAncestorParams) (IsAncestorOutput, error)
	FindOversizeFiles(
		ctx context.Context,
		params *FindOversizeFilesParams,
	) (*FindOversizeFilesOutput, error)

	/*
	 * Git Cli Service
	 */
	GetInfoRefs(ctx context.Context, w io.Writer, params *InfoRefsParams) error
	ServicePack(ctx context.Context, params *ServicePackParams) error

	/*
	 * Diff services
	 */
	RawDiff(ctx context.Context, w io.Writer, in *DiffParams, files ...api.FileDiffRequest) error
	Diff(ctx context.Context, in *DiffParams, files ...api.FileDiffRequest) (<-chan *FileDiff, <-chan error)
	DiffFileNames(ctx context.Context, in *DiffParams) (DiffFileNamesOutput, error)
	CommitDiff(ctx context.Context, params *GetCommitParams, w io.Writer) error
	DiffShortStat(ctx context.Context, params *DiffParams) (DiffShortStatOutput, error)
	DiffStats(ctx context.Context, params *DiffParams) (DiffStatsOutput, error)

	GetDiffHunkHeaders(ctx context.Context, params GetDiffHunkHeadersParams) (GetDiffHunkHeadersOutput, error)
	DiffCut(ctx context.Context, params *DiffCutParams) (DiffCutOutput, error)

	/*
	 * Merge services
	 */
	Merge(ctx context.Context, in *MergeParams) (MergeOutput, error)

	/*
	 * Blame services
	 */
	Blame(ctx context.Context, params *BlameParams) (<-chan *BlamePart, <-chan error)
	PushRemote(ctx context.Context, params *PushRemoteParams) error

	GeneratePipeline(ctx context.Context, params *GeneratePipelineParams) (GeneratePipelinesOutput, error)

	/*
	 * Secret Scanning service
	 */
	ScanSecrets(ctx context.Context, param *ScanSecretsParams) (*ScanSecretsOutput, error)
	Archive(ctx context.Context, params ArchiveParams, w io.Writer) error
}
