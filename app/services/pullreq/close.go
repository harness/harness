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
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
	"github.com/rs/zerolog/log"
)

type NonUniqueMergeBaseInput struct {
	PullReqStore      store.PullReqStore
	ActivityStore     store.PullReqActivityStore
	PullReqEvReporter *pullreqevents.Reporter
	SSEStreamer       sse.Streamer
}

func CloseBecauseNonUniqueMergeBase(
	ctx context.Context,
	in NonUniqueMergeBaseInput,
	targetSHA sha.SHA,
	sourceSHA sha.SHA,
	pr *types.PullReq,
) error {
	systemPrincipal := bootstrap.NewSystemServiceSession().Principal
	systemPrincipalID := systemPrincipal.ID

	var activitySeqMergeBase, activitySeqPRClosed int64
	pr, err := in.PullReqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		// to avoid racing conditions with merge
		if pr.State != enum.PullReqStateOpen {
			return errPRNotOpen
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
	if errors.Is(err, errPRNotOpen) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to close pull request after non-unique merge base: %w", err)
	}

	pr.ActivitySeq = activitySeqMergeBase
	payloadNonUniqueMergeBase := &types.PullRequestActivityPayloadNonUniqueMergeBase{
		TargetSHA: targetSHA,
		SourceSHA: sourceSHA,
	}
	_, err = in.ActivityStore.CreateWithPayload(ctx, pr, systemPrincipalID, payloadNonUniqueMergeBase, nil)
	if err != nil {
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
	if _, err := in.ActivityStore.CreateWithPayload(ctx, pr, systemPrincipalID, payloadStateChange, nil); err != nil {
		// non-critical error
		log.Ctx(ctx).Err(err).Msg(
			"failed to write pull request activity for pull request closure after non-unique merge-base",
		)
	}

	in.PullReqEvReporter.Closed(ctx, &pullreqevents.ClosedPayload{
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

	in.SSEStreamer.Publish(ctx, pr.TargetRepoID, enum.SSETypePullReqUpdated, pr)

	return nil
}
