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

package merge

import (
	"context"
	"fmt"

	"github.com/harness/gitness/git/adapter"
	"github.com/harness/gitness/git/sharedrepo"
	"github.com/harness/gitness/git/types"
)

// Func represents a merge method function. The concrete merge implementation functions must have this signature.
type Func func(
	ctx context.Context,
	repoPath, tmpDir string,
	author, committer *types.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA string,
) (mergeSHA string, conflicts []string, err error)

// Merge merges two the commits (targetSHA and sourceSHA) using the Merge method.
func Merge(
	ctx context.Context,
	repoPath, tmpDir string,
	author, committer *types.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA string,
) (mergeSHA string, conflicts []string, err error) {
	return mergeInternal(ctx,
		repoPath, tmpDir,
		author, committer,
		message,
		mergeBaseSHA, targetSHA, sourceSHA,
		false)
}

// Squash merges two the commits (targetSHA and sourceSHA) using the Squash method.
func Squash(
	ctx context.Context,
	repoPath, tmpDir string,
	author, committer *types.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA string,
) (mergeSHA string, conflicts []string, err error) {
	return mergeInternal(ctx,
		repoPath, tmpDir,
		author, committer,
		message,
		mergeBaseSHA, targetSHA, sourceSHA,
		true)
}

// mergeInternal is internal implementation of merge used for Merge and Squash methods.
func mergeInternal(
	ctx context.Context,
	repoPath, tmpDir string,
	author, committer *types.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA string,
	squash bool,
) (mergeSHA string, conflicts []string, err error) {
	err = runInSharedRepo(ctx, tmpDir, repoPath, func(s *sharedrepo.SharedRepo) error {
		var err error

		var treeSHA string

		treeSHA, conflicts, err = s.MergeTree(ctx, mergeBaseSHA, targetSHA, sourceSHA)
		if err != nil {
			return fmt.Errorf("merge tree failed: %w", err)
		}

		if len(conflicts) > 0 {
			return nil
		}

		parents := make([]string, 0, 2)
		parents = append(parents, targetSHA)
		if !squash {
			parents = append(parents, sourceSHA)
		}

		mergeSHA, err = s.CommitTree(ctx, author, committer, treeSHA, message, false, parents...)
		if err != nil {
			return fmt.Errorf("commit tree failed: %w", err)
		}

		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("merge method=merge squash=%t: %w", squash, err)
	}

	return mergeSHA, conflicts, nil
}

// Rebase merges two the commits (targetSHA and sourceSHA) using the Rebase method.
func Rebase(
	ctx context.Context,
	repoPath, tmpDir string,
	_, committer *types.Signature, // commit author isn't used here - it's copied from every commit
	_ string, // commit message isn't used here
	mergeBaseSHA, targetSHA, sourceSHA string,
) (mergeSHA string, conflicts []string, err error) {
	err = runInSharedRepo(ctx, tmpDir, repoPath, func(s *sharedrepo.SharedRepo) error {
		sourceSHAs, err := s.CommitSHAList(ctx, mergeBaseSHA, sourceSHA)
		if err != nil {
			return fmt.Errorf("failed to find commit list in rebase merge: %w", err)
		}

		lastCommitSHA := targetSHA

		for i := len(sourceSHAs) - 1; i >= 0; i-- {
			commitSHA := sourceSHAs[i]

			var treeSHA string
			var commitConflicts []string

			commitInfo, err := adapter.GetCommit(ctx, s.Directory(), commitSHA, "")
			if err != nil {
				return fmt.Errorf("failed to get commit data in rebase merge: %w", err)
			}

			// rebase merge preserves the commit author (and date) and the commit message, but changes the committer.
			author := &commitInfo.Author
			message := commitInfo.Title
			if commitInfo.Message != "" {
				message += "\n\n" + commitInfo.Message
			}

			treeSHA, commitConflicts, err = s.MergeTree(ctx, mergeBaseSHA, lastCommitSHA, commitSHA)
			if err != nil {
				return fmt.Errorf("failed to merge tree in rebase merge: %w", err)
			}

			if len(commitConflicts) > 0 {
				_, _, conflicts, err = FindConflicts(ctx, s.Directory(), targetSHA, sourceSHA)
				if err != nil {
					return fmt.Errorf("failed to find conflicts in rebase merge: %w", err)
				}

				if len(conflicts) == 0 {
					return fmt.Errorf("expected to find conflicts after rebase merge between %s and %s, but couldn't",
						mergeBaseSHA, sourceSHA)
				}

				return nil
			}

			lastCommitSHA, err = s.CommitTree(ctx, author, committer, treeSHA, message, false, lastCommitSHA)
			if err != nil {
				return fmt.Errorf("failed to commit tree in rebase merge: %w", err)
			}
		}

		mergeSHA = lastCommitSHA

		return nil
	})
	if err != nil {
		return "", nil, fmt.Errorf("merge method=rebase: %w", err)
	}

	return mergeSHA, conflicts, nil
}

// runInSharedRepo is helper function used to run the provided function inside a shared repository.
func runInSharedRepo(
	ctx context.Context,
	tmpDir, repoPath string,
	fn func(s *sharedrepo.SharedRepo) error,
) error {
	s, err := sharedrepo.NewSharedRepo(tmpDir, repoPath)
	if err != nil {
		return err
	}

	defer s.Close(ctx)

	err = s.InitAsBare(ctx)
	if err != nil {
		return err
	}

	err = fn(s)
	if err != nil {
		return err
	}

	err = s.MoveObjects(ctx)
	if err != nil {
		return err
	}

	return nil
}
