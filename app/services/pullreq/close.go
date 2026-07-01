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
	"fmt"

	"github.com/harness/gitness/app/bootstrap"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

func (s *Service) CloseBecauseNonUniqueMergeBase(
	ctx context.Context,
	targetSHA sha.SHA,
	sourceSHA sha.SHA,
	pr *types.PullReq,
) error {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	systemPrincipalID := systemPrincipal.ID

	repo, err := s.repoStore.Find(ctx, pr.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to find repository: %w", err)
	}

	var activitySeqMergeBase, activitySeqPRClosed int64
	pr, err = s.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		// to avoid racing conditions with merge
		if pr.State != enum.PullReqStateOpen {
			return ErrPullReqNotOpen
		}

		pr.ActivitySeq += 2
		activitySeqMergeBase = pr.ActivitySeq - 1
		activitySeqPRClosed = pr.ActivitySeq

		pr.SourceSHA = sourceSHA.String()
		pr.MergeTargetSHA = ptr.String(targetSHA.String())

		pr.State = enum.PullReqStateClosed
		pr.SubState = enum.PullReqSubStateNone

		pr.MergeSHA = nil
		pr.MarkAsMergeUnchecked()

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to close pull request after non-unique merge base: %w", err)
	}

	// The pull request has been closed. All subsequent operations are best-effort,
	// so use a non-cancellable context to make sure they complete even if the caller's
	// context gets canceled, and don't fail the function on their failures.

	ctxNoCancel := context.WithoutCancel(ctx)

	_, err = s.repoStore.UpdateOptLock(ctxNoCancel, repo, func(repo *types.Repository) error {
		repo.NumClosedPulls++
		repo.NumOpenPulls--
		return nil
	})
	if err != nil {
		log.Ctx(ctx).Err(err).
			Int("num_open_pulls_delta", -1).
			Int("num_closed_pulls_delta", 1).
			Msg("failed to update number of pull requests in repository after PR close on non-unique merge base")
	}

	pr.ActivitySeq = activitySeqMergeBase
	payloadNonUniqueMergeBase := &types.PullRequestActivityPayloadNonUniqueMergeBase{
		TargetSHA: targetSHA,
		SourceSHA: sourceSHA,
	}
	if _, err := s.activityStore.CreateWithPayload(
		ctxNoCancel, pr, systemPrincipalID, payloadNonUniqueMergeBase, nil,
	); err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msg("failed to write pull request activity for non-unique merge-base")
	}

	pr.ActivitySeq = activitySeqPRClosed
	payloadStateChange := &types.PullRequestActivityPayloadStateChange{
		Old:      enum.PullReqStateOpen,
		New:      enum.PullReqStateClosed,
		OldDraft: pr.IsDraft,
		NewDraft: pr.IsDraft,
	}
	if _, err := s.activityStore.CreateWithPayload(
		ctxNoCancel, pr, systemPrincipalID, payloadStateChange, nil,
	); err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msg(
			"failed to write pull request activity for pull request closure after non-unique merge-base",
		)
	}

	s.pullreqEvReporter.Closed(ctxNoCancel, &pullreqevents.ClosedPayload{
		Base: pullreqevents.Base{
			PullReqID:    pr.ID,
			SourceRepoID: pr.SourceRepoID,
			TargetRepoID: pr.TargetRepoID,
			PrincipalID:  systemPrincipalID,
			Number:       pr.Number,
		},
		SourceSHA:    pr.SourceSHA,
		SourceBranch: pr.SourceBranch,
	})

	s.sseStreamer.Publish(ctxNoCancel, pr.TargetRepoID, enum.SSETypePullReqUpdated, pr)

	return nil
}
