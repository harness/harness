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

package api

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/parser"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
)

type FileDiffRequest struct {
	Path      string `json:"path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"-"` // warning: changes are possible and this field may not exist in the future
}

type FileDiffRequests []FileDiffRequest

type DiffShortStat struct {
	Files     int
	Additions int
	Deletions int
}

// modifyHeader needs to modify diff hunk header with the new start line
// and end line with calculated span.
// if diff hunk header is -100, 50 +100, 50 and startLine = 120, endLine=140
// then we need to modify header to -120,20 +120,20.
// warning: changes are possible and param endLine may not exist in the future.
func modifyHeader(hunk parser.HunkHeader, startLine, endLine int) []byte {
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

func (g *Git) RawDiff(
	ctx context.Context,
	w io.Writer,
	repoPath string,
	baseRef string,
	headRef string,
	mergeBase bool,
	alternates []string,
	files ...FileDiffRequest,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}

	baseTag, err := g.GetAnnotatedTag(ctx, repoPath, baseRef)
	if err == nil {
		baseRef = baseTag.TargetSha.String()
	}

	headTag, err := g.GetAnnotatedTag(ctx, repoPath, headRef)
	if err == nil {
		headRef = headTag.TargetSha.String()
	}

	cmd := command.New("diff",
		command.WithFlag("-M"),
		command.WithFlag("--full-index"),
		command.WithAlternateObjectDirs(alternates...),
	)
	if mergeBase {
		cmd.Add(command.WithFlag("--merge-base"))
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

	newCmd := cmd.Clone()

	if len(files) > 0 {
		startLine = files[processed].StartLine
		endLine = files[processed].EndLine
	}

	if perFileDiffRequired {
		if startLine > 0 || endLine > 0 {
			newCmd.Add(command.WithFlag("-U" + strconv.Itoa(math.MaxInt32)))
		}
		paths = []string{files[processed].Path}
	}

	newCmd.Add(command.WithArg(baseRef, headRef))

	if len(paths) > 0 {
		newCmd.Add(command.WithPostSepArg(paths...))
	}

	pipeRead, pipeWrite := io.Pipe()
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		if err = newCmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite),
		); err != nil {
			err = processGitErrorf(err, "git diff failed between %q and %q", baseRef, headRef)
		}
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

// CommitDiff will stream diff for provided ref.
func (g *Git) CommitDiff(
	ctx context.Context,
	repoPath string,
	rev string,
	w io.Writer,
) error {
	if repoPath == "" {
		return ErrRepositoryPathEmpty
	}
	if rev == "" {
		return errors.InvalidArgument("git revision cannot be empty")
	}

	cmd := command.New("show",
		command.WithFlag("--full-index"),
		command.WithFlag("--pretty=format:%b"),
		command.WithArg(rev),
	)

	if err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(w),
	); err != nil {
		return processGitErrorf(err, "commit diff error")
	}
	return nil
}

func (g *Git) DiffShortStat(
	ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	useMergeBase bool,
) (DiffShortStat, error) {
	if repoPath == "" {
		return DiffShortStat{}, ErrRepositoryPathEmpty
	}
	separator := ".."
	if useMergeBase {
		separator = "..."
	}

	shortstatArgs := []string{baseRef + separator + headRef}
	if len(baseRef) == 0 || baseRef == types.NilSHA {
		shortstatArgs = []string{sha.EmptyTree.String(), headRef}
	}
	stat, err := GetDiffShortStat(ctx, repoPath, shortstatArgs...)
	if err != nil {
		return DiffShortStat{}, processGitErrorf(err, "failed to get diff short stat between %s and %s",
			baseRef, headRef)
	}
	return stat, nil
}

// GetDiffHunkHeaders for each file in diff output returns file name (old and new to detect renames),
// and all hunk headers. The diffs are generated with unified=0 parameter to create minimum sized hunks.
// Hunks' body is ignored.
// The purpose of this function is to get data based on which code comments could be repositioned.
func (g *Git) GetDiffHunkHeaders(
	ctx context.Context,
	repoPath string,
	targetRef string,
	sourceRef string,
) ([]*parser.DiffFileHunkHeaders, error) {
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

		cmd := command.New("diff",
			command.WithFlag("--patch"),
			command.WithFlag("--full-index"),
			command.WithFlag("--no-color"),
			command.WithFlag("--unified=0"),
			command.WithArg(sourceRef),
			command.WithArg(targetRef),
		)
		err = cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite),
			command.WithStderr(stderr), // We capture stderr output in a buffer.
		)
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
func (g *Git) DiffCut(
	ctx context.Context,
	repoPath string,
	targetRef string,
	sourceRef string,
	path string,
	params parser.DiffCutParams,
) (parser.HunkHeader, parser.Hunk, error) {
	if repoPath == "" {
		return parser.HunkHeader{}, parser.Hunk{}, ErrRepositoryPathEmpty
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
		return parser.HunkHeader{}, parser.Hunk{}, fmt.Errorf("failed to find the list of changed files: %w", err)
	}

	var oldSHA, newSHA string

	for _, entry := range diffEntries {
		if entry.Status == parser.DiffStatusRenamed || entry.Status == parser.DiffStatusCopied {
			// Entries with the status 'R' and 'C' output two paths: the old path and the new path.
			// Using the params.LineStartNew flag to match the path with the entry's old or new path.

			if entry.Path != path && entry.OldPath != path {
				continue
			}

			if params.LineStartNew && path == entry.OldPath {
				msg := "for renamed files provide the new file name if commenting the changed lines"
				return parser.HunkHeader{}, parser.Hunk{}, errors.InvalidArgument(msg)
			}
			if !params.LineStartNew && path == entry.Path {
				msg := "for renamed files provide the old file name if commenting the old lines"
				return parser.HunkHeader{}, parser.Hunk{}, errors.InvalidArgument(msg)
			}
		} else if entry.Path != path {
			// All other statuses output just one path: If the path doesn't match it, proceed with the next entry.
			continue
		}

		rawFileMode := entry.OldFileMode
		if params.LineStartNew {
			rawFileMode = entry.NewFileMode
		}

		fileType, _, err := parseTreeNodeMode(rawFileMode)
		if err != nil {
			return parser.HunkHeader{}, parser.Hunk{},
				fmt.Errorf("failed to parse file mode %s for path %s: %w", rawFileMode, path, err)
		}

		switch fileType {
		default:
			return parser.HunkHeader{}, parser.Hunk{}, errors.Internal(nil, "unrecognized object type")
		case TreeNodeTypeCommit:
			msg := "code comment is not allowed on a submodule"
			return parser.HunkHeader{}, parser.Hunk{}, errors.InvalidArgument(msg)
		case TreeNodeTypeTree:
			msg := "code comment is not allowed on a directory"
			return parser.HunkHeader{}, parser.Hunk{}, errors.InvalidArgument(msg)
		case TreeNodeTypeBlob:
			// a blob is what we want
		}

		var hunkHeader parser.HunkHeader
		var hunk parser.Hunk

		switch entry.Status {
		case parser.DiffStatusRenamed, parser.DiffStatusCopied, parser.DiffStatusModified:
			// For modified and renamed compare file blob SHAs directly.
			oldSHA = entry.OldBlobSHA
			newSHA = entry.NewBlobSHA
			hunkHeader, hunk, err = g.diffCutFromHunk(ctx, repoPath, oldSHA, newSHA, params)
		case parser.DiffStatusAdded, parser.DiffStatusDeleted, parser.DiffStatusType:
			// for added and deleted files read the file content directly
			if params.LineStartNew {
				hunkHeader, hunk, err = g.diffCutFromBlob(ctx, repoPath, true, entry.NewBlobSHA, params)
			} else {
				hunkHeader, hunk, err = g.diffCutFromBlob(ctx, repoPath, false, entry.OldBlobSHA, params)
			}
		default:
			return parser.HunkHeader{}, parser.Hunk{},
				fmt.Errorf("unrecognized git diff file status=%c for path=%s", entry.Status, path)
		}
		if err != nil {
			return parser.HunkHeader{}, parser.Hunk{},
				fmt.Errorf("failed to extract hunk for git diff file status=%c path=%s: %w", entry.Status, path, err)
		}

		// The returned diff hunk will be stored in the DB and will only be used for display, as a reference.
		// Therefore, we trim too long lines to protect the system and the DB.
		const maxLineLen = 200
		parser.LimitLineLen(&hunk.Lines, maxLineLen)

		return hunkHeader, hunk, nil
	}

	return parser.HunkHeader{}, parser.Hunk{}, errors.NotFound("path not found")
}

func (g *Git) diffCutFromHunk(
	ctx context.Context,
	repoPath string,
	oldSHA string,
	newSHA string,
	params parser.DiffCutParams,
) (parser.HunkHeader, parser.Hunk, error) {
	pipeRead, pipeWrite := io.Pipe()
	stderr := bytes.NewBuffer(nil)
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := command.New("diff",
			command.WithFlag("--patch"),
			command.WithFlag("--full-index"),
			command.WithFlag("--no-color"),
			command.WithFlag("--unified=100000000"),
			command.WithArg(oldSHA),
			command.WithArg(newSHA))

		err = cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite),
			command.WithStderr(stderr))
	}()

	diffCutHeader, linesHunk, err := parser.DiffCut(pipeRead, params)
	if errStderr := parseDiffStderr(stderr); errStderr != nil {
		// First check if there's something in the stderr buffer, if yes that's the error
		return parser.HunkHeader{}, parser.Hunk{}, errStderr
	}
	if err != nil {
		// Next check if reading the git diff output caused an error
		return parser.HunkHeader{}, parser.Hunk{}, err
	}

	return diffCutHeader, linesHunk, nil
}

func (g *Git) diffCutFromBlob(
	ctx context.Context,
	repoPath string,
	asAdded bool,
	sha string,
	params parser.DiffCutParams,
) (parser.HunkHeader, parser.Hunk, error) {
	pipeRead, pipeWrite := io.Pipe()
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := command.New("cat-file",
			command.WithFlag("-p"),
			command.WithArg(sha))

		err = cmd.Run(ctx,
			command.WithDir(repoPath),
			command.WithStdout(pipeWrite))
	}()

	cutHeader, cut, err := parser.BlobCut(pipeRead, params)
	if err != nil {
		// Next check if reading the git diff output caused an error
		return parser.HunkHeader{}, parser.Hunk{}, err
	}

	// Convert parser.CutHeader to parser.HunkHeader and parser.Cut to parser.Hunk.

	var hunkHeader parser.HunkHeader
	var hunk parser.Hunk

	if asAdded {
		for i := range cut.Lines {
			cut.Lines[i] = "+" + cut.Lines[i]
		}

		hunkHeader = parser.HunkHeader{
			NewLine: cutHeader.Line,
			NewSpan: cutHeader.Span,
		}

		hunk = parser.Hunk{
			HunkHeader: parser.HunkHeader{NewLine: cut.Line, NewSpan: cut.Span},
			Lines:      cut.Lines,
		}
	} else {
		for i := range cut.Lines {
			cut.Lines[i] = "-" + cut.Lines[i]
		}

		hunkHeader = parser.HunkHeader{
			OldLine: cutHeader.Line,
			OldSpan: cutHeader.Span,
		}

		hunk = parser.Hunk{
			HunkHeader: parser.HunkHeader{OldLine: cut.Line, OldSpan: cut.Span},
			Lines:      cut.Lines,
		}
	}

	return hunkHeader, hunk, nil
}

func (g *Git) DiffFileName(ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	mergeBase bool,
) ([]string, error) {
	cmd := command.New("diff", command.WithFlag("--name-only"))
	if mergeBase {
		cmd.Add(command.WithFlag("--merge-base"))
	}
	cmd.Add(command.WithArg(baseRef, headRef))

	stdout := &bytes.Buffer{}
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(stdout),
	)
	if err != nil {
		return nil, processGitErrorf(err, "failed to trigger diff command")
	}

	return parseLinesToSlice(stdout.Bytes()), nil
}

// GetDiffShortStat counts number of changed files, number of additions and deletions.
func GetDiffShortStat(
	ctx context.Context,
	repoPath string,
	args ...string,
) (DiffShortStat, error) {
	// Now if we call:
	// $ git diff --shortstat 1ebb35b98889ff77299f24d82da426b434b0cca0...788b8b1440462d477f45b0088875
	// we get:
	// " 9902 files changed, 2034198 insertions(+), 298800 deletions(-)\n"

	cmd := command.New("diff",
		command.WithFlag("--shortstat"),
		command.WithArg(args...),
	)

	stdout := &bytes.Buffer{}
	if err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(stdout),
	); err != nil {
		return DiffShortStat{}, err
	}

	return parseDiffStat(stdout.String())
}

var shortStatFormat = regexp.MustCompile(
	`\s*(\d+) files? changed(?:, (\d+) insertions?\(\+\))?(?:, (\d+) deletions?\(-\))?`)

func parseDiffStat(stdout string) (stat DiffShortStat, err error) {
	if len(stdout) == 0 || stdout == "\n" {
		return DiffShortStat{}, nil
	}
	groups := shortStatFormat.FindStringSubmatch(stdout)
	if len(groups) != 4 {
		return DiffShortStat{}, fmt.Errorf("unable to parse shortstat: %s groups: %s", stdout, groups)
	}

	stat.Files, err = strconv.Atoi(groups[1])
	if err != nil {
		return DiffShortStat{}, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumFiles %w",
			stdout, err)
	}

	if len(groups[2]) != 0 {
		stat.Additions, err = strconv.Atoi(groups[2])
		if err != nil {
			return DiffShortStat{}, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumAdditions %w",
				stdout, err)
		}
	}

	if len(groups[3]) != 0 {
		stat.Deletions, err = strconv.Atoi(groups[3])
		if err != nil {
			return DiffShortStat{}, fmt.Errorf("unable to parse shortstat: %s. Error parsing NumDeletions %w",
				stdout, err)
		}
	}
	return stat, nil
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
		return parser.ErrSHADoesNotMatch
	}

	return errors.New(errRaw)
}
