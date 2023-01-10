// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/gitrpc/internal/tempdir"
	"github.com/harness/gitness/gitrpc/internal/types"

	"code.gitea.io/gitea/models"
	"code.gitea.io/gitea/modules/git"
)

// CreateTemporaryRepo creates a temporary repo with "base" for pr.BaseBranch and "tracking" for  pr.HeadBranch
// it also create a second base branch called "original_base".
//
//nolint:funlen,gocognit // need refactor
func (g Adapter) CreateTemporaryRepoForPR(
	ctx context.Context,
	reposTempPath string,
	pr *types.PullRequest,
) (string, error) {
	if pr.BaseRepoPath == "" && pr.HeadRepoPath != "" {
		pr.BaseRepoPath = pr.HeadRepoPath
	}

	if pr.HeadRepoPath == "" && pr.BaseRepoPath != "" {
		pr.HeadRepoPath = pr.BaseRepoPath
	}

	if pr.BaseBranch == "" {
		return "", errors.New("empty base branch")
	}

	if pr.HeadBranch == "" {
		return "", errors.New("empty head branch")
	}

	baseRepoPath := pr.BaseRepoPath
	headRepoPath := pr.HeadRepoPath

	// Clone base repo.
	tmpBasePath, err := tempdir.CreateTemporaryPath(reposTempPath, "pull")
	if err != nil {
		return "", err
	}

	if err = g.InitRepository(ctx, tmpBasePath, false); err != nil {
		// log.Error("git init tmpBasePath: %v", err)
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", err
	}

	remoteRepoName := "head_repo"
	baseBranch := "base"

	// Add head repo remote.
	addCacheRepo := func(staging, cache string) error {
		var f *os.File
		alternates := filepath.Join(staging, ".git", "objects", "info", "alternates")
		f, err = os.OpenFile(alternates, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		defer f.Close()
		data := filepath.Join(cache, "objects")
		if _, err = fmt.Fprintln(f, data); err != nil {
			return err
		}
		return nil
	}

	if err = addCacheRepo(tmpBasePath, baseRepoPath); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to add base repository to temporary repo [%s -> tmpBasePath]: %w", pr.BaseRepoPath, err)
	}

	var outbuf, errbuf strings.Builder
	if err = git.NewCommand(ctx, "remote", "add", "-t", pr.BaseBranch, "-m", pr.BaseBranch, "origin", baseRepoPath).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to add base repository as origin "+
			"[%s -> tmpBasePath]: %w\n%s\n%s", pr.BaseRepoPath, err, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	if err = git.NewCommand(ctx, "fetch", "origin", "--no-tags", "--",
		pr.BaseBranch+":"+baseBranch, pr.BaseBranch+":original_"+baseBranch).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to fetch origin base branch "+
			"[%s:%s -> base, original_base in tmpBasePath]: %w\n%s\n%s",
			pr.BaseRepoPath, pr.BaseBranch, err, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	if err = git.NewCommand(ctx, "symbolic-ref", "HEAD", git.BranchPrefix+baseBranch).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to set HEAD as base "+
			"branch [tmpBasePath]: %w\n%s\n%s", err, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	if err = addCacheRepo(tmpBasePath, headRepoPath); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to head base repository "+
			"to temporary repo [%s -> tmpBasePath]: %w", pr.HeadRepoPath, err)
	}

	if err = git.NewCommand(ctx, "remote", "add", remoteRepoName, headRepoPath).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return "", fmt.Errorf("unable to add head repository as head_repo "+
			"[%s -> tmpBasePath]: %w\n%s\n%s", pr.HeadRepoPath, err, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	trackingBranch := "tracking"
	headBranch := git.BranchPrefix + pr.HeadBranch
	if err = git.NewCommand(ctx, "fetch", "--no-tags", remoteRepoName, headBranch+":"+trackingBranch).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		if !git.IsBranchExist(ctx, pr.HeadRepoPath, headBranch) {
			return "", models.ErrBranchDoesNotExist{
				BranchName: headBranch,
			}
		}
		return "", fmt.Errorf("unable to fetch head_repo head branch "+
			"[%s:%s -> tracking in tmpBasePath]: %w\n%s\n%s",
			pr.HeadRepoPath, headBranch, err, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	return tmpBasePath, nil
}

func (g Adapter) Merge(
	ctx context.Context,
	pr *types.PullRequest,
	mergeMethod string,
	trackingBranch string,
	tmpBasePath string,
	env []string,
) error {
	var outbuf, errbuf strings.Builder
	cmd := git.NewCommand(ctx, "merge", "--no-ff", trackingBranch)
	if err := cmd.Run(&git.RunOpts{
		Env:    env,
		Dir:    tmpBasePath,
		Stdout: &outbuf,
		Stderr: &errbuf,
	}); err != nil {
		// Merge will leave a MERGE_HEAD file in the .git folder if there is a conflict
		if _, statErr := os.Stat(filepath.Join(tmpBasePath, ".git", "MERGE_HEAD")); statErr == nil {
			// We have a merge conflict error
			return types.MergeConflictsError{
				Method: mergeMethod,
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		} else if strings.Contains(errbuf.String(), "refusing to merge unrelated histories") {
			return types.MergeUnrelatedHistoriesError{
				Method: mergeMethod,
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		}
		return fmt.Errorf("git merge [%s -> %s]: %w\n%s\n%s",
			pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String())
	}

	return nil
}

func (g Adapter) GetDiffTree(ctx context.Context, repoPath, baseBranch, headBranch string) (string, error) {
	getDiffTreeFromBranch := func(repoPath, baseBranch, headBranch string) (string, error) {
		var outbuf, errbuf strings.Builder
		// Compute the diff-tree for sparse-checkout
		if err := git.NewCommand(ctx, "diff-tree", "--no-commit-id",
			"--name-only", "-r", "-z", "--root", baseBranch, headBranch, "--").
			Run(&git.RunOpts{
				Dir:    repoPath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return "", fmt.Errorf("git diff-tree [%s base:%s head:%s]: %s", repoPath, baseBranch, headBranch, errbuf.String())
		}
		return outbuf.String(), nil
	}

	scanNullTerminatedStrings := func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexByte(data, '\x00'); i >= 0 {
			return i + 1, data[0:i], nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	list, err := getDiffTreeFromBranch(repoPath, baseBranch, headBranch)
	if err != nil {
		return "", err
	}

	// Prefixing '/' for each entry, otherwise all files with the same name in subdirectories would be matched.
	out := bytes.Buffer{}
	scanner := bufio.NewScanner(strings.NewReader(list))
	scanner.Split(scanNullTerminatedStrings)
	for scanner.Scan() {
		filepath := scanner.Text()
		// escape '*', '?', '[', spaces and '!' prefix
		filepath = escapedSymbols.ReplaceAllString(filepath, `\$1`)
		// no necessary to escape the first '#' symbol because the first symbol is '/'
		fmt.Fprintf(&out, "/%s\n", filepath)
	}

	return out.String(), nil
}
