// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"
)

type DiffService struct {
	rpc.UnimplementedDiffServiceServer
	adapter      GitAdapter
	reposRoot    string
	reposTempDir string
}

func NewDiffService(adapter GitAdapter, reposRoot string, reposTempDir string) (*DiffService, error) {
	return &DiffService{
		adapter:      adapter,
		reposRoot:    reposRoot,
		reposTempDir: reposTempDir,
	}, nil
}

func (s DiffService) RawDiff(request *rpc.DiffRequest, stream rpc.DiffService_RawDiffServer) error {
	err := validateDiffRequest(request)
	if err != nil {
		return err
	}

	ctx := stream.Context()
	base := request.GetBase()

	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.RawDiffResponse{Data: p})
	})

	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	args := []string{}
	if request.GetMergeBase() {
		args = []string{
			"--merge-base",
		}
	}

	return s.adapter.RawDiff(ctx, repoPath, request.GetBaseRef(), request.GetHeadRef(), sw, args...)
}

func validateDiffRequest(in *rpc.DiffRequest) error {
	if in.GetBase() == nil {
		return types.ErrBaseCannotBeEmpty
	}
	if in.GetBaseRef() == "" {
		return types.ErrEmptyBaseRef
	}
	if in.GetHeadRef() == "" {
		return types.ErrEmptyHeadRef
	}

	return nil
}

func (s DiffService) DiffShortStat(ctx context.Context, r *rpc.DiffRequest) (*rpc.DiffShortStatResponse, error) {
	err := validateDiffRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to validate request for short diff statistic "+
			"between %s and %s with err: %w", r.GetBaseRef(), r.GetHeadRef(), err)
	}

	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	// direct comparison
	// when direct is false then its like you use --merge-base
	// to find best common ancestor(s) between two refs
	direct := !r.GetMergeBase()

	stat, err := s.adapter.DiffShortStat(ctx, repoPath, r.GetBaseRef(), r.GetHeadRef(), direct)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch short statistics "+
			"between %s and %s with err: %w", r.GetBaseRef(), r.GetHeadRef(), err)
	}

	return &rpc.DiffShortStatResponse{
		Files:     int32(stat.Files),
		Additions: int32(stat.Additions),
		Deletions: int32(stat.Deletions),
	}, nil
}

func (s DiffService) GetDiffHunkHeaders(
	ctx context.Context,
	r *rpc.GetDiffHunkHeadersRequest,
) (*rpc.GetDiffHunkHeadersResponse, error) {
	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	hunkHeaders, err := s.adapter.GetDiffHunkHeaders(ctx, repoPath, r.TargetCommitSha, r.SourceCommitSha)
	if err != nil {
		return nil, processGitErrorf(err, "failed to get diff hunk headers between two commits")
	}

	return &rpc.GetDiffHunkHeadersResponse{
		Files: mapDiffFileHunkHeaders(hunkHeaders),
	}, nil
}

func (s DiffService) DiffCut(
	ctx context.Context,
	r *rpc.DiffCutRequest,
) (*rpc.DiffCutResponse, error) {
	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	mergeBase, _, err := s.adapter.GetMergeBase(ctx, repoPath, "", r.TargetBranch, r.SourceBranch)
	if err != nil {
		return nil, processGitErrorf(err, "failed to find merge base")
	}

	sourceCommits, err := s.adapter.ListCommitSHAs(ctx, repoPath, r.SourceBranch, r.TargetBranch, 0, 1)
	if err != nil || len(sourceCommits) == 0 {
		return nil, processGitErrorf(err, "failed to get list of source branch commits")
	}

	diffHunkHeader, linesHunk, err := s.adapter.DiffCut(ctx,
		repoPath,
		r.TargetCommitSha, r.SourceCommitSha,
		r.Path,
		types.DiffCutParams{
			LineStart:    int(r.LineStart),
			LineStartNew: r.LineStartNew,
			LineEnd:      int(r.LineEnd),
			LineEndNew:   r.LineEndNew,
			BeforeLines:  2,
			AfterLines:   2,
			LineLimit:    40,
		})
	if err != nil {
		return nil, processGitErrorf(err, "failed to get diff hunk")
	}

	return &rpc.DiffCutResponse{
		HunkHeader:      mapHunkHeader(diffHunkHeader),
		LinesHeader:     linesHunk.HunkHeader.String(),
		Lines:           linesHunk.Lines,
		MergeBaseSha:    mergeBase,
		LatestSourceSha: sourceCommits[0],
	}, nil
}
