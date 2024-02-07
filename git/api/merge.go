package api

import (
	"bytes"
	"context"
	"github.com/harness/gitness/git/command"
	"strings"
)

const (
	// RemotePrefix is the base directory of the remotes information of git.
	RemotePrefix = "refs/remotes/"
)

// GetMergeBase checks and returns merge base of two branches and the reference used as base.
func (g *Git) GetMergeBase(
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
		tmpBaseName := RemotePrefix + remote + "/tmp_" + base
		// Fetch commit into a temporary branch in order to be able to handle commits and tags
		cmd := command.New("fetch",
			command.WithFlag("--no-tags"),
			command.WithArg(remote),
			command.WithPostSepArg(base+":"+tmpBaseName),
		)
		err := cmd.Run(ctx, command.WithDir(repoPath))
		if err == nil {
			base = tmpBaseName
		}
	}

	stdout := &bytes.Buffer{}
	cmd := command.New("merge-base",
		command.WithPostSepArg(base, head),
	)
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(stdout),
	)
	if err != nil {
		return "", "", processGitErrorf(err, "failed to get merge-base [%s, %s]", base, head)
	}

	return strings.TrimSpace(stdout.String()), base, nil
}

// IsAncestor returns if the provided commit SHA is ancestor of the other commit SHA.
func (g *Git) IsAncestor(
	ctx context.Context,
	repoPath string,
	ancestorCommitSHA, descendantCommitSHA string,
) (bool, error) {
	if repoPath == "" {
		return false, ErrRepositoryPathEmpty
	}

	cmd := command.New("merge-base",
		command.WithFlag("--is-ancestor"),
		command.WithArg(ancestorCommitSHA, descendantCommitSHA),
	)

	err := cmd.Run(ctx, command.WithDir(repoPath))
	if err != nil {
		cmdErr := command.AsError(err)
		if cmdErr != nil && cmdErr.IsExitCode(1) && len(cmdErr.StdErr) == 0 {
			return false, nil
		}
		return false, processGitErrorf(err, "failed to check commit ancestry [%s, %s]",
			ancestorCommitSHA, descendantCommitSHA)
	}

	return true, nil
}
