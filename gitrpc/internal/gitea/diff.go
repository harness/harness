// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bytes"
	"context"
	"fmt"
	"io"

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
	separator := "..."
	if !useMergeBase {
		separator = ".."
	}

	shortstatArgs := []string{baseRef + separator + headRef}
	if len(baseRef) == 0 || baseRef == git.EmptySHA {
		shortstatArgs = []string{git.EmptyTreeSHA, headRef}
	}
	numFiles, totalAdditions, totalDeletions, err := git.GetDiffShortStat(ctx, repoPath, shortstatArgs...)
	if err != nil {
		return types.DiffShortStat{}, fmt.Errorf("failed to get diff short stat between %s and %s with err: %w",
			baseRef, headRef, err)
	}
	return types.DiffShortStat{
		Files:     numFiles,
		Additions: totalAdditions,
		Deletions: totalDeletions,
	}, nil
}
