// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

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
	w io.Writer,
	customArgs ...string,
) error {
	args := []string{
		"diff",
		"-M",
	}
	args = append(args, customArgs...)
	args = append(args, baseRef, headRef)

	cmd := git.NewCommand(ctx, args...)
	cmd.SetDescription(fmt.Sprintf("GetDiffRange [repo_path: %s]", repoPath))
	errbuf := bytes.Buffer{}
	if err := cmd.Run(&git.RunOpts{
		Dir:    repoPath,
		Stderr: &errbuf,
		Stdout: w,
	}); err != nil {
		return fmt.Errorf("git diff [%s base:%s head:%s]: %s", repoPath, baseRef, headRef, errbuf.String())
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
