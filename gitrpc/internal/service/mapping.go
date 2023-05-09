// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func mapSortOrder(s rpc.SortOrder) types.SortOrder {
	switch s {
	case rpc.SortOrder_Asc:
		return types.SortOrderAsc
	case rpc.SortOrder_Desc:
		return types.SortOrderDesc
	case rpc.SortOrder_Default:
		return types.SortOrderDefault
	default:
		// no need to error out - just use default for sorting
		return types.SortOrderDefault
	}
}

func mapListCommitTagsSortOption(s rpc.ListCommitTagsRequest_SortOption) types.GitReferenceField {
	switch s {
	case rpc.ListCommitTagsRequest_Date:
		return types.GitReferenceFieldCreatorDate
	case rpc.ListCommitTagsRequest_Name:
		return types.GitReferenceFieldRefName
	case rpc.ListCommitTagsRequest_Default:
		return types.GitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return types.GitReferenceFieldRefName
	}
}

func mapListBranchesSortOption(s rpc.ListBranchesRequest_SortOption) types.GitReferenceField {
	switch s {
	case rpc.ListBranchesRequest_Date:
		return types.GitReferenceFieldCreatorDate
	case rpc.ListBranchesRequest_Name:
		return types.GitReferenceFieldRefName
	case rpc.ListBranchesRequest_Default:
		return types.GitReferenceFieldRefName
	default:
		// no need to error out - just use default for sorting
		return types.GitReferenceFieldRefName
	}
}

// TODO: Add UTs to ensure enum values match!
func mapGitNodeType(t types.TreeNodeType) rpc.TreeNodeType {
	return rpc.TreeNodeType(t)
}

// TODO: Add UTs to ensure enum values match!
func mapGitMode(m types.TreeNodeMode) rpc.TreeNodeMode {
	return rpc.TreeNodeMode(m)
}

func mapGitBranch(gitBranch *types.Branch) (*rpc.Branch, error) {
	if gitBranch == nil {
		return nil, status.Errorf(codes.Internal, "git branch is nil")
	}

	var commit *rpc.Commit
	var err error
	if gitBranch.Commit != nil {
		commit, err = mapGitCommit(gitBranch.Commit)
		if err != nil {
			return nil, err
		}
	}

	return &rpc.Branch{
		Name:   gitBranch.Name,
		Sha:    gitBranch.SHA,
		Commit: commit,
	}, nil
}

func mapGitCommit(gitCommit *types.Commit) (*rpc.Commit, error) {
	if gitCommit == nil {
		return nil, status.Errorf(codes.Internal, "git commit is nil")
	}

	return &rpc.Commit{
		Sha:       gitCommit.SHA,
		Title:     gitCommit.Title,
		Message:   gitCommit.Message,
		Author:    mapGitSignature(gitCommit.Author),
		Committer: mapGitSignature(gitCommit.Committer),
	}, nil
}

func mapGitSignature(gitSignature types.Signature) *rpc.Signature {
	return &rpc.Signature{
		Identity: &rpc.Identity{
			Name:  gitSignature.Identity.Name,
			Email: gitSignature.Identity.Email,
		},
		When: gitSignature.When.Unix(),
	}
}
func mapHunkHeader(hunkHeader types.HunkHeader) *rpc.HunkHeader {
	return &rpc.HunkHeader{
		OldLine: int32(hunkHeader.OldLine),
		OldSpan: int32(hunkHeader.OldSpan),
		NewLine: int32(hunkHeader.NewLine),
		NewSpan: int32(hunkHeader.NewSpan),
		Text:    hunkHeader.Text,
	}
}

func mapDiffFileHeader(h types.DiffFileHeader) *rpc.DiffFileHeader {
	return &rpc.DiffFileHeader{
		OldFileName: h.OldFileName,
		NewFileName: h.NewFileName,
		Extensions:  h.Extensions,
	}
}

func mapDiffFileHunkHeaders(diffHunkHeaders []*types.DiffFileHunkHeaders) []*rpc.DiffFileHunkHeaders {
	res := make([]*rpc.DiffFileHunkHeaders, len(diffHunkHeaders))
	for i, diffHunkHeader := range diffHunkHeaders {
		hunkHeaders := make([]*rpc.HunkHeader, len(diffHunkHeader.HunksHeaders))
		for j, hunkHeader := range diffHunkHeader.HunksHeaders {
			hunkHeaders[j] = mapHunkHeader(hunkHeader)
		}
		res[i] = &rpc.DiffFileHunkHeaders{
			FileHeader:  mapDiffFileHeader(diffHunkHeader.FileHeader),
			HunkHeaders: hunkHeaders,
		}
	}
	return res
}

func mapRenameDetails(renameDetails *types.PathRenameDetails) *rpc.RenameDetails {
	if renameDetails == nil {
		return &rpc.RenameDetails{IsRenamed: false}
	}

	return &rpc.RenameDetails{IsRenamed: renameDetails.Renamed,
		OldPath: renameDetails.OldPath,
		NewPath: renameDetails.NewPath}
}
