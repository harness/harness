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
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git"
	gitenum "github.com/harness/gitness/git/enum"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (s *Service) reprocess(
	ctx context.Context,
	repo *types.RepositoryCore,
	q *types.MergeQueue,
	mergeQueueSetup protection.MergeQueueSetup,
) error {
	targetSHA, err := s.getLastCommitSHA(ctx, repo, q.Branch)
	if err != nil {
		return fmt.Errorf("failed to get last commit SHA for branch %q: %w", q.Branch, err)
	}

	data, err := s.createProcessCommonData(ctx, repo, q, targetSHA)
	if err != nil {
		return fmt.Errorf("failed to create process common data: %w", err)
	}

	entries, err := s.mergeQueueEntryStore.ListForMergeQueue(ctx, q.ID)
	if err != nil {
		return fmt.Errorf("failed to list merge queue entries: %w", err)
	}

	err = s.process(ctx, data, entries, mergeQueueSetup)
	if err != nil {
		return fmt.Errorf("failed to process merge queue: %w", err)
	}

	return nil
}

// nolint:nestif
func (s *Service) process(
	ctx context.Context,
	data *processData,
	entries []*types.MergeQueueEntry,
	setup protection.MergeQueueSetup,
) error {
	unlock, err := s.locker.LockPR(
		ctx,
		data.repo.ID,
		0,                            // repo level lock
		merge.Timeout+30*time.Second, // add 30s to the lock to give enough time for pre + post merge
	)
	if err != nil {
		return fmt.Errorf("failed to lock repository for merge queue process: %w", err)
	}
	defer unlock()

	lastMergeCommitSHA := sha.Nil

	var (
		prev           *types.MergeQueueEntry
		index          int
		pendingEntries []*types.MergeQueueEntry

		countEntriesProcessed int
		countMergesCreated    int
	)

	// checksToStop collects ChecksCommitSHAs that become stale when new merge commits are created.
	// These are stopped after the main loop to avoid cancelling checks for valid merge commits.
	checksToStop := make(map[sha.SHA]struct{})

	// Limits. Entries beyond these limits remain in merge pending state and will be processed in a future run.
	const (
		maxMergeCommitsInQueue = 20
		maxMergeCommitsCreated = 10
	)

	// Create merge commits if necessary.

	for index = 0; index < len(entries); {
		var pr *types.PullReq
		entry := entries[index]

		pr, err = s.pullreqStore.Find(ctx, entry.PullReqID)
		if err != nil {
			return fmt.Errorf("failed to find pull request: %w", err)
		}

		if pr.State != enum.PullReqStateOpen || pr.SubState != enum.PullReqSubStateMergeQueue || pr.IsDraft {
			// This should not happen in a normal flow. A pull request should be locked while in merge queue.
			log.Ctx(ctx).Warn().Msg("pull request not in merge queue")

			entries = append(entries[:index], entries[index+1:]...)

			err = s.remove(ctx, pr.ID, enum.MergeQueueRemovalReasonNotQueueable)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).
					Msg("failed to remove from merge queue because PR is not queueable")
			}

			continue
		}

		sourceRepo := data.repo
		if pr.SourceRepoID != nil && *pr.SourceRepoID != pr.TargetRepoID {
			sourceRepo, err = s.repoFinder.FindByID(ctx, *pr.SourceRepoID)
			if err != nil {
				return fmt.Errorf("failed to find source repo for pull request: %w", err)
			}
		}

		var headCommitSHA, baseCommitSHA, mergeCommitSHA sha.SHA

		headCommitSHA, err = sha.New(pr.SourceSHA)
		if err != nil {
			// This should not happen. A pull request should always have a valid source commit SHA.
			log.Ctx(ctx).Warn().Err(err).Msg("failed to convert source commit SHA")

			entries = append(entries[:index], entries[index+1:]...)

			err = s.remove(ctx, pr.ID, enum.MergeQueueRemovalReasonError)
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).
					Msg("failed to remove from merge queue because invalid source SHA")
			}

			continue
		}

		if prev == nil {
			baseCommitSHA = data.targetSHA
		} else {
			baseCommitSHA = prev.MergeCommitSHA
		}

		needsMerge :=
			entry.MergeCommitSHA.IsEmpty() ||
				!entry.BaseCommitSHA.Equal(baseCommitSHA) ||
				!entry.HeadCommitSHA.Equal(headCommitSHA)

		switch {
		case needsMerge:
			// If the entry was a checks leader, its old checks are now stale.
			if entry.State == enum.MergeQueueEntryStateChecksInProgress &&
				!entry.ChecksCommitSHA.IsEmpty() && !entry.ChecksCommitSHA.IsNil() {
				checksToStop[entry.ChecksCommitSHA] = struct{}{}
			}

			input := s.mergeService.PreparePullReqMergeInputNoRefUpdates(
				pr,
				sourceRepo,
				data.principalInfo,
				entry.MergeMethod,
				entry.CommitTitle,
				entry.CommitMessage)

			var mergeOutput git.MergeOutput

			mergeOutput, err = s.createMergeCommit(ctx, data.writeParams, baseCommitSHA, headCommitSHA, entry, input)
			if err != nil {
				// This can happen. In case of a conflict or a missing commit, merging can fail.

				reason := enum.MergeQueueRemovalReasonConflict
				if !errors.Is(err, errMergeConflict) {
					log.Ctx(ctx).Warn().Err(err).Msg("failed to create a merge commit")
					reason = enum.MergeQueueRemovalReasonError
				}

				entries = append(entries[:index], entries[index+1:]...)

				err = s.remove(ctx, pr.ID, reason)
				if err != nil {
					log.Ctx(ctx).Warn().Err(err).
						Msg("failed to remove from merge queue because merge commit create failed")
				}

				continue
			}

			mergeCommitSHA = mergeOutput.MergeSHA

			entry, err = s.mergeQueueEntryStore.UpdateOptLock(ctx, entry, func(entry *types.MergeQueueEntry) error {
				entry.State = enum.MergeQueueEntryStateChecksPending
				entry.HeadCommitSHA = headCommitSHA
				entry.BaseCommitSHA = baseCommitSHA
				entry.MergeCommitSHA = mergeCommitSHA
				entry.MergeBaseSHA = mergeOutput.MergeBaseSHA
				entry.CommitCount = mergeOutput.CommitCount
				entry.ChangedFileCount = mergeOutput.ChangedFileCount
				entry.Additions = mergeOutput.Additions
				entry.Deletions = mergeOutput.Deletions
				entry.ChecksCommitSHA = sha.None
				entry.ChecksStarted = nil
				entry.ChecksDeadline = nil
				return nil
			})
			if err != nil {
				log.Ctx(ctx).Warn().Err(err).Msg("failed to update merge queue entry")

				entries = append(entries[:index], entries[index+1:]...)

				err = s.remove(ctx, pr.ID, enum.MergeQueueRemovalReasonError)
				if err != nil {
					log.Ctx(ctx).Warn().Err(err).
						Msg("failed to remove from merge queue because merge queue entry update failed")
				}

				continue
			}

			entries[index] = entry
			countMergesCreated++
		default:
			mergeCommitSHA = entry.MergeCommitSHA
		}

		prev = entry
		index++
		countEntriesProcessed++

		lastMergeCommitSHA = mergeCommitSHA

		if countEntriesProcessed >= maxMergeCommitsInQueue || countMergesCreated >= maxMergeCommitsCreated {
			// Truncate entries to only the processed ones; the rest remain in merge pending state.
			pendingEntries = entries[index:]
			entries = entries[:index]
			break
		}
	}

	// Reset the remaining entries to merge pending state.
	s.resetEntries(ctx, pendingEntries)

	// Update the merge queue reference to point to the last merge commit in the queue.
	s.updateReference(ctx, data, lastMergeCommitSHA)

	// Stop checks that became stale because new merge commits were created.
	for commitSHA := range checksToStop {
		if err := s.stopChecks(ctx, commitSHA); err != nil {
			log.Ctx(ctx).Warn().Err(err).
				Str("commit_sha", commitSHA.String()).
				Msg("failed to stop stale merge queue checks")
		}
	}

	// Find the entries for which checks should be triggered and make merge groups.
	entries, entriesToUpdate := s.updateChecks(
		entries,
		setup.GroupSize,
		setup.ChecksConcurrency,
		setup.MaxCheckDurationSeconds,
		time.Now().UnixMilli(),
	)

	// Update entries to set the checks commits SHA and the timestamp and start checks.

	if len(entriesToUpdate) > 0 {
		err = s.tx.WithTx(ctx, func(ctx context.Context) error {
			for _, entry := range entriesToUpdate {
				err = s.mergeQueueEntryStore.Update(ctx, entry)
				if err != nil {
					// Opt lock errors are not allowed here. We'll restart the whole process if this happens.
					return fmt.Errorf("failed to update merge queue entry %d: %w", entry.PullReqID, err)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update merge queue entries: %w", err)
		}

		for _, entry := range entriesToUpdate {
			if entry.State == enum.MergeQueueEntryStateChecksInProgress {
				if err = s.startChecks(ctx, entry.MergeCommitSHA); err != nil {
					return fmt.Errorf("failed to start checks for entry %d: %w", entry.PullReqID, err)
				}

				log.Ctx(ctx).Info().
					Str("commit_sha", entry.MergeCommitSHA.String()).
					Msg("triggered merge queue checks for merge commit")
			}
		}
	}

	log.Ctx(ctx).Info().
		Int64("repo_id", data.repo.ID).
		Str("branch", data.queue.Branch).
		Int("entries", len(entries)).
		Int("entries_updated", len(entriesToUpdate)).
		Int("entries_processed", countEntriesProcessed).
		Int("merge_commits_created", countMergesCreated).
		Msg("merge queue processing finished")

	return nil
}

type processData struct {
	queue         *types.MergeQueue
	repo          *types.RepositoryCore
	targetSHA     sha.SHA
	session       *auth.Session
	principalInfo *types.PrincipalInfo
	writeParams   git.WriteParams
}

func (s *Service) createProcessCommonData(
	ctx context.Context,
	repo *types.RepositoryCore,
	q *types.MergeQueue,
	targetSHA sha.SHA,
) (*processData, error) {
	session := bootstrap.NewSystemServiceSession()
	principalInfo := session.Principal.ToPrincipalInfo()

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, s.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create rpc system references write params: %w", err)
	}

	return &processData{
		queue:         q,
		repo:          repo,
		targetSHA:     targetSHA,
		session:       session,
		principalInfo: principalInfo,
		writeParams:   writeParams,
	}, nil
}

func (s *Service) resetEntries(ctx context.Context, entries []*types.MergeQueueEntry) {
	for _, entry := range entries {
		if entry.State == enum.MergeQueueEntryStateMergePending {
			continue
		}

		if entry.State == enum.MergeQueueEntryStateChecksInProgress {
			if err := s.stopChecks(ctx, entry.ChecksCommitSHA); err != nil {
				log.Ctx(ctx).Warn().Err(err).
					Str("commit_sha", entry.ChecksCommitSHA.String()).
					Msg("failed to stop merge queue checks")
			}
		}

		_, err := s.mergeQueueEntryStore.UpdateOptLock(ctx, entry, func(entry *types.MergeQueueEntry) error {
			entry.State = enum.MergeQueueEntryStateMergePending
			entry.HeadCommitSHA = sha.None
			entry.BaseCommitSHA = sha.None
			entry.MergeBaseSHA = sha.None
			entry.CancelMerge()
			return nil
		})
		if err != nil {
			log.Ctx(ctx).Error().Err(err).
				Msg("failed to reset merge queue entry")
		}
	}
}

func (s *Service) updateReference(
	ctx context.Context,
	data *processData,
	refSHA sha.SHA,
) {
	err := s.git.UpdateRef(ctx, git.UpdateRefParams{
		WriteParams: data.writeParams,
		Type:        gitenum.RefTypePullReqMergeQueue,
		Name:        data.queue.Branch,
		NewValue:    refSHA,
		OldValue:    sha.None,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).
			Msg("failed to update merge queue reference")
	}
}

func (s *Service) deleteReference(
	ctx context.Context,
	repo *types.RepositoryCore,
	branch string,
) {
	session := bootstrap.NewSystemServiceSession()

	writeParams, err := controller.CreateRPCSystemReferencesWriteParams(ctx, s.urlProvider, session, repo)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).
			Msg("failed to create write params for merge queue reference deletion")
		return
	}

	err = s.git.UpdateRef(ctx, git.UpdateRefParams{
		WriteParams: writeParams,
		Type:        gitenum.RefTypePullReqMergeQueue,
		Name:        branch,
		NewValue:    sha.Nil,
		OldValue:    sha.None,
	})
	if err != nil {
		log.Ctx(ctx).Error().Err(err).
			Msg("failed to delete merge queue reference")
	}
}
