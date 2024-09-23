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

	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/hook"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/git/sharedrepo"

	"github.com/rs/zerolog/log"
)

var (
	// errConflict is used to error out of sharedrepo Run method without erroring out of merge in case of conflicts.
	errConflict = errors.New("conflict")
)

// Func represents a merge method function. The concrete merge implementation functions must have this signature.
type Func func(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	author, committer *api.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
) (mergeSHA sha.SHA, conflicts []string, err error)

// Merge merges two the commits (targetSHA and sourceSHA) using the Merge method.
func Merge(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	author, committer *api.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
) (mergeSHA sha.SHA, conflicts []string, err error) {
	return mergeInternal(ctx,
		refUpdater,
		repoPath, tmpDir,
		author, committer,
		message,
		mergeBaseSHA, targetSHA, sourceSHA,
		false)
}

// Squash merges two the commits (targetSHA and sourceSHA) using the Squash method.
func Squash(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	author, committer *api.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
) (mergeSHA sha.SHA, conflicts []string, err error) {
	return mergeInternal(ctx,
		refUpdater,
		repoPath, tmpDir,
		author, committer,
		message,
		mergeBaseSHA, targetSHA, sourceSHA,
		true)
}

// mergeInternal is internal implementation of merge used for Merge and Squash methods.
func mergeInternal(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	author, committer *api.Signature,
	message string,
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
	squash bool,
) (mergeSHA sha.SHA, conflicts []string, err error) {
	err = sharedrepo.Run(ctx, refUpdater, tmpDir, repoPath, func(s *sharedrepo.SharedRepo) error {
		var err error

		var treeSHA sha.SHA

		treeSHA, conflicts, err = s.MergeTree(ctx, mergeBaseSHA, targetSHA, sourceSHA)
		if err != nil {
			return fmt.Errorf("merge tree failed: %w", err)
		}

		if len(conflicts) > 0 {
			return errConflict
		}

		parents := make([]sha.SHA, 0, 2)
		parents = append(parents, targetSHA)
		if !squash {
			parents = append(parents, sourceSHA)
		}

		mergeSHA, err = s.CommitTree(ctx, author, committer, treeSHA, message, false, parents...)
		if err != nil {
			return fmt.Errorf("commit tree failed: %w", err)
		}

		if err := refUpdater.InitNew(ctx, mergeSHA); err != nil {
			return fmt.Errorf("refUpdater.InitNew failed: %w", err)
		}

		return nil
	})
	if err != nil && !errors.Is(err, errConflict) {
		return sha.None, nil, fmt.Errorf("merge method=merge squash=%t: %w", squash, err)
	}

	return mergeSHA, conflicts, nil
}

// Rebase merges two the commits (targetSHA and sourceSHA) using the Rebase method.
//
//nolint:gocognit // refactor if needed.
func Rebase(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	_, committer *api.Signature, // commit author isn't used here - it's copied from every commit
	_ string, // commit message isn't used here
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
) (mergeSHA sha.SHA, conflicts []string, err error) {
	err = sharedrepo.Run(ctx, refUpdater, tmpDir, repoPath, func(s *sharedrepo.SharedRepo) error {
		sourceSHAs, err := s.CommitSHAsForRebase(ctx, mergeBaseSHA, sourceSHA)
		if err != nil {
			return fmt.Errorf("failed to find commit list in rebase merge: %w", err)
		}

		lastCommitSHA := targetSHA
		lastTreeSHA, err := s.GetTreeSHA(ctx, targetSHA.String())
		if err != nil {
			return fmt.Errorf("failed to get tree sha for target: %w", err)
		}

		for _, commitSHA := range sourceSHAs {
			var treeSHA sha.SHA

			commitInfo, err := api.GetCommit(ctx, s.Directory(), commitSHA.String())
			if err != nil {
				return fmt.Errorf("failed to get commit data in rebase merge: %w", err)
			}

			// rebase merge preserves the commit author (and date) and the commit message, but changes the committer.
			author := &commitInfo.Author
			message := commitInfo.Title
			if commitInfo.Message != "" {
				message += "\n\n" + commitInfo.Message
			}

			var mergeTreeMergeBaseSHA sha.SHA
			if len(commitInfo.ParentSHAs) > 0 {
				// use parent of commit as merge base to only apply changes introduced by commit.
				// See example usage of when --merge-base was introduced:
				// https://github.com/git/git/commit/66265a693e8deb3ab86577eb7f69940410044081
				//
				// NOTE: CommitSHAsForRebase only returns non-merge commits.
				mergeTreeMergeBaseSHA = commitInfo.ParentSHAs[0]
			}

			treeSHA, conflicts, err = s.MergeTree(ctx, mergeTreeMergeBaseSHA, lastCommitSHA, commitSHA)
			if err != nil {
				return fmt.Errorf("failed to merge tree in rebase merge: %w", err)
			}
			if len(conflicts) > 0 {
				return errConflict
			}

			// Drop any commit which after being rebased would be empty.
			// There's two cases in which that can happen:
			// 1. Empty commit.
			//    Github is dropping empty commits, so we'll do the same.
			// 2. The changes of the commit already exist on the target branch.
			//    Git's `git rebase` is dropping such commits on default (and so does Github)
			//    https://git-scm.com/docs/git-rebase#Documentation/git-rebase.txt---emptydropkeepask
			if treeSHA.Equal(lastTreeSHA) {
				log.Ctx(ctx).Debug().Msgf("skipping commit %s as it's empty after rebase", commitSHA)
				continue
			}

			lastCommitSHA, err = s.CommitTree(ctx, author, committer, treeSHA, message, false, lastCommitSHA)
			if err != nil {
				return fmt.Errorf("failed to commit tree in rebase merge: %w", err)
			}
			lastTreeSHA = treeSHA
		}

		if err := refUpdater.InitNew(ctx, lastCommitSHA); err != nil {
			return fmt.Errorf("refUpdater.InitNew failed: %w", err)
		}

		mergeSHA = lastCommitSHA

		return nil
	})
	if err != nil && !errors.Is(err, errConflict) {
		return sha.None, nil, fmt.Errorf("merge method=rebase: %w", err)
	}

	return mergeSHA, conflicts, nil
}

// FastForward points the  is internal implementation of merge used for Merge and Squash methods.
func FastForward(
	ctx context.Context,
	refUpdater *hook.RefUpdater,
	repoPath, tmpDir string,
	_, _ *api.Signature, // commit author and committer aren't used here
	_ string, // commit message isn't used here
	mergeBaseSHA, targetSHA, sourceSHA sha.SHA,
) (mergeSHA sha.SHA, conflicts []string, err error) {
	if targetSHA != mergeBaseSHA {
		return sha.None, nil,
			errors.Conflict("Target branch has diverged from the source branch. Fast-forward not possible.")
	}

	err = sharedrepo.Run(ctx, refUpdater, tmpDir, repoPath, func(*sharedrepo.SharedRepo) error {
		return refUpdater.InitNew(ctx, sourceSHA)
	})
	if err != nil {
		return sha.None, nil, fmt.Errorf("merge method=fast-forward: %w", err)
	}

	return sourceSHA, nil, nil
}
