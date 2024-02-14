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

package adapter

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/types"

	"code.gitea.io/gitea/modules/git"
)

// modifyHeader needs to modify diff hunk header with the new start line
// and end line with calculated span.
// if diff hunk header is -100, 50 +100, 50 and startLine = 120, endLine=140
// then we need to modify header to -120,20 +120,20.
// warning: changes are possible and param endLine may not exist in the future.
func modifyHeader(hunk types.HunkHeader, startLine, endLine int) []byte {
	oldStartLine := hunk.OldLine
	newStartLine := hunk.NewLine
	oldSpan := hunk.OldSpan
	newSpan := hunk.NewSpan

	oldEndLine := oldStartLine + oldSpan
	newEndLine := newStartLine + newSpan

	if startLine > 0 {
		if startLine < oldEndLine {
			oldStartLine = startLine
		}

		if startLine < newEndLine {
			newStartLine = startLine
		}
	}

	if endLine > 0 {
		if endLine < oldEndLine {
			oldSpan = endLine - startLine
		} else if oldEndLine > startLine {
			oldSpan = oldEndLine - startLine
		}

		if endLine < newEndLine {
			newSpan = endLine - startLine
		} else if newEndLine > startLine {
			newSpan = newEndLine - startLine
		}
	}

	return []byte(fmt.Sprintf("@@ -%d,%d +%d,%d @@",
		oldStartLine, oldSpan, newStartLine, newSpan))
}

// cutLinesFromFullFileDiff reads from r and writes to w headers and between
// startLine and endLine. if startLine and endLine is equal to 0 then it uses io.Copy
// warning: changes are possible and param endLine may not exist in the future
//
//nolint:gocognit
func cutLinesFromFullFileDiff(w io.Writer, r io.Reader, startLine, endLine int) error {
	if startLine < 0 {
		startLine = 0
	}

	if endLine < 0 {
		endLine = 0
	}

	if startLine == 0 && endLine > 0 {
		startLine = 1
	}

	if endLine < startLine {
		endLine = 0
	}

	// no need for processing lines just copy the data
	if startLine == 0 && endLine == 0 {
		_, err := io.Copy(w, r)
		return err
	}

	linePos := 0
	start := false
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()

		if start {
			linePos++
		}

		if endLine > 0 && linePos > endLine {
			break
		}

		if linePos > 0 &&
			(startLine > 0 && linePos < startLine) {
			continue
		}

		if len(line) >= 2 && bytes.HasPrefix(line, []byte{'@', '@'}) {
			hunk, ok := parser.ParseDiffHunkHeader(string(line)) // TBD: maybe reader?
			if !ok {
				return fmt.Errorf("failed to extract lines from diff, range [%d,%d] : %w",
					startLine, endLine, ErrParseDiffHunkHeader)
			}
			line = modifyHeader(hunk, startLine, endLine)
			start = true
		}

		if _, err := w.Write(line); err != nil {
			return err
		}

		if _, err := w.Write([]byte{'\n'}); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func (a Adapter) RawDiff(
	ctx context.Context,
	w io.Writer,
	repoPath string,
	baseRef string,
	headRef string,
	mergeBase bool,
	files ...types.FileDiffRequest,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	baseTag, err := a.GetAnnotatedTag(ctx, repoPath, baseRef)
	if err == nil {
		baseRef = baseTag.TargetSha
	}

	headTag, err := a.GetAnnotatedTag(ctx, repoPath, headRef)
	if err == nil {
		headRef = headTag.TargetSha
	}

	args := make([]string, 0, 8)
	args = append(args, "diff", "-M", "--full-index")
	if mergeBase {
		args = append(args, "--merge-base")
	}
	perFileDiffRequired := false
	paths := make([]string, 0, len(files))
	if len(files) > 0 {
		for _, file := range files {
			paths = append(paths, file.Path)
			if file.StartLine > 0 || file.EndLine > 0 {
				perFileDiffRequired = true
			}
		}
	}

	processed := 0

again:
	startLine := 0
	endLine := 0
	newargs := make([]string, len(args), len(args)+8)
	copy(newargs, args)

	if len(files) > 0 {
		startLine = files[processed].StartLine
		endLine = files[processed].EndLine
	}

	if perFileDiffRequired {
		if startLine > 0 || endLine > 0 {
			newargs = append(newargs, "-U"+strconv.Itoa(math.MaxInt32))
		}
		paths = []string{files[processed].Path}
	}

	newargs = append(newargs, baseRef, headRef)

	if len(paths) > 0 {
		newargs = append(newargs, "--")
		newargs = append(newargs, paths...)
	}

	pipeRead, pipeWrite := io.Pipe()
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		err = a.rawDiff(ctx, pipeWrite, repoPath, baseRef, headRef, newargs...)
	}()

	if err = cutLinesFromFullFileDiff(w, pipeRead, startLine, endLine); err != nil {
		return err
	}

	if perFileDiffRequired {
		processed++
		if processed < len(files) {
			goto again
		}
	}

	return nil
}

func (a Adapter) rawDiff(
	ctx context.Context,
	w io.Writer,
	repoPath string,
	baseRef string,
	headRef string,
	args ...string,
) error {
	cmd := git.NewCommand(ctx, args...)
	cmd.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
	errbuf := bytes.Buffer{}
	if err := cmd.Run(&git.RunOpts{
		Dir:    repoPath,
		Stderr: &errbuf,
		Stdout: w,
	}); err != nil {
		if errbuf.Len() > 0 {
			err = &runStdError{err: err, stderr: errbuf.String()}
		}
		return processGiteaErrorf(err, "git diff failed between %q and %q", baseRef, headRef)
	}
	return nil
}

// CommitDiff will stream diff for provided ref.
func (a Adapter) CommitDiff(
	ctx context.Context,
	repoPath string,
	sha string,
	w io.Writer,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	if sha == "" {
		return errors.InvalidArgument("commit sha cannot be empty")
	}
	args := make([]string, 0, 8)
	args = append(args, "show", "--full-index", "--pretty=format:%b", sha)

	stderr := new(bytes.Buffer)
	cmd := git.NewCommand(ctx, args...)
	if err := cmd.Run(&git.RunOpts{
		Dir:    repoPath,
		Stdout: w,
		Stderr: stderr,
	}); err != nil {
		return processGiteaErrorf(err, "commit diff error: %v", stderr)
	}
	return nil
}

func (a Adapter) DiffShortStat(
	ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	useMergeBase bool,
) (types.DiffShortStat, error) {
	if repoPath == "" {
		return types.DiffShortStat{}, ErrRepositoryPathEmpty
	}
	separator := ".."
	if useMergeBase {
		separator = "..."
	}

	shortstatArgs := []string{baseRef + separator + headRef}
	if len(baseRef) == 0 || baseRef == git.EmptySHA {
		shortstatArgs = []string{git.EmptyTreeSHA, headRef}
	}
	numFiles, totalAdditions, totalDeletions, err := git.GetDiffShortStat(ctx, repoPath, shortstatArgs...)
	if err != nil {
		return types.DiffShortStat{}, processGiteaErrorf(err, "failed to get diff short stat between %s and %s",
			baseRef, headRef)
	}
	return types.DiffShortStat{
		Files:     numFiles,
		Additions: totalAdditions,
		Deletions: totalDeletions,
	}, nil
}

// GetDiffHunkHeaders for each file in diff output returns file name (old and new to detect renames),
// and all hunk headers. The diffs are generated with unified=0 parameter to create minimum sized hunks.
// Hunks' body is ignored.
// The purpose of this function is to get data based on which code comments could be repositioned.
func (a Adapter) GetDiffHunkHeaders(
	ctx context.Context,
	repoPath string,
	targetRef string,
	sourceRef string,
) ([]*types.DiffFileHunkHeaders, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	pipeRead, pipeWrite := io.Pipe()
	stderr := &bytes.Buffer{}
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := git.NewCommand(ctx,
			"diff", "--patch", "--no-color", "--unified=0", sourceRef, targetRef)
		err = cmd.Run(&git.RunOpts{
			Dir:    repoPath,
			Stdout: pipeWrite,
			Stderr: stderr, // We capture stderr output in a buffer.
		})
	}()

	fileHunkHeaders, err := parser.GetHunkHeaders(pipeRead)

	// First check if there's something in the stderr buffer, if yes that's the error
	if errStderr := parseDiffStderr(stderr); errStderr != nil {
		return nil, errStderr
	}

	// Next check if reading the git diff output caused an error
	if err != nil {
		return nil, err
	}

	return fileHunkHeaders, nil
}

// DiffCut parses full file git diff output and returns lines specified with the parameters.
// The purpose of this function is to get diff data with which code comments could be generated.
//
//nolint:gocognit
func (a Adapter) DiffCut(
	ctx context.Context,
	repoPath string,
	targetRef string,
	sourceRef string,
	path string,
	params types.DiffCutParams,
) (types.HunkHeader, types.Hunk, error) {
	if repoPath == "" {
		return types.HunkHeader{}, types.Hunk{}, ErrRepositoryPathEmpty
	}

	// first fetch the list of the changed files

	pipeRead, pipeWrite := io.Pipe()
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := command.New("diff",
			command.WithFlag("--raw"),
			command.WithFlag("--merge-base"),
			command.WithFlag("-z"),
			command.WithFlag("--find-renames"),
			command.WithArg(targetRef),
			command.WithArg(sourceRef))
		err = cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(pipeWrite))
	}()

	diffEntries, err := parser.DiffRaw(pipeRead)
	if err != nil {
		return types.HunkHeader{}, types.Hunk{}, fmt.Errorf("failed to find the list of changed files: %w", err)
	}

	var (
		oldSHA, newSHA string
		filePath       string
	)

	for _, entry := range diffEntries {
		switch entry.Status {
		case parser.DiffStatusRenamed, parser.DiffStatusCopied:
			if entry.Path != path && entry.OldPath != path {
				continue
			}

			if params.LineStartNew && path == entry.OldPath {
				msg := "for renamed files provide the new file name if commenting the changed lines"
				return types.HunkHeader{}, types.Hunk{}, errors.InvalidArgument(msg)
			}
			if !params.LineStartNew && path == entry.Path {
				msg := "for renamed files provide the old file name if commenting the old lines"
				return types.HunkHeader{}, types.Hunk{}, errors.InvalidArgument(msg)
			}
		default:
			if entry.Path != path {
				continue
			}
		}

		switch entry.Status {
		case parser.DiffStatusRenamed, parser.DiffStatusCopied, parser.DiffStatusModified:
			// For modified and renamed compare file blobs directly.
			oldSHA = entry.OldBlobSHA
			newSHA = entry.NewBlobSHA
		case parser.DiffStatusAdded, parser.DiffStatusDeleted:
			// For added and deleted compare commits, but with the provided path.
			oldSHA = targetRef
			newSHA = sourceRef
			filePath = entry.Path
		}

		break
	}

	if newSHA == "" {
		return types.HunkHeader{}, types.Hunk{}, errors.NotFound("file %s not found in the diff", path)
	}

	// next pull the diff cut for the requested file

	pipeRead, pipeWrite = io.Pipe()
	stderr := bytes.NewBuffer(nil)
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := command.New("diff",
			command.WithFlag("--patch"),
			command.WithFlag("--no-color"),
			command.WithFlag("--unified=100000000"),
			command.WithArg(oldSHA),
			command.WithArg(newSHA))
		if filePath != "" {
			cmd.Add(
				command.WithFlag("--merge-base"),
				command.WithPostSepArg(filePath))
		}

		err = cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite),
			command.WithStderr(stderr))
	}()

	diffCutHeader, linesHunk, err := parser.DiffCut(pipeRead, params)
	if errStderr := parseDiffStderr(stderr); errStderr != nil {
		// First check if there's something in the stderr buffer, if yes that's the error
		return types.HunkHeader{}, types.Hunk{}, errStderr
	}
	if err != nil {
		// Next check if reading the git diff output caused an error
		return types.HunkHeader{}, types.Hunk{}, err
	}

	return diffCutHeader, linesHunk, nil
}

func (a Adapter) DiffFileName(ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	mergeBase bool,
) ([]string, error) {
	args := make([]string, 0, 8)
	args = append(args, "diff", "--name-only")
	if mergeBase {
		args = append(args, "--merge-base")
	}
	args = append(args, baseRef, headRef)
	cmd := git.NewCommand(ctx, args...)
	stdout, _, runErr := cmd.RunStdBytes(&git.RunOpts{Dir: repoPath})
	if runErr != nil {
		return nil, processGiteaErrorf(runErr, "failed to trigger diff command")
	}

	return parseLinesToSlice(stdout), nil
}

func parseDiffStderr(stderr *bytes.Buffer) error {
	errRaw := stderr.String() // assume there will never be a lot of output to stdout
	if len(errRaw) == 0 {
		return nil
	}

	if idx := strings.IndexByte(errRaw, '\n'); idx > 0 {
		errRaw = errRaw[:idx] // get only the first line of the output
	}

	errRaw = strings.TrimPrefix(errRaw, "fatal: ") // git errors start with the "fatal: " prefix

	if strings.Contains(errRaw, "bad revision") {
		return types.ErrSHADoesNotMatch
	}

	return errors.New(errRaw)
}
