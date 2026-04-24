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
	"slices"

	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// handlerCheckFinished is invoked whenever a status check is reported for any commit.
// It looks up any merge queue entries whose ChecksCommitSHA matches the reported SHA and
// acts on the result: successful checks advance the entry toward being merged;
// failed checks remove the entry from the queue.
func (s *Service) handlerCheckFinished(
	ctx context.Context,
	event *events.Event[*checkevents.ReportedPayload],
) error {
	status := event.Payload.Status

	if status != enum.CheckStatusSuccess && status != enum.CheckStatusFailure {
		return nil
	}

	repoID := event.Payload.RepoID
	commitSHA, err := sha.New(event.Payload.SHA)
	if err != nil {
		err = fmt.Errorf("invalid commit SHA format: %q: %w", event.Payload.SHA, err)
		return events.NewDiscardEventError(err)
	}

	repo, err := s.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo: %w", err)
	}

	// Find merge queue entry by merge commit in the DB. This is column has a unique index,
	// because every entry's merge commit must be unique.
	entry, err := s.mergeQueueEntryStore.FindByMergeCommit(ctx, commitSHA)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil // This check is not a merge queue check. We're done.
	}
	if err != nil {
		return fmt.Errorf("failed to list merge queue entries by checks commit SHA: %w", err)
	}

	q, err := s.mergeQueueStore.Find(ctx, entry.MergeQueueID)
	if err != nil {
		return fmt.Errorf("failed to find merge queue from entry's ID: %w", err)
	}

	branchProtection, err := s.protectionManager.ListRepoBranchRules(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to list repo branch rules: %w", err)
	}

	mergeQueueSetup, err := branchProtection.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         repo,
		TargetBranch: q.Branch,
	})
	if err != nil {
		return fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	if !mergeQueueSetup.IsActive() {
		err = s.removeAll(ctx, repo, q)
		if err != nil {
			return fmt.Errorf("failed to clear merge queue: %w", err)
		}

		return nil
	}

	// One failure is enough to remove the PR from the merge queue.
	if status == enum.CheckStatusFailure {
		err = s.remove(ctx, entry.PullReqID, enum.MergeQueueRemovalReasonCheckFail)
		if errors.Is(err, ErrNotInQueue) {
			// Perhaps a check was running after the PR has already been taken from the merge queue.
			return nil
		}
		if err != nil {
			log.Ctx(ctx).Warn().Err(err).
				Int64("repo_id", repoID).
				Int64("pullreq_id", entry.PullReqID).
				Msg("failed to remove merge queue entry from merge queue")
			return err
		}

		err = s.reprocess(ctx, repo, q, mergeQueueSetup)
		if err != nil {
			return fmt.Errorf("failed to process merge queue after removing entry because failed check: %w", err)
		}

		return nil
	}

	// Only act on success checks for entries that are the checks leader.
	if entry.State != enum.MergeQueueEntryStateChecksInProgress {
		return nil
	}

	checks, err := s.checkStore.List(ctx, repoID, commitSHA.String(), types.CheckListOptions{
		ListQueryFilter: types.ListQueryFilter{
			Pagination: types.Pagination{Page: 0, Size: MaximumQueueSize},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to list merge queue entries by checks commit SHA: %w", err)
	}

	completedChecks := make([]string, 0, len(checks))
	for _, check := range checks {
		if check.Status == enum.CheckStatusSuccess {
			completedChecks = append(completedChecks, check.Identifier)
		}
	}

	requiredChecks := mergeQueueSetup.RequiredChecks

	for _, check := range requiredChecks {
		if !slices.Contains(completedChecks, check) {
			// Not all required checks have finished successfully. Waiting for more checks to finish...
			return nil
		}
	}

	err = s.fastForward(ctx, q, entry)
	if err != nil {
		return fmt.Errorf("failed to fast forward target branch to the merge commit: %w", err)
	}

	err = s.reprocess(ctx, repo, q, mergeQueueSetup)
	if err != nil {
		return fmt.Errorf("failed to process merge queue after fast forwarding: %w", err)
	}

	return nil
}
