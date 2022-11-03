// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package gitea

import (
	"bytes"
	"context"
	"fmt"

	gitea "code.gitea.io/gitea/modules/git"
	"github.com/harness/gitness/gitrpc/internal/types"
)

const (
	giteaPrettyLogFormat = `--pretty=format:%H`
)

// GetLatestCommit gets the latest commit of a path relative from the provided reference.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetLatestCommit(ctx context.Context, repoPath string,
	ref string, treePath string) (*types.Commit, error) {
	treePath = cleanTreePath(treePath)

	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	giteaCommit, err := giteaGetCommitByPath(giteaRepo, ref, treePath)
	if err != nil {
		return nil, fmt.Errorf("error getting latest commit for '%s': %w", treePath, err)
	}

	return mapGiteaCommit(giteaCommit)
}

// giteaGetCommitByPath is a copy of gitea code - required as we want latest commit per specific branch.
func giteaGetCommitByPath(giteaRepo *gitea.Repository, ref string, treePath string) (*gitea.Commit, error) {
	if treePath == "" {
		treePath = "."
	}

	// NOTE: the difference to gitea implementation is passing `ref`.
	stdout, _, runErr := gitea.NewCommand(giteaRepo.Ctx, "log", ref, "-1", giteaPrettyLogFormat, "--", treePath).
		RunStdBytes(&gitea.RunOpts{Dir: giteaRepo.Path})
	if runErr != nil {
		return nil, runErr
	}

	giteaCommits, err := giteaParsePrettyFormatLogToList(giteaRepo, stdout)
	if err != nil {
		return nil, err
	}

	return giteaCommits[0], nil
}

// giteaParsePrettyFormatLogToList is an exact copy of gitea code.
func giteaParsePrettyFormatLogToList(giteaRepo *gitea.Repository, logs []byte) ([]*gitea.Commit, error) {
	var giteaCommits []*gitea.Commit
	if len(logs) == 0 {
		return giteaCommits, nil
	}

	parts := bytes.Split(logs, []byte{'\n'})

	for _, commitID := range parts {
		commit, err := giteaRepo.GetCommit(string(commitID))
		if err != nil {
			return nil, err
		}
		giteaCommits = append(giteaCommits, commit)
	}

	return giteaCommits, nil
}

// ListCommits lists the commits reachable from ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) ListCommits(ctx context.Context, repoPath string,
	ref string, page int, pageSize int) ([]types.Commit, int64, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, 0, err
	}
	defer giteaRepo.Close()

	// Get the giteaTopCommit object for the ref
	giteaTopCommit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting commit for ref '%s': %w", ref, err)
	}

	giteaCommits, err := giteaTopCommit.CommitsByRange(page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting commits: %w", err)
	}

	totalCount, err := giteaTopCommit.CommitsCount()
	if err != nil {
		return nil, 0, fmt.Errorf("error getting total commit count: %w", err)
	}

	commits := make([]types.Commit, len(giteaCommits))
	for i := range giteaCommits {
		var commit *types.Commit
		commit, err = mapGiteaCommit(giteaCommits[i])
		if err != nil {
			return nil, 0, err
		}
		commits[i] = *commit
	}

	// TODO: save to cast to int from int64, or we expect exceeding int.MaxValue?
	return commits, totalCount, nil
}

// GetCommit returns the (latest) commit for a specific ref.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetCommit(ctx context.Context, repoPath string, ref string) (*types.Commit, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	commit, err := giteaRepo.GetCommit(ref)
	if err != nil {
		return nil, err
	}

	return mapGiteaCommit(commit)
}

// GetCommits returns the (latest) commits for a specific list of refs.
// Note: ref can be Branch / Tag / CommitSHA.
func (g Adapter) GetCommits(ctx context.Context, repoPath string, refs []string) ([]types.Commit, error) {
	giteaRepo, err := gitea.OpenRepository(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	defer giteaRepo.Close()

	commits := make([]types.Commit, len(refs))
	for i, sha := range refs {
		var giteaCommit *gitea.Commit
		giteaCommit, err = giteaRepo.GetCommit(sha)
		if err != nil {
			return nil, err
		}

		var commit *types.Commit
		commit, err = mapGiteaCommit(giteaCommit)
		if err != nil {
			return nil, err
		}
		commits[i] = *commit
	}

	return commits, nil
}
