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

package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/harness/gitness/gitrpc/diff"
	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.RawDiffResponse{Data: p})
	})

	return s.rawDiff(stream.Context(), request, sw)
}

func (s DiffService) rawDiff(ctx context.Context, request *rpc.DiffRequest, w io.Writer) error {
	err := validateDiffRequest(request)
	if err != nil {
		return err
	}

	base := request.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	err = s.adapter.RawDiff(ctx, repoPath, request.GetBaseRef(), request.GetHeadRef(), request.MergeBase, w)
	if err != nil {
		return processGitErrorf(err, "failed to fetch diff "+
			"between %s and %s", request.GetBaseRef(), request.GetHeadRef())
	}
	return nil
}

func (s DiffService) CommitDiff(request *rpc.CommitDiffRequest, stream rpc.DiffService_CommitDiffServer) error {
	err := validateCommitDiffRequest(request)
	if err != nil {
		return err
	}

	base := request.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	sw := streamio.NewWriter(func(p []byte) error {
		return stream.Send(&rpc.CommitDiffResponse{Data: p})
	})

	return s.adapter.CommitDiff(stream.Context(), repoPath, request.Sha, sw)
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

func validateCommitDiffRequest(in *rpc.CommitDiffRequest) error {
	if in.Base == nil {
		return types.ErrBaseCannotBeEmpty
	}

	if !isValidGitSHA(in.Sha) {
		return status.Errorf(codes.InvalidArgument, "the provided commit sha '%s' is of invalid format.", in.Sha)
	}

	return nil
}

func (s DiffService) DiffShortStat(ctx context.Context, r *rpc.DiffRequest) (*rpc.DiffShortStatResponse, error) {
	err := validateDiffRequest(r)
	if err != nil {
		return nil, fmt.Errorf("failed to validate request for short diff statistic, error: %w", err)
	}

	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	stat, err := s.adapter.DiffShortStat(ctx, repoPath, r.GetBaseRef(), r.GetHeadRef(), r.GetMergeBase())
	if err != nil {
		return nil, processGitErrorf(err, "failed to fetch short statistics "+
			"between %s and %s", r.GetBaseRef(), r.GetHeadRef())
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

func (s DiffService) DiffFileStat(
	ctx context.Context,
	r *rpc.DiffRequest,
) (*rpc.DiffFileStatResponse, error) {
	base := r.GetBase()
	repoPath := getFullPathForRepo(s.reposRoot, base.GetRepoUid())

	files, err := s.adapter.DiffFileStat(ctx, repoPath, r.BaseRef, r.HeadRef)
	if err != nil {
		return nil, processGitErrorf(err, "failed to get diff file stat")
	}
	return &rpc.DiffFileStatResponse{Files: files}, nil
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

	sourceCommits, err := s.adapter.ListCommitSHAs(ctx, repoPath, r.SourceBranch, 0, 1,
		types.CommitFilter{AfterRef: r.TargetBranch})
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

func (s DiffService) Diff(request *rpc.DiffRequest, stream rpc.DiffService_DiffServer) error {
	done := make(chan bool)
	defer close(done)

	pr, pw := io.Pipe()
	defer pr.Close()

	parser := diff.Parser{
		Reader: bufio.NewReader(pr),
	}

	go func() {
		defer pw.Close()
		err := s.rawDiff(stream.Context(), request, pw)
		if err != nil {
			return
		}
	}()

	return parser.Parse(func(f *diff.File) error {
		return streamDiffFile(f, request.IncludePatch, stream)
	})
}

func streamDiffFile(f *diff.File, includePatch bool, stream rpc.DiffService_DiffServer) error {
	var status rpc.DiffResponse_FileStatus
	switch f.Type {
	case diff.FileAdd:
		status = rpc.DiffResponse_ADDED
	case diff.FileChange:
		status = rpc.DiffResponse_MODIFIED
	case diff.FileDelete:
		status = rpc.DiffResponse_DELETED
	case diff.FileRename:
		status = rpc.DiffResponse_RENAMED
	default:
		status = rpc.DiffResponse_UNDEFINED
	}

	patch := bytes.Buffer{}
	if includePatch {
		for _, sec := range f.Sections {
			for _, line := range sec.Lines {
				if line.Type != diff.DiffLinePlain {
					patch.WriteString(line.Content)
				}
			}
		}
	}

	err := stream.Send(&rpc.DiffResponse{
		Path:      f.Path,
		OldPath:   f.OldPath,
		Sha:       f.SHA,
		OldSha:    f.OldSHA,
		Status:    status,
		Additions: int32(f.NumAdditions()),
		Deletions: int32(f.NumDeletions()),
		Changes:   int32(f.NumChanges()),
		Patch:     patch.Bytes(),
	})
	if err != nil {
		return fmt.Errorf("failed to send diff response on stream: %w", err)
	}
	return nil
}
