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

package mergequeue

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"

	"github.com/rs/zerolog/log"
)

func (s *Service) fastForward(
	ctx context.Context,
	q *types.MergeQueue,
	entry *types.MergeQueueEntry,
) error {
	repo, err := s.repoFinder.FindByID(ctx, q.RepoID)
	if err != nil {
		return fmt.Errorf("failed to find repo %d: %w", q.RepoID, err)
	}

	unlock, err := s.locker.LockPR(
		ctx,
		repo.ID,
		0,
		merge.Timeout+30*time.Second,
	)
	if err != nil {
		return fmt.Errorf("failed to lock repository for merge queue fast forward: %w", err)
	}
	defer unlock()

	entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
	if err != nil {
		return fmt.Errorf("failed to list merge queue entries: %w", err)
	}

	// Merge queue merges pull requests in a specific order. If checks succeeded for an entry
	// deeper in the queue, we should mark all these PRs as merged.

	mergeEntries, err := s.findEntriesToMerge(entry, entries)
	if err != nil {
		return fmt.Errorf("failed to find merge entries: %w", err)
	}

	expectedTargetBranchSHA := mergeEntries[0].BaseCommitSHA
	newTargetBranchSHA := entry.MergeCommitSHA

	pullReqs := make([]*types.PullReq, len(mergeEntries))
	for i, mergeEntry := range mergeEntries {
		pullReqs[i], err = s.pullreqStore.Find(ctx, mergeEntry.PullReqID)
		if err != nil {
			return fmt.Errorf("failed to find pull request %d: %w", mergeEntry.PullReqID, err)
		}

		pullReq := pullReqs[i]

		// Make sure the pull request source branch commit SHA matches the merge queue entry's head commit SHA.
		// This should be the case because pull requests should be locked when in the merge queue.
		if mergeEntry.HeadCommitSHA.String() != pullReq.SourceSHA {
			return errIncompleteMergeQueue
		}
	}

	systemSession := bootstrap.NewSystemServiceSession()
	systemPrincipalInfo := systemSession.Principal.ToPrincipalInfo()

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, s.urlProvider, systemSession, repo)
	if err != nil {
		return fmt.Errorf("failed to create rpc system references write params: %w", err)
	}

	var updateRefs []git.RefUpdate

	deleteSourceBranchMap := make(map[int64]int64) // map[pullReqID]->seqSourceBranchDeleted

	// Need to update the target branch
	updateRefs = append(updateRefs, git.RefUpdate{
		Name: git.GetBranchRefPath(q.Branch),
		New:  newTargetBranchSHA,
		Old:  expectedTargetBranchSHA,
	})

	err = s.tx.WithTx(ctx, func(ctx context.Context) error {
		// Merge each of the pull requests.
		for i, pr := range pullReqs {
			mergeEntry := mergeEntries[i]

			err = s.mergeQueueEntryStore.Delete(ctx, mergeEntry.PullReqID)
			if err != nil && !errors.Is(err, ErrNotInQueue) {
				return fmt.Errorf("failed to delete merge queue entry for merged PR: %w", err)
			}

			mergeOutput := git.MergeOutput{
				BaseSHA:          mergeEntry.BaseCommitSHA,
				HeadSHA:          mergeEntry.HeadCommitSHA,
				MergeBaseSHA:     mergeEntry.MergeBaseSHA,
				MergeSHA:         mergeEntry.MergeCommitSHA,
				CommitCount:      mergeEntry.CommitCount,
				ChangedFileCount: mergeEntry.ChangedFileCount,
				Additions:        mergeEntry.Additions,
				Deletions:        mergeEntry.Deletions,
				ConflictFiles:    nil,
			}

			var seqSourceBranchDeleted int64

			pullReqs[i], seqSourceBranchDeleted, err = s.mergeService.DatabaseUpdateNoOptLock(
				ctx,
				pr,
				mergeEntry.MergeMethod,
				mergeOutput,
				systemPrincipalInfo,
				false,
				"",
			)
			if err != nil {
				return fmt.Errorf("failed to mark pull request as merged: %w", err)
			}

			if mergeEntries[i].DeleteSourceBranch {
				deleteSourceBranchMap[mergeEntry.PullReqID] = seqSourceBranchDeleted
			}

			prNumber := strconv.FormatInt(pr.Number, 10)

			refPullReqHead, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqHead)
			if err != nil {
				return fmt.Errorf("failed to get pull request head ref path: %w", err)
			}

			refPullReqMerge, err := git.GetRefPath(prNumber, gitenum.RefTypePullReqMerge)
			if err != nil {
				return fmt.Errorf("failed to get pull request merge ref path: %w", err)
			}

			// Make sure the PR head ref points to the correct commit after the merge.
			updateRefs = append(updateRefs, git.RefUpdate{
				Name: refPullReqHead,
				Old:  sha.SHA{}, // don't care about the old value.
				New:  mergeEntry.HeadCommitSHA,
			})

			// Delete pull request merge reference.
			updateRefs = append(updateRefs, git.RefUpdate{
				Name: refPullReqMerge,
				Old:  sha.SHA{}, // don't care about the old value.
				New:  sha.Nil,   // delete the reference
			})
		}

		// This is a deliberate git call to update git references during a DB transaction.
		// In general this should not be done. But here, we update a bunch of pull requests
		// and remove merge queue entries. And the git call is a simple references update,
		// which should be fast enough.
		err = s.git.UpdateRefs(ctx, git.UpdateRefsParams{
			WriteParams: writeParams,
			Refs:        updateRefs,
		})
		if err != nil {
			return fmt.Errorf("failed to fast-forward merge queue branch to merge commit: %w", err)
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return fmt.Errorf("failed to fast-forward merge queue branch to merge commit: %w", err)
	}

	// At this point pull requests are merged (in the DB and in the git repository).
	// Their merge queue entries are removed from the DB.
	// Now, need to attempt to delete the source branch and publish the merges.

	for i, pullReq := range pullReqs {
		var branchDeleted bool
		seqSourceBranchDeleted, ok := deleteSourceBranchMap[pullReq.ID]
		if ok {
			branchDeleted = s.mergeService.DeleteBranchTry(ctx, pullReq, systemPrincipalInfo, seqSourceBranchDeleted)
		}

		mergeEntry := mergeEntries[i]
		mergeOutput := git.MergeOutput{
			BaseSHA:          mergeEntry.BaseCommitSHA,
			HeadSHA:          mergeEntry.HeadCommitSHA,
			MergeBaseSHA:     mergeEntry.MergeBaseSHA,
			MergeSHA:         mergeEntry.MergeCommitSHA,
			CommitCount:      mergeEntry.CommitCount,
			ChangedFileCount: mergeEntry.ChangedFileCount,
			Additions:        mergeEntry.Additions,
			Deletions:        mergeEntry.Deletions,
		}

		s.mergeService.Publish(ctx, pullReq, repo, mergeEntry.MergeMethod, mergeOutput, systemPrincipalInfo)

		log.Info().
			Str("merge_method", string(*pullReq.MergeMethod)).
			Bool("branch_deleted", branchDeleted).
			Msg("successfully fast-forwarded pull request from merge queue to target branch")
	}

	return nil
}

var errIncompleteMergeQueue = errors.New("incomplete merge queue")

func (s *Service) findEntriesToMerge(
	entry *types.MergeQueueEntry,
	entries []*types.MergeQueueEntry,
) ([]*types.MergeQueueEntry, error) {
	for i, e := range entries {
		if e.BaseCommitSHA.IsEmpty() || e.MergeCommitSHA.IsEmpty() {
			return nil, errIncompleteMergeQueue
		}

		if e.OrderIndex != entry.OrderIndex {
			continue
		}

		return entries[:i+1], nil
	}

	return nil, errIncompleteMergeQueue
}
