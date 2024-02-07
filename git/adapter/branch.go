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
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/types"
)

// GetBranch gets an existing branch.
func (a Adapter) GetBranch(
	ctx context.Context,
	repoPath string,
	branchName string,
) (*types.Branch, error) {
	if repoPath == "" {
		return nil, ErrRepositoryPathEmpty
	}
	if branchName == "" {
		return nil, ErrBranchNameEmpty
	}

	ref := GetReferenceFromBranchName(branchName)
	commit, err := GetCommit(ctx, repoPath, ref, "")
	if err != nil {
		return nil, fmt.Errorf("failed to find the commit for the branch: %w", err)
	}

	return &types.Branch{
		Name:   branchName,
		SHA:    commit.SHA,
		Commit: commit,
	}, nil
}

// HasBranches returns true iff there's at least one branch in the repo (any branch)
// NOTE: This is different from repo.Empty(),
// as it doesn't care whether the existing branch is the default branch or not.
func (a Adapter) HasBranches(
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
		return false, processGiteaErrorf(err, "failed to trigger rev-list command")
	}

	return strings.TrimSpace(output.String()) == "", nil
}
