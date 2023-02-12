// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strings"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// triggerPREventOnBranchUpdate handles branch update events. For every open pull request
// it writes an activity entry and triggers the pull request Branch Updated event.
func (s *Service) triggerPREventOnBranchUpdate(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload],
) error {
	s.forEveryOpenPR(ctx, event.Payload.RepoID, event.Payload.Ref, func(pr *types.PullReq) error {
		pr, err := s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			pr.ActivitySeq++
			if pr.SourceSHA != event.Payload.OldSHA {
				return fmt.Errorf(
					"failed to set SourceSHA for PR %d to value '%s', expected SHA '%s' but current pr has '%s'",
					pr.Number, event.Payload.NewSHA, event.Payload.OldSHA, pr.SourceSHA)
			}
			pr.SourceSHA = event.Payload.NewSHA
			pr.MergeRefSHA = nil
			pr.MergeBaseSHA = nil
			pr.MergeHeadSHA = nil
			return nil
		})
		if err != nil {
			return err
		}

		payload := &types.PullRequestActivityPayloadBranchUpdate{
			Old: event.Payload.OldSHA,
			New: event.Payload.NewSHA,
		}

		_, err = s.activityStore.CreateWithPayload(ctx, pr, event.Payload.PrincipalID, payload)
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
			OldSHA: event.Payload.OldSHA,
			NewSHA: event.Payload.NewSHA,
			Forced: event.Payload.Forced,
		})
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
		pr, err := s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
			pr.State = enum.PullReqStateClosed
			pr.MergeRefSHA = nil
			pr.MergeBaseSHA = nil
			pr.MergeHeadSHA = nil
			pr.ActivitySeq++ // because we need to write the activity
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to close pull request after branch delete: %w", err)
		}

		_, errAct := s.activityStore.CreateWithPayload(ctx, pr, event.Payload.PrincipalID,
			&types.PullRequestActivityPayloadBranchDelete{SHA: event.Payload.SHA})
		if errAct != nil {
			// non-critical error
			log.Ctx(ctx).Err(errAct).Msgf("failed to write pull request activity after branch delete")
		}

		s.pullreqEvReporter.Closed(ctx, &pullreqevents.ClosedPayload{
			Base: pullreqevents.Base{
				PullReqID:    pr.ID,
				SourceRepoID: pr.SourceRepoID,
				TargetRepoID: pr.TargetRepoID,
				PrincipalID:  event.Payload.PrincipalID,
				Number:       pr.Number,
			},
		})
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
	const refPrefix = "refs/heads/"
	const largeLimit = 1000000

	if !strings.HasPrefix(ref, refPrefix) {
		log.Ctx(ctx).Error().Msg("failed to get branch name from branch ref")
		return
	}

	branch := ref[len(refPrefix):]
	if len(branch) == 0 {
		log.Ctx(ctx).Error().Msg("got an empty branch name from branch ref")
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
