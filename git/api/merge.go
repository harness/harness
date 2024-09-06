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
	"bytes"
	"context"

	"github.com/harness/gitness/git/command"
	"github.com/harness/gitness/git/sha"
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
) (sha.SHA, string, error) {
	if repoPath == "" {
		return sha.None, "", ErrRepositoryPathEmpty
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

	cmd := command.New("merge-base",
		command.WithArg(base, head),
	)

	stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
	err := cmd.Run(ctx,
		command.WithDir(repoPath),
		command.WithStdout(stdout),
		command.WithStderr(stderr),
	)
	if err != nil {
		// git merge-base rev1 rev2
		// if there is unrelated history then stderr is empty with
		// exit code 1. This cannot be handled in processGitErrorf because stderr is empty.
		if command.AsError(err).IsExitCode(1) && stderr.Len() == 0 {
			return sha.None, "", &UnrelatedHistoriesError{
				BaseRef: base,
				HeadRef: head,
			}
		}
		return sha.None, "", processGitErrorf(err, "failed to get merge-base [%s, %s]", base, head)
	}

	result, err := sha.New(stdout.String())
	if err != nil {
		return sha.None, "", err
	}
	return result, base, nil
}

// IsAncestor returns if the provided commit SHA is ancestor of the other commit SHA.
func (g *Git) IsAncestor(
	ctx context.Context,
	repoPath string,
	alternates []string,
	ancestorCommitSHA,
	descendantCommitSHA sha.SHA,
) (bool, error) {
	if repoPath == "" {
		return false, ErrRepositoryPathEmpty
	}

	cmd := command.New("merge-base",
		command.WithFlag("--is-ancestor"),
		command.WithArg(ancestorCommitSHA.String(), descendantCommitSHA.String()),
		command.WithAlternateObjectDirs(alternates...),
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
