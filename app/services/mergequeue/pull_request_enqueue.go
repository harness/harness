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
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/bootstrap"
	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/app/services/merge"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// Enqueue enqueues the pull request in the merge queue.
// To be called from the API controller.
func (s *Service) Enqueue(
	ctx context.Context,
	pr *types.PullReq,
	targetRepo *types.RepositoryCore,
	principalID int64,
	mergeMethod enum.MergeMethod,
	commitTitle string,
	commitMessage string,
	deleteBranch bool,
) (*types.PullReq, []types.RuleViolations, error) {
	if m, ok := mergeMethod.Sanitize(); ok {
		mergeMethod = m
	} else {
		return nil, nil, usererror.BadRequestf("Invalid merge method: %q.", mergeMethod)
	}

	if mergeMethod == enum.MergeMethodFastForward {
		return nil, nil, usererror.BadRequest("Fast forward method is not supported by merge queue.")
	}

	err := s.VerifyIfMergeQueueable(pr)
	if err != nil {
		return nil, nil, err
	}

	sourceRepo := targetRepo
	if pr.SourceRepoID != nil && pr.TargetRepoID != *pr.SourceRepoID {
		sourceRepo, err = s.repoFinder.FindByID(ctx, *pr.SourceRepoID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to find source repo: %w", err)
		}
	}

	branchProtection, err := s.protectionManager.ListRepoBranchRules(ctx, targetRepo.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list repository branch rules: %w", err)
	}

	// use the system principal, as it will merge the PR in the merge queue
	systemSession := bootstrap.NewSystemServiceSession()

	_, violations, err := s.mergeService.CheckRules(ctx, branchProtection, merge.CheckRulesInput{
		PullReq:          pr,
		TargetRepo:       targetRepo,
		SourceRepo:       sourceRepo,
		Actor:            &systemSession.Principal,
		IsRepoOwner:      false,
		MergeMethod:      mergeMethod,
		AllowBypassRules: false, // system principal can't bypass rules
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify protection rules: %w", err)
	}

	if protection.IsCritical(violations) {
		return nil, violations, nil
	}

	setup, err := branchProtection.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         targetRepo,
		TargetBranch: pr.TargetBranch,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	if !setup.IsActive() {
		return nil, nil,
			usererror.BadRequestf("Merge queue has not been configured for branch %q.", pr.TargetBranch)
	}

	count, err := s.mergeQueueEntryStore.CountForRepoAndBranch(ctx, targetRepo.ID, pr.TargetBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get merge queue entry count: %w", err)
	}

	if count >= MaximumQueueSize {
		return nil, nil, usererror.BadRequestf("Merge queue is full (maximum %d entries).", MaximumQueueSize)
	}

	q, err := s.FindOrCreateMergeQueue(ctx, targetRepo.ID, pr.TargetBranch)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create merge queue: %w", err)
	}

	prID := pr.ID

	q, seq, err := s.reserveSequenceNumber(ctx, q)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to reserve merge queue entry sequence number: %w", err)
	}

	var entry *types.MergeQueueEntry

	err = controller.TxOptLock(ctx, s.tx, func(ctx context.Context) error {
		pr, err = s.pullreqStore.Find(ctx, prID)
		if err != nil {
			return fmt.Errorf("failed to find pull request: %w", err)
		}

		err = s.VerifyIfMergeQueueable(pr)
		if err != nil {
			return err
		}

		now := time.Now().UnixMilli()
		entry = &types.MergeQueueEntry{
			PullReqID:          pr.ID,
			MergeQueueID:       q.ID,
			Version:            0,
			CreatedBy:          principalID,
			Created:            now,
			Updated:            now,
			OrderIndex:         seq,
			State:              enum.MergeQueueEntryStateMergePending,
			BaseCommitSHA:      sha.None,
			HeadCommitSHA:      sha.None,
			MergeCommitSHA:     sha.None,
			MergeBaseSHA:       sha.None,
			CommitCount:        0,
			ChangedFileCount:   0,
			Additions:          0,
			Deletions:          0,
			ChecksCommitSHA:    sha.None,
			ChecksStarted:      nil,
			ChecksDeadline:     nil,
			MergeMethod:        mergeMethod,
			CommitTitle:        commitTitle,
			CommitMessage:      commitMessage,
			DeleteSourceBranch: deleteBranch,
		}

		err = s.mergeQueueEntryStore.Create(ctx, entry)
		if err != nil {
			return fmt.Errorf("failed to create merge queue entry: %w", err)
		}

		pr.SubState = enum.PullReqSubStateMergeQueue
		pr.ActivitySeq++

		err = s.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		return nil
	}, dbtx.TxRepeatableRead)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create merge queue: %w", err)
	}

	payload := &types.PullRequestActivityPayloadMergeQueueAdd{
		MergeMethod: entry.MergeMethod,
	}

	_, err = s.activityStore.CreateWithPayload(ctx, pr, principalID, payload, nil)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("failed to create merge queue pull request activity")
	}

	s.mergeQueueEventReporter.Updated(ctx, &mergequeueevents.UpdatedPayload{
		Base: mergequeueevents.Base{
			RepoID: q.RepoID,
			Branch: q.Branch,
		},
	})

	return pr, nil, nil
}

func (s *Service) reserveSequenceNumber(
	ctx context.Context,
	q *types.MergeQueue,
) (*types.MergeQueue, int64, error) {
	var seq int64
	q, err := s.mergeQueueStore.UpdateOptLock(ctx, q, func(q *types.MergeQueue) error {
		q.OrderSequence++
		seq = q.OrderSequence
		if seq <= 0 {
			return fmt.Errorf("invalid sequence number of merge queue: %d", seq)
		}
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("failed to increment sequence number: %w", err)
	}

	return q, seq, nil
}

func (*Service) VerifyIfMergeQueueable(pr *types.PullReq) error {
	if pr.Merged != nil {
		return usererror.BadRequest("Pull request is already merged.")
	}

	if pr.State != enum.PullReqStateOpen {
		return usererror.BadRequest("Pull request must be open.")
	}

	if pr.IsDraft {
		return usererror.BadRequest("Draft pull requests cannot be added to the merge queue. Clear the draft flag first.")
	}

	if pr.SubState == enum.PullReqSubStateMergeQueue {
		return usererror.BadRequest("Pull request is already in the merge queue.")
	}

	if pr.SubState != "" {
		return usererror.BadRequest("Pull request must not be in a sub-state.")
	}

	return nil
}
