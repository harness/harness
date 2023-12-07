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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/tempdir"
	"github.com/harness/gitness/git/types"

	"code.gitea.io/gitea/modules/git"
)

// CreateTemporaryRepo creates a temporary repo with "base" for pr.BaseBranch and "tracking" for  pr.HeadBranch
// it also create a second base branch called "original_base".
//
//nolint:funlen,gocognit // need refactor
func (a Adapter) CreateTemporaryRepoForPR(
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

	if err = a.InitRepository(ctx, tmpBasePath, false); err != nil {
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
	baseCommit, err := a.GetCommit(ctx, pr.BaseRepoPath, pr.BaseBranch)
	if err != nil {
		return types.TempRepository{}, fmt.Errorf("failed to get commit of base branch '%s', error: %w", pr.BaseBranch, err)
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

	headCommit, err := a.GetCommit(ctx, pr.HeadRepoPath, pr.HeadBranch)
	if err != nil {
		return types.TempRepository{}, fmt.Errorf("failed to get commit of head branch '%s', error: %w", pr.HeadBranch, err)
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

type runMergeResult struct {
	conflictFiles []string
}

func runMergeCommand(
	ctx context.Context,
	pr *types.PullRequest,
	mergeMethod enum.MergeMethod,
	cmd *git.Command,
	tmpBasePath string,
	env []string,
) (runMergeResult, error) {
	var outbuf, errbuf strings.Builder
	if err := cmd.Run(&git.RunOpts{
		Dir:    tmpBasePath,
		Stdout: &outbuf,
		Stderr: &errbuf,
		Env:    env,
	}); err != nil {
		if strings.Contains(errbuf.String(), "refusing to merge unrelated histories") {
			return runMergeResult{}, &types.MergeUnrelatedHistoriesError{
				Method: mergeMethod,
				StdOut: outbuf.String(),
				StdErr: errbuf.String(),
				Err:    err,
			}
		}

		// Merge will leave a MERGE_HEAD file in the .git folder if there is a conflict
		if _, statErr := os.Stat(filepath.Join(tmpBasePath, ".git", "MERGE_HEAD")); statErr == nil {
			// We have a merge conflict error
			files, cferr := conflictFiles(ctx, pr, env, tmpBasePath)
			if cferr != nil {
				return runMergeResult{}, cferr
			}
			return runMergeResult{
				conflictFiles: files,
			}, nil
		}

		giteaErr := &giteaRunStdError{err: err, stderr: errbuf.String()}
		return runMergeResult{}, processGiteaErrorf(giteaErr, "git merge [%s -> %s]\n%s\n%s",
			pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
	}

	return runMergeResult{}, nil
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
func (a Adapter) Merge(
	ctx context.Context,
	pr *types.PullRequest,
	mergeMethod enum.MergeMethod,
	baseBranch string,
	trackingBranch string,
	tmpBasePath string,
	mergeMsg string,
	identity *types.Identity,
	env ...string,
) (types.MergeResult, error) {
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
		result, err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env)
		if err != nil {
			return types.MergeResult{}, fmt.Errorf("unable to merge tracking into base: %w", err)
		}
		if len(result.conflictFiles) > 0 {
			return types.MergeResult{ConflictFiles: result.conflictFiles}, nil
		}

		if err := commitAndSignNoAuthor(ctx, pr, mergeMsg, signArg, tmpBasePath, env); err != nil {
			return types.MergeResult{}, fmt.Errorf("unable to make final commit: %w", err)
		}
	case enum.MergeMethodSquash:
		// Merge with squash
		cmd := git.NewCommand(ctx, "merge", "--squash", trackingBranch)
		result, err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env)
		if err != nil {
			return types.MergeResult{}, fmt.Errorf("unable to merge --squash tracking into base: %w", err)
		}
		if len(result.conflictFiles) > 0 {
			return types.MergeResult{ConflictFiles: result.conflictFiles}, nil
		}

		if signArg == "" {
			if err := git.NewCommand(ctx, "commit", fmt.Sprintf("--author='%s'", identity.String()), "-m", mergeMsg).
				Run(&git.RunOpts{
					Env:    env,
					Dir:    tmpBasePath,
					Stdout: &outbuf,
					Stderr: &errbuf,
				}); err != nil {
				return types.MergeResult{}, processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
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
				return types.MergeResult{}, processGiteaErrorf(err, "git commit [%s -> %s]\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, outbuf.String(), errbuf.String())
			}
		}
	case enum.MergeMethodRebase:
		// Create staging branch
		if err := git.NewCommand(ctx, "checkout", "-b", stagingBranch, trackingBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return types.MergeResult{}, fmt.Errorf(
				"git checkout base prior to merge post staging rebase  [%s -> %s]: %w\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
			)
		}
		outbuf.Reset()
		errbuf.Reset()

		var conflicts bool

		// Rebase before merging
		if err := git.NewCommand(ctx, "rebase", baseBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			// Rebase will leave a REBASE_HEAD file in .git if there is a conflict
			if _, statErr := os.Stat(filepath.Join(tmpBasePath, ".git", "REBASE_HEAD")); statErr == nil {
				// Rebase works by processing commit by commit. To get the full list of conflict files
				// all commits would have to be applied. It's simpler to revert the rebase and
				// get the list conflict using git merge.
				conflicts = true
			} else {
				return types.MergeResult{}, fmt.Errorf(
					"git rebase staging on to base [%s -> %s]: %w\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
				)
			}
		}
		outbuf.Reset()
		errbuf.Reset()

		if conflicts {
			// Rebase failed because there are conflicts. Abort the rebase.
			if err := git.NewCommand(ctx, "rebase", "--abort").
				Run(&git.RunOpts{
					Dir:    tmpBasePath,
					Stdout: &outbuf,
					Stderr: &errbuf,
				}); err != nil {
				return types.MergeResult{}, fmt.Errorf(
					"git abort rebase [%s -> %s]: %w\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
				)
			}
			outbuf.Reset()
			errbuf.Reset()

			// Go back to the base branch.
			if err := git.NewCommand(ctx, "checkout", baseBranch).
				Run(&git.RunOpts{
					Dir:    tmpBasePath,
					Stdout: &outbuf,
					Stderr: &errbuf,
				}); err != nil {
				return types.MergeResult{}, fmt.Errorf(
					"return to the base branch [%s -> %s]: %w\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
				)
			}
			outbuf.Reset()
			errbuf.Reset()

			// Run the ordinary merge to get the list of conflict files.
			cmd := git.NewCommand(ctx, "merge", "--no-ff", "--no-commit", trackingBranch)
			result, err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env)
			if err != nil {
				return types.MergeResult{}, fmt.Errorf(
					"git abort rebase [%s -> %s]: %w\n%s\n%s",
					pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
				)
			}
			if len(result.conflictFiles) > 0 {
				return types.MergeResult{ConflictFiles: result.conflictFiles}, nil
			}

			return types.MergeResult{}, errors.New("rebase reported conflicts, but merge gave no conflict files")
		}

		// Checkout base branch again
		if err := git.NewCommand(ctx, "checkout", baseBranch).
			Run(&git.RunOpts{
				Dir:    tmpBasePath,
				Stdout: &outbuf,
				Stderr: &errbuf,
			}); err != nil {
			return types.MergeResult{}, fmt.Errorf(
				"git checkout base prior to merge post staging rebase  [%s -> %s]: %w\n%s\n%s",
				pr.HeadBranch, pr.BaseBranch, err, outbuf.String(), errbuf.String(),
			)
		}
		outbuf.Reset()
		errbuf.Reset()

		// Prepare merge with commit
		cmd := git.NewCommand(ctx, "merge", "--ff-only", stagingBranch)
		result, err := runMergeCommand(ctx, pr, mergeMethod, cmd, tmpBasePath, env)
		if err != nil {
			return types.MergeResult{}, fmt.Errorf("unable to ff-olny merge tracking into base: %w", err)
		}
		if len(result.conflictFiles) > 0 {
			return types.MergeResult{ConflictFiles: result.conflictFiles}, nil
		}
	default:
		return types.MergeResult{}, fmt.Errorf("wrong merge method provided: %s", mergeMethod)
	}

	return types.MergeResult{}, nil
}

func conflictFiles(
	ctx context.Context,
	pr *types.PullRequest,
	env []string,
	repoPath string,
) (files []string, err error) {
	stdout, stderr, err := git.NewCommand(
		ctx, "diff", "--name-only", "--diff-filter=U", "--relative",
	).RunStdString(&git.RunOpts{
		Env: env,
		Dir: repoPath,
	})
	if err != nil {
		return nil, processGiteaErrorf(err, "failed to list conflict files [%s -> %s], stderr: %v, err: %v",
			pr.HeadBranch, pr.BaseBranch, stderr, err)
	}
	if len(stdout) > 0 {
		files = strings.Split(stdout[:len(stdout)-1], "\n")
	}
	return files, nil
}

func (a Adapter) GetDiffTree(
	ctx context.Context,
	repoPath string,
	baseBranch string,
	headBranch string,
) (string, error) {
	if repoPath == "" {
		return "", ErrRepositoryPathEmpty
	}
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
func (a Adapter) GetMergeBase(
	ctx context.Context,
	repoPath string,
	remote string,
	base string,
	head string,
) (string, string, error) {
	if repoPath == "" {
		return "", "", ErrRepositoryPathEmpty
	}
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

	stdout, stderr, err := git.NewCommand(ctx, "merge-base", "--", base, head).RunStdString(&git.RunOpts{Dir: repoPath})
	if err != nil {
		return "", "", processGiteaErrorf(err, "failed to get merge-base: %v", stderr)
	}

	return strings.TrimSpace(stdout), base, nil
}

// IsAncestor returns if the provided commit SHA is ancestor of the other commit SHA.
func (a Adapter) IsAncestor(
	ctx context.Context,
	repoPath string,
	ancestorCommitSHA, descendantCommitSHA string,
) (bool, error) {
	if repoPath == "" {
		return false, ErrRepositoryPathEmpty
	}

	_, stderr, runErr := git.NewCommand(ctx, "merge-base", "--is-ancestor", ancestorCommitSHA, descendantCommitSHA).
		RunStdString(&git.RunOpts{Dir: repoPath})
	if runErr != nil {
		if runErr.IsExitCode(1) && stderr == "" {
			return false, nil
		}
		return false, processGiteaErrorf(runErr, "failed to check commit ancestry: %v", stderr)
	}

	return true, nil
}

// giteaRunStdError is an implementation of the RunStdError interface in the gitea codebase.
// It allows us to process gitea errors even when using cmd.Run() instead of cmd.RunStdString() or run.StdBytes().
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
