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

package gitea

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/parser"
	"github.com/harness/gitness/gitrpc/internal/types"

	"code.gitea.io/gitea/modules/git"
)

func (g Adapter) RawDiff(
	ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	mergeBase bool,
	w io.Writer,
) error {
	args := make([]string, 0, 8)
	args = append(args, "diff", "-M", "--full-index")
	if mergeBase {
		args = append(args, "--merge-base")
	}
	args = append(args, baseRef, headRef)

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
		return processGiteaErrorf(err, "git diff failed between '%s' and '%s' with err: %v", baseRef, headRef, err)
	}
	return nil
}

// CommitDiff will stream diff for provided ref.
func (g Adapter) CommitDiff(ctx context.Context, repoPath, sha string, w io.Writer) error {
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

func (g Adapter) DiffShortStat(
	ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
	useMergeBase bool,
) (types.DiffShortStat, error) {
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
func (g Adapter) GetDiffHunkHeaders(
	ctx context.Context,
	repoPath, targetRef, sourceRef string,
) ([]*types.DiffFileHunkHeaders, error) {
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
func (g Adapter) DiffCut(
	ctx context.Context,
	repoPath, targetRef, sourceRef, path string,
	params types.DiffCutParams,
) (types.HunkHeader, types.Hunk, error) {
	pipeRead, pipeWrite := io.Pipe()
	stderr := &bytes.Buffer{}
	go func() {
		var err error

		defer func() {
			// If running of the command below fails, make the pipe reader also fail with the same error.
			_ = pipeWrite.CloseWithError(err)
		}()

		cmd := git.NewCommand(ctx,
			"diff", "--merge-base", "--patch", "--no-color", "--unified=100000000",
			targetRef, sourceRef, "--", path)
		err = cmd.Run(&git.RunOpts{
			Dir:    repoPath,
			Stdout: pipeWrite,
			Stderr: stderr, // We capture stderr output in a buffer.
		})
	}()

	diffCutHeader, linesHunk, err := parser.DiffCut(pipeRead, params)

	// First check if there's something in the stderr buffer, if yes that's the error
	if errStderr := parseDiffStderr(stderr); errStderr != nil {
		return types.HunkHeader{}, types.Hunk{}, errStderr
	}

	// Next check if reading the git diff output caused an error
	if err != nil {
		return types.HunkHeader{}, types.Hunk{}, err
	}

	return diffCutHeader, linesHunk, nil
}

func (g Adapter) DiffFileStat(
	ctx context.Context,
	repoPath string,
	baseRef string,
	headRef string,
) ([]string, error) {
	cmd := git.NewCommand(ctx,
		"diff", "--name-only", headRef, baseRef)
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
