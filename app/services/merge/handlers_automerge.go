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

	checkevents "github.com/harness/gitness/app/events/check"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

func (s *Service) mergePRsOnMergeCheckSucceeded(
	ctx context.Context,
	event *events.Event[*pullreqevents.MergeCheckSucceededPayload],
) error {
	pr, err := s.pullreqStore.Find(ctx, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request by ID for automerge: %w", err)
	}

	return s.autoMerge(ctx, pr)
}

func (s *Service) mergePRsOnApproval(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewSubmittedPayload],
) error {
	if event.Payload.Decision != enum.PullReqReviewDecisionApproved {
		return nil
	}

	pr, err := s.pullreqStore.Find(ctx, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request by ID for automerge: %w", err)
	}

	return s.autoMerge(ctx, pr)
}

func (s *Service) mergePRsOnCommentResolve(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentStatusUpdatedPayload],
) error {
	if event.Payload.OldStatus != enum.PullReqCommentStatusActive ||
		event.Payload.NewStatus != enum.PullReqCommentStatusResolved {
		return nil
	}

	pr, err := s.pullreqStore.Find(ctx, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("failed to find pull request by ID for automerge: %w", err)
	}

	return s.autoMerge(ctx, pr)
}

func (s *Service) mergePRsOnCheckSucceeded(
	ctx context.Context,
	event *events.Event[*checkevents.ReportedPayload],
) error {
	if event.Payload.Status != enum.CheckStatusSuccess {
		return nil
	}

	sourceCommitSHA, err := sha.New(event.Payload.SHA)
	if err != nil {
		return events.NewDiscardEventError(fmt.Errorf("invalid commit SHA format: %s", event.Payload.SHA))
	}

	err = s.forEveryOpenPullReqWithSourceSHA(ctx, event.Payload.RepoID, sourceCommitSHA, func(pr *types.PullReq) error {
		return s.autoMerge(ctx, pr)
	})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to process open pull requests with the specific source SHA")
	}

	return nil
}

// forEveryOpenPullReqWithSourceSHA is utility function that executes the provided function
// for every open pull request with the specific source commit SHA.
func (s *Service) forEveryOpenPullReqWithSourceSHA(
	ctx context.Context,
	repoID int64,
	commitSHA sha.SHA,
	fn func(pr *types.PullReq) error,
) error {
	const largeLimit = 1000000

	pullreqList, err := s.pullreqStore.List(ctx, &types.PullReqFilter{
		Page:             0,
		Size:             largeLimit,
		SourceRepoID:     repoID,
		SourceSHA:        commitSHA,
		States:           []enum.PullReqState{enum.PullReqStateOpen},
		IsDraft:          ptr.Bool(false),
		MergeCheckStatus: ptr.Of[enum.MergeCheckStatus](enum.MergeCheckStatusMergeable),
		Sort:             enum.PullReqSortNumber,
		Order:            enum.OrderAsc,
	})
	if err != nil {
		return fmt.Errorf("failed to get list of open pull requests: %w", err)
	}

	for _, pr := range pullreqList {
		if err = fn(pr); err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to process pull req")
		}
	}

	return nil
}
