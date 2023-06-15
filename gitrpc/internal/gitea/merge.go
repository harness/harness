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
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/gitrpc/enum"
	"github.com/harness/gitness/gitrpc/internal/tempdir"
	"github.com/harness/gitness/gitrpc/internal/types"

	"code.gitea.io/gitea/modules/git"
	"github.com/rs/zerolog/log"
)

// CreateTemporaryRepo creates a temporary repo with "base" for pr.BaseBranch and "tracking" for  pr.HeadBranch
// it also create a second base branch called "original_base".
//
//nolint:funlen,gocognit // need refactor
func (g Adapter) CreateTemporaryRepoForPR(
	ctx context.Context,
	reposTempPath string,
	pr *types.PullRequest,
	baseBranch string,
	trackingBranch string,
) (types.TempRepository, error) {
	if pr.BaseRepoPath == "" && pr.HeadRepoPath != "" {
		pr.BaseRepoPath = pr.HeadRepoPath
	}

	if pr.HeadRepoPath == "" && pr.BaseRepoPath != "" {
		pr.HeadRepoPath = pr.BaseRepoPath
	}

	if pr.BaseBranch == "" {
		return types.TempRepository{}, errors.New("empty base branch")
	}

	if pr.HeadBranch == "" {
		return types.TempRepository{}, errors.New("empty head branch")
	}

	baseRepoPath := pr.BaseRepoPath
	headRepoPath := pr.HeadRepoPath

	// Clone base repo.
	tmpBasePath, err := tempdir.CreateTemporaryPath(reposTempPath, "pull")
	if err != nil {
		return types.TempRepository{}, err
	}

	if err = g.InitRepository(ctx, tmpBasePath, false); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return types.TempRepository{}, err
	}

	remoteRepoName := "head_repo"

	// Add head repo remote.
	addCacheRepo := func(staging, cache string) error {
		var f *os.File
		alternates := filepath.Join(staging, ".git", "objects", "info", "alternates")
		f, err = os.OpenFile(alternates, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("failed to open alternates file '%s': %w", alternates, err)
		}
		defer f.Close()
		data := filepath.Join(cache, "objects")
		if _, err = fmt.Fprintln(f, data); err != nil {
			return fmt.Errorf("failed to write alternates file '%s': %w", alternates, err)
		}
		return nil
	}

	if err = addCacheRepo(tmpBasePath, baseRepoPath); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		return types.TempRepository{},
			fmt.Errorf("unable to add base repository to temporary repo [%s -> tmpBasePath]: %w", pr.BaseRepoPath, err)
	}

	var outbuf, errbuf strings.Builder
	if err = git.NewCommand(ctx, "remote", "add", "-t", pr.BaseBranch, "-m", pr.BaseBranch, "origin", baseRepoPath).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to add base repository as origin "+
			"[%s -> tmpBasePath]:\n%s\n%s", pr.BaseRepoPath, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	// Fetch base branch
	baseCommit, err := g.GetCommit(ctx, pr.BaseRepoPath, pr.BaseBranch)
	if err != nil {
		return types.TempRepository{}, fmt.Errorf("failed to get commit of %s branch: %w", baseBranch, err)
	}
	baseID := baseCommit.SHA
	if err = git.NewCommand(ctx, "fetch", "origin", "--no-tags", "--",
		baseID+":"+baseBranch, baseID+":original_"+baseBranch).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to fetch origin base branch "+
			"[%s:%s -> base, original_base in tmpBasePath].\n%s\n%s",
			pr.BaseRepoPath, pr.BaseBranch, outbuf.String(), errbuf.String())
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
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to set HEAD as base "+
			"branch [tmpBasePath]:\n%s\n%s", outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	if err = addCacheRepo(tmpBasePath, headRepoPath); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to head base repository "+
			"to temporary repo [%s -> tmpBasePath]", pr.HeadRepoPath)
	}

	if err = git.NewCommand(ctx, "remote", "add", remoteRepoName, headRepoPath).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to add head repository as head_repo "+
			"[%s -> tmpBasePath]:\n%s\n%s", pr.HeadRepoPath, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	headCommit, err := g.GetCommit(ctx, pr.HeadRepoPath, pr.HeadBranch)
	if err != nil {
		return types.TempRepository{}, fmt.Errorf("failed to get commit of %s branch: %w", trackingBranch, err)
	}
	headID := headCommit.SHA
	if err = git.NewCommand(ctx, "fetch", "--no-tags", remoteRepoName, headID+":"+trackingBranch).
		Run(&git.RunOpts{
			Dir:    tmpBasePath,
			Stdout: &outbuf,
			Stderr: &errbuf,
		}); err != nil {
		_ = tempdir.RemoveTemporaryPath(tmpBasePath)
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return types.TempRepository{}, processGiteaErrorf(giteaErr, "unable to fetch head_repo head branch "+
			"[%s:%s -> tracking in tmpBasePath]:\n%s\n%s",
			pr.HeadRepoPath, pr.HeadBranch, outbuf.String(), errbuf.String())
	}
	outbuf.Reset()
	errbuf.Reset()

	return types.TempRepository{
		Path:    tmpBasePath,
		BaseSHA: baseID,
		HeadSHA: headID,
	}, nil
}

func runMergeCommand(
	ctx context.Context,
	pr *types.PullRequest,
	mergeMethod enum.MergeMethod,
	cmd *git.Command,
	tmpBasePath string,
	env []string,
) error {
	var outbuf, errbuf strings.Builder
	if err := cmd.Run(&git.RunOpts{
		Dir:    tmpBasePath,
		Stdout: &outbuf,
		Stderr: &errbuf,
		Env:    env,
	}); err != nil {
		// Merge will leave a MERGE_HEAD file in the .git folder if there is a conflict
		if _, statErr := os.Stat(filepath.Join(tmpBasePath, ".git", "MERGE_HEAD")); statErr == nil {
			// We have a merge conflict error
			if err = conflictFiles(ctx, pr, env, tmpBasePath, &outbuf); err != nil {
				return err
			}
			return &types.MergeConflictsError{
				Method: mergeMethod,
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		} else if strings.Contains(errbuf.String(), "refusing to merge unrelated histories") {
			return &types.MergeUnrelatedHistoriesError{
				Method: mergeMethod,
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		}
		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return processGiteaErrorf(giteaErr, "git merge [%s -> %s]\n%s\n%s",
			pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
	}

	return nil
}

func commitAndSignNoAuthor(
	ctx context.Context,
	pr *types.PullRequest,
	message string,
	signArg string,
	tmpBasePath string,
	env []string,
) error {
	var outbuf, errbuf strings.Builder
	if signArg == "" {
		if err := git.NewCommand(ctx, "commit", "-m", message).
			Run(&git.RunOpts{
				Env:    env,
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
		}
	} else {
		if err := git.NewCommand(ctx, "commit", signArg, "-m", message).
			Run(&git.RunOpts{
				Env:    env,
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
		}
	}
	return nil
}

// Merge merges changes between 2 refs (branch, commits or tags).
//
//nolint:gocognit,nestif
func (g Adapter) Merge(
	ctx context.Context,
	pr *types.PullRequest,
	mergeMethod enum.MergeMethod,
	baseBranch string,
	trackingBranch string,
	tmpBasePath string,
	mergeMsg string,
	env []string,
	identity *types.Identity,
) error {
	var (
		outbuf, errbuf strings.Builder
	)

	if mergeMsg == "" {
		mergeMsg = "Merge commit"
	}

	stagingBranch := "staging"
	// TODO: sign merge commit
	signArg := "--no-gpg-sign"

	switch mergeMethod {
	case enum.MergeMethodMerge:
		cmd := git.NewCommand(ctx, "merge", "--no-ff", "--no-commit", trackingBranch)
		if err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env); err != nil {
			return fmt.Errorf("unable to merge tracking into base: %w", err)
		}

		if err := commitAndSignNoAuthor(ctx, pr, mergeMsg, signArg, tmpBasePath, env); err != nil {
			return fmt.Errorf("unable to make final commit: %w", err)
		}
	case enum.MergeMethodSquash:
		// Merge with squash
		cmd := git.NewCommand(ctx, "merge", "--squash", trackingBranch)
		if err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env); err != nil {
			return fmt.Errorf("unable to merge --squash tracking into base: %w", err)
		}

		if signArg == "" {
			if err := git.NewCommand(ctx, "commit", fmt.Sprintf("--author='%s'", identity.String()), "-m", mergeMsg).
				Run(&git.RunOpts{
					Env:    env,
					Dir:    tmpBasePath,
					Stdout: &outbuf,
					Stderr: &errbuf,
				}); err != nil {
				return processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
			}
		} else {
			if err := git.NewCommand(ctx, "commit", signArg, fmt.Sprintf("--author='%s'", identity.String()), "-m", mergeMsg).
				Run(&git.RunOpts{
					Env:    env,
					Dir:    tmpBasePath,
					Stdout: &outbuf,
					Stderr: &errbuf,
				}); err != nil {
				return processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
			}
		}
	case enum.MergeMethodRebase:
		// Checkout head branch
		if err := git.NewCommand(ctx, "checkout", "-b", stagingBranch, trackingBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return fmt.Errorf(
				"git checkout base prior to merge post staging rebase  [%s -> %s]: %w\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
			)
		}
		outbuf.Reset()
		errbuf.Reset()

		// Rebase before merging
		if err := git.NewCommand(ctx, "rebase", baseBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			// Rebase will leave a REBASE_HEAD file in .git if there is a conflict
			if _, statErr := os.Stat(filepath.Join(tmpBasePath, ".git", "REBASE_HEAD")); statErr == nil {
				var commitSha string

				// TBD git version we will support
				// failingCommitPath := filepath.Join(tmpBasePath, ".git", "rebase-apply", "original-commit") // Git < 2.26
				// if _, cpErr := os.Stat(failingCommitPath); statErr != nil {
				// 	return fmt.Errorf("git rebase staging on to base [%s -> %s]: %v\n%s\n%s",
				// 	pr.HeadBranch, pr.BaseBranch, cpErr, outbuf.String(), errbuf.String())
				// }

				failingCommitPath := filepath.Join(tmpBasePath, ".git", "rebase-merge", "stopped-sha") // Git >= 2.26
				if _, cpErr := os.Stat(failingCommitPath); cpErr != nil {
					return fmt.Errorf(
						"git rebase staging on to base [%s -> %s]: %w\n%s\n%s",
						pr.HeadBranch, pr.BaseBranch, cpErr, outbuf.String(), errbuf.String(),
					)
				}

				commitShaBytes, readErr := os.ReadFile(failingCommitPath)
				if readErr != nil {
					// Abandon this attempt to handle the error
					return fmt.Errorf(
						"git rebase staging on to base [%s -> %s]: %w\n%s\n%s",
						pr.HeadBranch, pr.BaseBranch, readErr, outbuf.String(), errbuf.String(),
					)
				}
				commitSha = strings.TrimSpace(string(commitShaBytes))

				log.Debug().Msgf("RebaseConflict at %s [%s -> %s]: %v\n%s\n%s",
					commitSha, pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
				)
				return &types.MergeConflictsError{
					Method:    mergeMethod,
					CommitSHA: commitSha,
					StdOut:    outbuf.String(),
					StdErr:    errbuf.String(),
					Err:       err,
				}
			}
			return fmt.Errorf(
				"git rebase staging on to base [%s -> %s]: %w\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
			)
		}
		outbuf.Reset()
		errbuf.Reset()

		// Checkout base branch again
		if err := git.NewCommand(ctx, "checkout", baseBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return fmt.Errorf(
				"git checkout base prior to merge post staging rebase  [%s -> %s]: %w\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
			)
		}
		outbuf.Reset()
		errbuf.Reset()

		cmd := git.NewCommand(ctx, "merge", "--ff-only", stagingBranch)

		// Prepare merge with commit
		if err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env); err != nil {
			return err
		}
	default:
		return fmt.Errorf("wrong merge method provided: %s", mergeMethod)
	}

	return nil
}

func conflictFiles(ctx context.Context,
	pr *types.PullRequest,
	env []string,
	repoPath string,
	buf *strings.Builder,
) error {
	stdout, stderr, cferr := git.NewCommand(
		ctx, "diff", "--name-only", "--diff-filter=U", "--relative",
	).RunStdString(&git.RunOpts{
		Env: env,
		Dir: repoPath,
	})
	if cferr != nil {
		return processGiteaErrorf(cferr, "failed to list conflict files [%s -> %s], stderr: %v, err: %v",
			pr.HeadBranch, pr.BaseBranch, stderr, cferr)
	}
	if len(stdout) > 0 {
		buf.Reset()
		buf.WriteString(stdout)
	}
	return nil
}

func (g Adapter) GetDiffTree(ctx context.Context, repoPath, baseBranch, headBranch string) (string, error) {
	getDiffTreeFromBranch := func(repoPath, baseBranch, headBranch string) (string, error) {
		var outbuf, errbuf strings.Builder
		if err := git.NewCommand(ctx, "diff-tree", "--no-commit-id",
			"--name-only", "-r", "-z", "--root", baseBranch, headBranch, "--").
			Run(&git.RunOpts{
				Dir:    repoPath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
			return "", processGiteaErrorf(giteaErr, "git diff-tree [%s base:%s head:%s]: %s",
				repoPath, baseBranch, headBranch, errbuf.String())
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

// GetMergeBase checks and returns merge base of two branches and the reference used as base.
func (g Adapter) GetMergeBase(ctx context.Context, repoPath, remote, base, head string) (string, string, error) {
	if remote == "" {
		remote = "origin"
	}

	if remote != "origin" {
		tmpBaseName := git.RemotePrefix + remote + "/tmp_" + base
		// Fetch commit into a temporary branch in order to be able to handle commits and tags
		_, _, err := git.NewCommand(ctx, "fetch", "--no-tags", remote, "--",
			base+":"+tmpBaseName).RunStdString(&git.RunOpts{Dir: repoPath})
		if err == nil {
			base = tmpBaseName
		}
	}

	stdout, _, err := git.NewCommand(ctx, "merge-base", "--", base, head).RunStdString(&git.RunOpts{Dir: repoPath})
	if err != nil {
		return "", "", processGiteaErrorf(err, "failed to get merge-base")
	}

	return strings.TrimSpace(stdout), base, nil
}

// giteaRunStdError is an implementation of the RunStdError interface in the gitea codebase.
// It allows us to process gitea errors even when using cmd.Run() instead of cmd.RunStdString() or run.StdBytes().
// TODO: solve this nicer once we have proper gitrpc error handling.
type giteaRunStdError struct {
	err    error
	stderr string
}

func (e *giteaRunStdError) Error() string {
	return fmt.Sprintf("failed with %s, error output: %s", e.err, e.stderr)
}

func (e *giteaRunStdError) Unwrap() error {
	return e.err
}

func (e *giteaRunStdError) Stderr() string {
	return e.stderr
}

func (e *giteaRunStdError) IsExitCode(code int) bool {
	var exitError *exec.ExitError
	if errors.As(e.err, &exitError) {
		return exitError.ExitCode() == code
	}
	return false
}
