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

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var (
	errPRNotOpen = errors.New("PR is not open")
)

// triggerPREventOnBranchUpdate handles branch update events. For every open pull request
// it writes an activity entry and triggers the pull request Branch Updated event.
//
//nolint:gocognit // refactor if needed
func (s *Service) triggerPREventOnBranchUpdate(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	// we should always update PR mergeable status check when target branch is updated.
	// - main
	//    |- develop
	//         |- feature1
	//         |- feature2
	// when feature2 merge changes into develop branch then feature1 branch is not consistent anymore
	// and need to run mergeable check even nothing was changed on feature1, same applies to main if someone
	// push new commit to main then develop should merge status should be unchecked.
	if branch, err := getBranchFromRef(event.Payload.Ref); err == nil {
		err = s.pullreqStore.ResetMergeCheckStatus(ctx, event.Payload.RepoID, branch)
		if err != nil {
			return err
		}
	}

	// TODO: This function is currently executed directly on branch update event.
	// TODO: But it should be executed after the PR's head ref has been updated.
	// TODO: This is to make sure the commit exists on the target repository for forked repositories.
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		// First check if the merge base has changed

		targetRepo, err := s.repoGitInfoCache.Get(ctx, pr.TargetRepoID)
		if err != nil {
			return fmt.Errorf("failed to get repo git info: %w", err)
		}

		mergeBaseInfo, err := s.git.MergeBase(ctx, git.MergeBaseParams{
			ReadParams: git.ReadParams{RepoUID: targetRepo.GitUID},
			Ref1:       event.Payload.NewSHA,
			Ref2:       pr.TargetBranch,
		})
		if err != nil {
			return fmt.Errorf("failed to get merge base after branch update to=%s for PR=%d: %w",
				event.Payload.NewSHA, pr.Number, err)
		}

		oldMergeBase := pr.MergeBaseSHA
		newMergeBase := mergeBaseInfo.MergeBaseSHA

		// Update the database with the latest source commit SHA and the merge base SHA.
		pr, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			// to avoid racing conditions with merge
			if pr.State != enum.PullReqStateOpen {
				return errPRNotOpen
			}

			pr.ActivitySeq++
			if pr.SourceSHA != event.Payload.OldSHA {
				return fmt.Errorf(
					"failed to set SourceSHA for PR %d to value '%s', expected SHA '%s' but current pr has '%s'",
					pr.Number, event.Payload.NewSHA, event.Payload.OldSHA, pr.SourceSHA)
			}

			pr.Edited = time.Now().UnixMilli()
			pr.SourceSHA = event.Payload.NewSHA
			pr.MergeBaseSHA = newMergeBase.String()

			// reset merge-check fields for new run
			pr.MergeCheckStatus = enum.MergeCheckStatusUnchecked
			pr.MergeSHA = nil
			pr.MergeConflicts = nil
			pr.Stats.DiffStats.Commits = nil
			pr.Stats.DiffStats.FilesChanged = nil

			return nil
		})
		if errors.Is(err, errPRNotOpen) {
			return nil
		}
		if err != nil {
			return err
		}

		payload := &types.PullRequestActivityPayloadBranchUpdate{
			Old: event.Payload.OldSHA,
			New: event.Payload.NewSHA,
		}

		_, err = s.activityStore.CreateWithPayload(ctx, pr, event.Payload.PrincipalID, payload, nil)
		if err != nil {
			// non-critical error
			log.Ctx(ctx).Err(err).Msgf("failed to write pull request activity after branch update")
		}

		s.pullreqEvReporter.BranchUpdated(ctx, &pullreqevents.BranchUpdatedPayload{
			Base: pullreqevents.Base{
				PullReqID:    pr.ID,
				SourceRepoID: pr.SourceRepoID,
				TargetRepoID: pr.TargetRepoID,
				PrincipalID:  event.Payload.PrincipalID,
				Number:       pr.Number,
			},
			OldSHA:          event.Payload.OldSHA,
			NewSHA:          event.Payload.NewSHA,
			OldMergeBaseSHA: oldMergeBase,
			NewMergeBaseSHA: newMergeBase.String(),
			Forced:          event.Payload.Forced,
		})

		if err = s.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
		}

		return nil
	})
	return nil
}

// closePullReqOnBranchDelete handles branch delete events.
// It closes every open pull request for the branch and triggers the pull request BranchDeleted event.
func (s *Service) closePullReqOnBranchDelete(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload],
) error {
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		targetRepo, err := s.repoGitInfoCache.Get(ctx, pr.TargetRepoID)
		if err != nil {
			return fmt.Errorf("failed to get repo info: %w", err)
		}

		var activitySeqBranchDeleted, activitySeqPRClosed int64
		pr, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			// to avoid racing conditions with merge
			if pr.State != enum.PullReqStateOpen {
				return errPRNotOpen
			}

			// get sequence numbers for both activities (branch deletion should be first)
			pr.ActivitySeq += 2
			activitySeqBranchDeleted = pr.ActivitySeq - 1
			activitySeqPRClosed = pr.ActivitySeq

			pr.State = enum.PullReqStateClosed
			pr.MergeCheckStatus = enum.MergeCheckStatusUnchecked
			pr.MergeSHA = nil
			pr.MergeConflicts = nil

			return nil
		})
		if errors.Is(err, errPRNotOpen) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("failed to close pull request after branch delete: %w", err)
		}

		// NOTE: We use the latest PR source sha for the branch deleted activity.
		// There is a chance the PR is behind, but we can't guarantee any missing commit exists after branch deletion.
		// Whatever is the source sha of the PR is most likely to be pointed at by the PR head ref.
		pr.ActivitySeq = activitySeqBranchDeleted
		_, err = s.activityStore.CreateWithPayload(ctx, pr, event.Payload.PrincipalID,
			&types.PullRequestActivityPayloadBranchDelete{SHA: pr.SourceSHA}, nil)
		if err != nil {
			// non-critical error
			log.Ctx(ctx).Err(err).Msg("failed to write pull request activity for branch deletion")
		}

		pr.ActivitySeq = activitySeqPRClosed
		payload := &types.PullRequestActivityPayloadStateChange{
			Old:      enum.PullReqStateOpen,
			New:      enum.PullReqStateClosed,
			OldDraft: pr.IsDraft,
			NewDraft: pr.IsDraft,
		}
		if _, err := s.activityStore.CreateWithPayload(ctx, pr, event.Payload.PrincipalID, payload, nil); err != nil {
			// non-critical error
			log.Ctx(ctx).Err(err).Msg(
				"failed to write pull request activity for pullrequest closure after branch deletion",
			)
		}

		s.pullreqEvReporter.Closed(ctx, &pullreqevents.ClosedPayload{
			Base: pullreqevents.Base{
				PullReqID:    pr.ID,
				SourceRepoID: pr.SourceRepoID,
				TargetRepoID: pr.TargetRepoID,
				PrincipalID:  event.Payload.PrincipalID,
				Number:       pr.Number,
			},
			SourceSHA: pr.SourceSHA,
		})

		if err = s.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullRequestUpdated, pr); err != nil {
			log.Ctx(ctx).Warn().Err(err).Msg("failed to publish PR changed event")
		}

		return nil
	})
	return nil
}

// forEveryOpenPR is utility function that executes the provided function
// for every open pull request created with the source branch given as a git ref.
func (s *Service) forEveryOpenPR(ctx context.Context,
	repoID int64, ref string,
	fn func(pr *types.PullReq) error,
) {
	const largeLimit = 1000000

	branch, err := getBranchFromRef(ref)
	if len(branch) == 0 {
		log.Ctx(ctx).Err(err).Send()
		return
	}

	pullreqList, err := s.pullreqStore.List(ctx, &types.PullReqFilter{
		Page:         0,
		Size:         largeLimit,
		SourceRepoID: repoID,
		SourceBranch: branch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to get list of open pull requests")
		return
	}

	for _, pr := range pullreqList {
		if err = fn(pr); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to process pull req")
		}
	}
}

func getBranchFromRef(ref string) (string, error) {
	const refPrefix = "refs/heads/"
	if !strings.HasPrefix(ref, refPrefix) {
		return "", fmt.Errorf("failed to get branch name from branch ref %s", ref)
	}

	branch := ref[len(refPrefix):]
	if len(branch) == 0 {
		return "", fmt.Errorf("got an empty branch name from branch ref %s", ref)
	}
	return branch, nil
}
