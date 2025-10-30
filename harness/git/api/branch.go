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
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"
)

type Branch struct {
	Name   string
	SHA    sha.SHA
	Commit *Commit
}

type BranchFilter struct {
	Query         string
	Page          int32
	PageSize      int32
	Sort          GitReferenceField
	Order         SortOrder
	IncludeCommit bool
}

// BranchPrefix base dir of the branch information file store on git.
const BranchPrefix = "refs/heads/"

// EnsureBranchPrefix ensures the ref always has "refs/heads/" as prefix.
func EnsureBranchPrefix(ref string) string {
	if !strings.HasPrefix(ref, BranchPrefix) {
		return BranchPrefix + ref
	}
	return ref
}

// GetBranch gets an existing branch.
func (g *Git) GetBranch(
	ctx context.Context,
	repoPath string,
	branchName string,
) (*Branch, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	if branchName == "" {
		return nil, ErrBranchNameEmpty
	}

	ref := GetReferenceFromBranchName(branchName)
	commit, err := g.GetCommitFromRev(ctx, repoPath, ref+"^{commit}") //nolint:goconst
	if err != nil {
		return nil, fmt.Errorf("failed to find the commit for the branch: %w", err)
	}

	return &Branch{
		Name:   branchName,
		SHA:    commit.SHA,
		Commit: commit,
	}, nil
}

// HasBranches returns true iff there's at least one branch in the repo (any branch)
// NOTE: This is different from repo.Empty(),
// as it doesn't care whether the existing branch is the default branch or not.
func (g *Git) HasBranches(
	ctx context.Context,
	repoPath string,
) (bool, error) {
	if repoPath == "" {
		return false, ErrRepositoryPathEmpty
	}
	// repo has branches IFF there's at least one commit that is reachable via a branch
	// (every existing branch points to a commit)
	cmd := command.New("rev-list",
		command.WithFlag("--max-count", "1"),
		command.WithFlag("--branches"),
	)
	output := &bytes.Buffer{}
	if err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(output)); err != nil {
		return false, processGitErrorf(err, "failed to trigger rev-list command")
	}

	return strings.TrimSpace(output.String()) == "", nil
}

func (g *Git) IsBranchExist(ctx context.Context, repoPath, name string) (bool, error) {
	cmd := command.New("show-ref",
		command.WithFlag("--exists", BranchPrefix+name),
	)
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
	)
	if cmdERR := command.AsError(err); cmdERR != nil && cmdERR.IsExitCode(2) {
		// git returns exit code 2 in case the ref doesn't exist.
		// On success it would be 0 and no error would be returned in the first place.
		// Any other exit code we fall through to default error handling.
		// https://git-scm.com/docs/git-show-ref#Documentation/git-show-ref.txt---exists
		return false, nil
	}
	if err != nil {
		return false, processGitErrorf(err, "failed to check if branch %q exist", name)
	}

	return true, nil
}

func (g *Git) GetBranchCount(
	ctx context.Context,
	repoPath string,
) (int, error) {
	if repoPath == "" {
		return 0, ErrRepositoryPathEmpty
	}

	pipeOut, pipeIn := io.Pipe()
	defer pipeOut.Close()

	cmd := command.New("branch",
		command.WithFlag("--list"),
		command.WithFlag("--format=%(refname:short)"),
	)

	go func() {
		err := cmd.Run(ctx, command.WithDir(repoPath), command.WithStdout(pipeIn))
		if err != nil {
			_ = pipeIn.CloseWithError(
				processGitErrorf(err, "failed to trigger branch command"),
			)
			return
		}
		_ = pipeIn.Close()
	}()

	return countLines(pipeOut)
}

func countLines(pipe io.Reader) (int, error) {
	scanner := bufio.NewScanner(pipe)
	count := 0

	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return count, nil
}
