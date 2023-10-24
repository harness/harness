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
	"errors"
	"io"

	"github.com/harness/gitness/gitrpc/internal/streamio"
	"github.com/harness/gitness/gitrpc/internal/types"
	"github.com/harness/gitness/gitrpc/rpc"

	"golang.org/x/sync/errgroup"
)

type DiffParams struct {
	ReadParams
	BaseRef      string
	HeadRef      string
	MergeBase    bool
	IncludePatch bool
}

func (p DiffParams) Validate() error {
	if err := p.ReadParams.Validate(); err != nil {
		return err
	}

	if p.HeadRef == "" {
		return ErrInvalidArgumentf("head ref cannot be empty")
	}
	return nil
}

func (c *Client) RawDiff(ctx context.Context, params *DiffParams, out io.Writer) error {
	if err := params.Validate(); err != nil {
		return err
	}
	diff, err := c.diffService.RawDiff(ctx, &rpc.DiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return processRPCErrorf(err, "failed to fetch diff between '%s' and '%s' with err: %v",
			params.BaseRef, params.HeadRef, err)
	}

	reader := streamio.NewReader(func() ([]byte, error) {
		var resp *rpc.RawDiffResponse
		resp, err = diff.Recv()
		return resp.GetData(), err
	})

	if _, err = io.Copy(out, reader); err != nil {
		return processRPCErrorf(err, "failed to fetch diff between '%s' and '%s' with err: %v",
			params.BaseRef, params.HeadRef, err)
	}

	return nil
}

func (c *Client) CommitDiff(ctx context.Context, params *GetCommitParams, out io.Writer) error {
	if err := params.Validate(); err != nil {
		return err
	}
	diff, err := c.diffService.CommitDiff(ctx, &rpc.CommitDiffRequest{
		Base: mapToRPCReadRequest(params.ReadParams),
		Sha:  params.SHA,
	})
	if err != nil {
		return processRPCErrorf(err, "failed to fetch diff for commit '%s': %v", params.SHA, err)
	}

	reader := streamio.NewReader(func() ([]byte, error) {
		var resp *rpc.CommitDiffResponse
		resp, err = diff.Recv()
		return resp.GetData(), err
	})

	if _, err = io.Copy(out, reader); err != nil {
		return err
	}

	return nil
}

type DiffShortStatOutput struct {
	Files     int
	Additions int
	Deletions int
}

// DiffShortStat returns files changed, additions and deletions metadata.
func (c *Client) DiffShortStat(ctx context.Context, params *DiffParams) (DiffShortStatOutput, error) {
	if err := params.Validate(); err != nil {
		return DiffShortStatOutput{}, err
	}
	stat, err := c.diffService.DiffShortStat(ctx, &rpc.DiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return DiffShortStatOutput{}, processRPCErrorf(err, "failed to get diff data between '%s' and '%s'",
			params.BaseRef, params.HeadRef)
	}
	return DiffShortStatOutput{
		Files:     int(stat.GetFiles()),
		Additions: int(stat.GetAdditions()),
		Deletions: int(stat.GetDeletions()),
	}, nil
}

type DiffFileStatOutput struct {
	Files []string
}

func (c *Client) DiffFileStat(ctx context.Context, params *DiffParams) (DiffFileStatOutput, error) {
	if err := params.Validate(); err != nil {
		return DiffFileStatOutput{}, err
	}
	fileStat, err := c.diffService.DiffFileStat(ctx, &rpc.DiffRequest{
		Base:      mapToRPCReadRequest(params.ReadParams),
		BaseRef:   params.BaseRef,
		HeadRef:   params.HeadRef,
		MergeBase: params.MergeBase,
	})
	if err != nil {
		return DiffFileStatOutput{}, processRPCErrorf(err, "failed to get diff file data between '%s' and '%s'",
			params.BaseRef, params.HeadRef)
	}
	return DiffFileStatOutput{
		Files: fileStat.GetFiles(),
	}, nil
}

type DiffStatsOutput struct {
	Commits      int
	FilesChanged int
}

func (c *Client) DiffStats(ctx context.Context, params *DiffParams) (DiffStatsOutput, error) {
	// declare variables which will be used in go routines,
	// no need for atomic operations because writing and reading variable
	// doesn't happen at the same time
	var (
		totalCommits int
		totalFiles   int
	)

	errGroup, groupCtx := errgroup.WithContext(ctx)

	errGroup.Go(func() error {
		// read total commits

		options := &GetCommitDivergencesParams{
			ReadParams: params.ReadParams,
			Requests: []CommitDivergenceRequest{
				{
					From: params.HeadRef,
					To:   params.BaseRef,
				},
			},
		}

		rpcOutput, err := c.GetCommitDivergences(groupCtx, options)
		if err != nil {
			return processRPCErrorf(err, "failed to count pull request commits between '%s' and '%s'",
				params.BaseRef, params.HeadRef)
		}
		if len(rpcOutput.Divergences) > 0 {
			totalCommits = int(rpcOutput.Divergences[0].Ahead)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read short stat
		stat, err := c.DiffShortStat(groupCtx, &DiffParams{
			ReadParams: params.ReadParams,
			BaseRef:    params.BaseRef,
			HeadRef:    params.HeadRef,
			MergeBase:  true, // must be true, because commitDivergences use tripple dot notation
		})
		if err != nil {
			return err
		}
		totalFiles = stat.Files
		return nil
	})

	err := errGroup.Wait()
	if err != nil {
		return DiffStatsOutput{}, err
	}

	return DiffStatsOutput{
		Commits:      totalCommits,
		FilesChanged: totalFiles,
	}, nil
}

type GetDiffHunkHeadersParams struct {
	ReadParams
	SourceCommitSHA string
	TargetCommitSHA string
}

type DiffFileHeader struct {
	OldName    string
	NewName    string
	Extensions map[string]string
}

type HunkHeader struct {
	OldLine int
	OldSpan int
	NewLine int
	NewSpan int
	Text    string
}

type DiffFileHunkHeaders struct {
	FileHeader  DiffFileHeader
	HunkHeaders []HunkHeader
}

type GetDiffHunkHeadersOutput struct {
	Files []DiffFileHunkHeaders
}

func (c *Client) GetDiffHunkHeaders(
	ctx context.Context,
	params GetDiffHunkHeadersParams,
) (GetDiffHunkHeadersOutput, error) {
	if params.SourceCommitSHA == params.TargetCommitSHA {
		return GetDiffHunkHeadersOutput{}, nil
	}

	hunkHeaders, err := c.diffService.GetDiffHunkHeaders(ctx, &rpc.GetDiffHunkHeadersRequest{
		Base:            mapToRPCReadRequest(params.ReadParams),
		SourceCommitSha: params.SourceCommitSHA,
		TargetCommitSha: params.TargetCommitSHA,
	})
	if err != nil {
		return GetDiffHunkHeadersOutput{}, processRPCErrorf(err, "failed to get git diff hunk headers")
	}

	files := make([]DiffFileHunkHeaders, len(hunkHeaders.Files))
	for i, file := range hunkHeaders.Files {
		headers := make([]HunkHeader, len(file.HunkHeaders))
		for j, header := range file.HunkHeaders {
			headers[j] = mapHunkHeader(header)
		}
		files[i] = DiffFileHunkHeaders{
			FileHeader:  mapDiffFileHeader(file.FileHeader),
			HunkHeaders: headers,
		}
	}

	return GetDiffHunkHeadersOutput{
		Files: files,
	}, nil
}

type DiffCutOutput struct {
	Header          HunkHeader
	LinesHeader     string
	Lines           []string
	MergeBaseSHA    string
	LatestSourceSHA string
}

type DiffCutParams struct {
	ReadParams
	SourceCommitSHA string
	SourceBranch    string
	TargetCommitSHA string
	TargetBranch    string
	Path            string
	LineStart       int
	LineStartNew    bool
	LineEnd         int
	LineEndNew      bool
}

// DiffCut extracts diff snippet from a git diff hunk.
// The snippet is from the specific commit (specified by commit SHA), between refs
// source branch and target branch, from the specific file.
func (c *Client) DiffCut(ctx context.Context, params *DiffCutParams) (DiffCutOutput, error) {
	result, err := c.diffService.DiffCut(ctx, &rpc.DiffCutRequest{
		Base:            mapToRPCReadRequest(params.ReadParams),
		SourceCommitSha: params.SourceCommitSHA,
		SourceBranch:    params.SourceBranch,
		TargetCommitSha: params.TargetCommitSHA,
		TargetBranch:    params.TargetBranch,
		Path:            params.Path,
		LineStart:       int32(params.LineStart),
		LineStartNew:    params.LineStartNew,
		LineEnd:         int32(params.LineEnd),
		LineEndNew:      params.LineEndNew,
	})
	if err != nil {
		return DiffCutOutput{}, processRPCErrorf(err, "failed to get git diff sub hunk")
	}

	hunkHeader := types.HunkHeader{
		OldLine: int(result.HunkHeader.OldLine),
		OldSpan: int(result.HunkHeader.OldSpan),
		NewLine: int(result.HunkHeader.NewLine),
		NewSpan: int(result.HunkHeader.NewSpan),
		Text:    result.HunkHeader.Text,
	}

	return DiffCutOutput{
		Header:          HunkHeader(hunkHeader),
		LinesHeader:     result.LinesHeader,
		Lines:           result.Lines,
		MergeBaseSHA:    result.MergeBaseSha,
		LatestSourceSHA: result.LatestSourceSha,
	}, nil
}

type FileDiff struct {
	SHA         string         `json:"sha"`
	OldSHA      string         `json:"old_sha,omitempty"`
	Path        string         `json:"path"`
	OldPath     string         `json:"old_path,omitempty"`
	Status      FileDiffStatus `json:"status"`
	Additions   int64          `json:"additions"`
	Deletions   int64          `json:"deletions"`
	Changes     int64          `json:"changes"`
	Patch       []byte         `json:"patch,omitempty"`
	IsBinary    bool           `json:"is_binary"`
	IsSubmodule bool           `json:"is_submodule"`
}

type FileDiffStatus string

const (
	// NOTE: keeping values upper case for now to stay consistent with current API.
	// TODO: change drone/go-scm (and potentially new dependencies) to case insensitive.

	FileDiffStatusUndefined FileDiffStatus = "UNDEFINED"
	FileDiffStatusAdded     FileDiffStatus = "ADDED"
	FileDiffStatusModified  FileDiffStatus = "MODIFIED"
	FileDiffStatusDeleted   FileDiffStatus = "DELETED"
	FileDiffStatusRenamed   FileDiffStatus = "RENAMED"
)

func (c *Client) Diff(ctx context.Context, params *DiffParams) (<-chan *FileDiff, <-chan error) {
	ch := make(chan *FileDiff)
	// needs to be buffered so it is not blocking on receiver side when all data is sent
	cherr := make(chan error, 1)

	go func() {
		defer close(ch)
		defer close(cherr)

		if err := params.Validate(); err != nil {
			cherr <- err
			return
		}

		stream, err := c.diffService.Diff(ctx, &rpc.DiffRequest{
			Base:         mapToRPCReadRequest(params.ReadParams),
			BaseRef:      params.BaseRef,
			HeadRef:      params.HeadRef,
			MergeBase:    params.MergeBase,
			IncludePatch: params.IncludePatch,
		})
		if err != nil {
			return
		}

		for {
			resp, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				cherr <- processRPCErrorf(err, "failed to get git diff file from stream")
				return
			}

			ch <- &FileDiff{
				SHA:         resp.Sha,
				OldSHA:      resp.OldSha,
				Path:        resp.Path,
				OldPath:     resp.OldPath,
				Status:      mapRPCFileDiffStatus(resp.Status),
				Additions:   int64(resp.Additions),
				Deletions:   int64(resp.Deletions),
				Changes:     int64(resp.Changes),
				Patch:       resp.Patch,
				IsBinary:    resp.IsBinary,
				IsSubmodule: resp.IsSubmodule,
			}
		}
	}()

	return ch, cherr
}

func mapRPCFileDiffStatus(status rpc.DiffResponse_FileStatus) FileDiffStatus {
	switch status {
	case rpc.DiffResponse_ADDED:
		return FileDiffStatusAdded
	case rpc.DiffResponse_DELETED:
		return FileDiffStatusDeleted
	case rpc.DiffResponse_MODIFIED:
		return FileDiffStatusModified
	case rpc.DiffResponse_RENAMED:
		return FileDiffStatusRenamed
	case rpc.DiffResponse_UNDEFINED:
		return FileDiffStatusUndefined
	default:
		return FileDiffStatusUndefined
	}
}
