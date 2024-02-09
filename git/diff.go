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
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"sync"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/diff"
	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/types"

	"golang.org/x/sync/errgroup"
)

// Parse "1 file changed, 3 insertions(+), 3 deletions(-)" for nums.
var shortStatsRegexp = regexp.MustCompile(
	`files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

type CommitShortStatParams struct {
	Path string
	Ref  string
}

func (p CommitShortStatParams) Validate() error {
	if p.Path == "" {
		return errors.InvalidArgument("path cannot be empty")
	}

	if p.Ref == "" {
		return errors.InvalidArgument("ref cannot be empty")
	}
	return nil
}

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
		return errors.InvalidArgument("head ref cannot be empty")
	}
	return nil
}

func (s *Service) RawDiff(
	ctx context.Context,
	out io.Writer,
	params *DiffParams,
	files ...types.FileDiffRequest,
) error {
	return s.rawDiff(ctx, out, params, files...)
}

func (s *Service) rawDiff(ctx context.Context, w io.Writer, params *DiffParams, files ...types.FileDiffRequest) error {
	if err := params.Validate(); err != nil {
		return err
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	err := s.adapter.RawDiff(ctx, w, repoPath, params.BaseRef, params.HeadRef, params.MergeBase, files...)
	if err != nil {
		return err
	}
	return nil
}

func (s *Service) CommitDiff(ctx context.Context, params *GetCommitParams, out io.Writer) error {
	if !isValidGitSHA(params.SHA) {
		return errors.InvalidArgument("the provided commit sha '%s' is of invalid format.", params.SHA)
	}
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	err := s.adapter.CommitDiff(ctx, repoPath, params.SHA, out)
	if err != nil {
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
func (s *Service) DiffShortStat(ctx context.Context, params *DiffParams) (DiffShortStatOutput, error) {
	if err := params.Validate(); err != nil {
		return DiffShortStatOutput{}, err
	}
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	stat, err := s.adapter.DiffShortStat(ctx,
		repoPath,
		params.BaseRef,
		params.HeadRef,
		params.MergeBase,
	)
	if err != nil {
		return DiffShortStatOutput{}, err
	}
	return DiffShortStatOutput{
		Files:     stat.Files,
		Additions: stat.Additions,
		Deletions: stat.Deletions,
	}, nil
}

type DiffStatsOutput struct {
	Commits      int
	FilesChanged int
}

func (s *Service) DiffStats(ctx context.Context, params *DiffParams) (DiffStatsOutput, error) {
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

		rpcOutput, err := s.GetCommitDivergences(groupCtx, options)
		if err != nil {
			return err
		}
		if len(rpcOutput.Divergences) > 0 {
			totalCommits = int(rpcOutput.Divergences[0].Ahead)
		}
		return nil
	})

	errGroup.Go(func() error {
		// read short stat
		stat, err := s.DiffShortStat(groupCtx, &DiffParams{
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

func parseCommitShortStat(statBuffer *bytes.Buffer) (CommitShortStatOutput, error) {
	matches := shortStatsRegexp.FindStringSubmatch(statBuffer.String())
	if len(matches) != 3 {
		return CommitShortStatOutput{}, errors.Internal(errors.New("failed to match stats line"), "")
	}

	var stat CommitShortStatOutput

	// if there are insertions; no insertions case: "1 file changed, 3 deletions(-)"
	if len(matches[1]) > 0 {
		if value, err := strconv.Atoi(matches[1]); err == nil {
			stat.Additions = value
		} else {
			return CommitShortStatOutput{}, fmt.Errorf("failed to parse additions stats: %w", err)
		}
	}

	// if there are deletions; no deletions case: "1 file changed, 3 insertions(+)"
	if len(matches[2]) > 0 {
		if value, err := strconv.Atoi(matches[2]); err == nil {
			stat.Deletions = value
		} else {
			return CommitShortStatOutput{}, fmt.Errorf("failed to parse deletions stats: %w", err)
		}
	}

	return stat, nil
}

type CommitShortStatOutput struct {
	Additions int
	Deletions int
}

func (s *Service) CommitShortStat(
	ctx context.Context,
	params *CommitShortStatParams,
) (CommitShortStatOutput, error) {
	if err := params.Validate(); err != nil {
		return CommitShortStatOutput{}, err
	}
	// git log -1 --shortstat --pretty=format:""
	cmd := command.New(
		"log",
		command.WithFlag("-1"),
		command.WithFlag("--shortstat"),
		command.WithFlag(`--pretty=format:""`),
		command.WithArg(params.Ref),
	)

	stdout := bytes.NewBuffer(nil)
	if err := cmd.Run(ctx, command.WithDir(params.Path), command.WithStdout(stdout)); err != nil {
		return CommitShortStatOutput{}, errors.Internal(err, "failed to show stats")
	}

	stat, err := parseCommitShortStat(stdout)
	if err != nil {
		return CommitShortStatOutput{}, errors.Internal(err, "failed to parse stats line")
	}

	return stat, nil
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

type DiffFileHunkHeaders struct {
	FileHeader  DiffFileHeader
	HunkHeaders []HunkHeader
}

type GetDiffHunkHeadersOutput struct {
	Files []DiffFileHunkHeaders
}

func (s *Service) GetDiffHunkHeaders(
	ctx context.Context,
	params GetDiffHunkHeadersParams,
) (GetDiffHunkHeadersOutput, error) {
	if params.SourceCommitSHA == params.TargetCommitSHA {
		return GetDiffHunkHeadersOutput{}, nil
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	hunkHeaders, err := s.adapter.GetDiffHunkHeaders(ctx, repoPath, params.TargetCommitSHA, params.SourceCommitSHA)
	if err != nil {
		return GetDiffHunkHeadersOutput{}, err
	}

	files := make([]DiffFileHunkHeaders, len(hunkHeaders))
	for i, file := range hunkHeaders {
		headers := make([]HunkHeader, len(file.HunksHeaders))
		for j := range file.HunksHeaders {
			headers[j] = mapHunkHeader(&file.HunksHeaders[j])
		}
		files[i] = DiffFileHunkHeaders{
			FileHeader:  mapDiffFileHeader(&hunkHeaders[i].FileHeader),
			HunkHeaders: headers,
		}
	}

	return GetDiffHunkHeadersOutput{
		Files: files,
	}, nil
}

type DiffCutOutput struct {
	Header       HunkHeader
	LinesHeader  string
	Lines        []string
	MergeBaseSHA string
}

type DiffCutParams struct {
	ReadParams
	SourceCommitSHA string
	TargetCommitSHA string
	Path            string
	LineStart       int
	LineStartNew    bool
	LineEnd         int
	LineEndNew      bool
}

// DiffCut extracts diff snippet from a git diff hunk.
// The snippet is from the diff between the specified commit SHAs.
func (s *Service) DiffCut(ctx context.Context, params *DiffCutParams) (DiffCutOutput, error) {
	if params.SourceCommitSHA == params.TargetCommitSHA {
		return DiffCutOutput{}, errors.InvalidArgument("source and target SHA cannot be the same")
	}

	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)

	mergeBaseSHA, _, err := s.adapter.GetMergeBase(ctx, repoPath, "", params.TargetCommitSHA, params.SourceCommitSHA)
	if err != nil {
		return DiffCutOutput{}, fmt.Errorf("DiffCut: failed to find merge base: %w", err)
	}

	header, linesHunk, err := s.adapter.DiffCut(ctx,
		repoPath,
		params.TargetCommitSHA,
		params.SourceCommitSHA,
		params.Path,
		types.DiffCutParams{
			LineStart:    params.LineStart,
			LineStartNew: params.LineStartNew,
			LineEnd:      params.LineEnd,
			LineEndNew:   params.LineEndNew,
			BeforeLines:  2,
			AfterLines:   2,
			LineLimit:    40,
		})
	if err != nil {
		return DiffCutOutput{}, fmt.Errorf("DiffCut: failed to get diff hunk: %w", err)
	}

	hunkHeader := HunkHeader{
		OldLine: header.OldLine,
		OldSpan: header.OldSpan,
		NewLine: header.NewLine,
		NewSpan: header.NewSpan,
		Text:    header.Text,
	}

	return DiffCutOutput{
		Header:       hunkHeader,
		LinesHeader:  linesHunk.HunkHeader.String(),
		Lines:        linesHunk.Lines,
		MergeBaseSHA: mergeBaseSHA,
	}, nil
}

type FileDiff struct {
	SHA         string              `json:"sha"`
	OldSHA      string              `json:"old_sha,omitempty"`
	Path        string              `json:"path"`
	OldPath     string              `json:"old_path,omitempty"`
	Status      enum.FileDiffStatus `json:"status"`
	Additions   int64               `json:"additions"`
	Deletions   int64               `json:"deletions"`
	Changes     int64               `json:"changes"`
	Patch       []byte              `json:"patch,omitempty"`
	IsBinary    bool                `json:"is_binary"`
	IsSubmodule bool                `json:"is_submodule"`
}

func parseFileDiffStatus(ftype diff.FileType) enum.FileDiffStatus {
	switch ftype {
	case diff.FileAdd:
		return enum.FileDiffStatusAdded
	case diff.FileDelete:
		return enum.FileDiffStatusDeleted
	case diff.FileChange:
		return enum.FileDiffStatusModified
	case diff.FileRename:
		return enum.FileDiffStatusRenamed
	default:
		return enum.FileDiffStatusUndefined
	}
}

//nolint:gocognit
func (s *Service) Diff(
	ctx context.Context,
	params *DiffParams,
	files ...types.FileDiffRequest,
) (<-chan *FileDiff, <-chan error) {
	wg := sync.WaitGroup{}
	ch := make(chan *FileDiff)
	cherr := make(chan error, 1)

	pr, pw := io.Pipe()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pw.Close()

		if err := params.Validate(); err != nil {
			cherr <- err
			return
		}

		err := s.rawDiff(ctx, pw, params, files...)
		if err != nil {
			cherr <- err
			return
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer pr.Close()

		parser := diff.Parser{
			Reader:       bufio.NewReader(pr),
			IncludePatch: params.IncludePatch,
		}

		err := parser.Parse(func(f *diff.File) error {
			ch <- &FileDiff{
				SHA:         f.SHA,
				OldSHA:      f.OldSHA,
				Path:        f.Path,
				OldPath:     f.OldPath,
				Status:      parseFileDiffStatus(f.Type),
				Additions:   int64(f.NumAdditions()),
				Deletions:   int64(f.NumDeletions()),
				Changes:     int64(f.NumChanges()),
				Patch:       f.Patch.Bytes(),
				IsBinary:    f.IsBinary,
				IsSubmodule: f.IsSubmodule,
			}
			return nil
		})
		if err != nil {
			cherr <- err
			return
		}
	}()

	go func() {
		wg.Wait()
		defer close(ch)
		defer close(cherr)
	}()

	return ch, cherr
}

type DiffFileNamesOutput struct {
	Files []string
}

func (s *Service) DiffFileNames(ctx context.Context, params *DiffParams) (DiffFileNamesOutput, error) {
	if err := params.Validate(); err != nil {
		return DiffFileNamesOutput{}, err
	}
	repoPath := getFullPathForRepo(s.reposRoot, params.RepoUID)
	fileNames, err := s.adapter.DiffFileName(
		ctx,
		repoPath,
		params.BaseRef,
		params.HeadRef,
		params.MergeBase,
	)
	if err != nil {
		return DiffFileNamesOutput{}, fmt.Errorf("failed to get diff file data between '%s' and '%s': %w",
			params.BaseRef, params.HeadRef, err)
	}
	return DiffFileNamesOutput{
		Files: fileNames,
	}, nil
}
