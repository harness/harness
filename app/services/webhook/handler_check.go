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

package webhook

import (
	"context"
	"fmt"

	checkevents "github.com/harness/gitness/app/events/check"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git/sha"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// PullReqCheckCompletedPayload describes the body of the pullreq check completed trigger.
type PullReqCheckCompletedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	PullReqCheckSegment
}

// handleEventCheckReported fans terminal check reports out to open PRs on that SHA.
func (s *Service) handleEventCheckReported(
	ctx context.Context,
	event *events.Event[*checkevents.ReportedPayload],
) error {
	if !event.Payload.Status.IsCompleted() {
		return nil
	}

	sourceCommitSHA, err := sha.New(event.Payload.SHA)
	if err != nil {
		return events.NewDiscardEventError(fmt.Errorf("invalid commit SHA format: %s", event.Payload.SHA))
	}

	const largeLimit = 1000000
	pullreqList, err := s.pullreqStore.List(ctx, &types.PullReqFilter{
		Page:         0,
		Size:         largeLimit,
		SourceRepoID: event.Payload.RepoID,
		SourceSHA:    sourceCommitSHA,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
		Sort:         enum.PullReqSortNumber,
		Order:        enum.OrderAsc,
	})
	if err != nil {
		return fmt.Errorf("failed to list open pull requests for check completed webhook: %w", err)
	}

	for _, pr := range pullreqList {
		// Distinct trigger id per PR so one check report can fan out cleanly.
		triggerEventID := fmt.Sprintf("%s-pr-%d", event.ID, pr.ID)

		err = s.triggerForEventWithPullReq(
			ctx,
			enum.WebhookTriggerPullReqCheckCompleted,
			triggerEventID, pr.CreatedBy, pr.ID,
			func(
				principal *types.Principal,
				pr *types.PullReq,
				targetRepo,
				sourceRepo *types.Repository,
			) (any, error) {
				targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
				sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

				return &PullReqCheckCompletedPayload{
					BaseSegment: BaseSegment{
						Trigger:   enum.WebhookTriggerPullReqCheckCompleted,
						Repo:      targetRepoInfo,
						Principal: principalInfoFrom(principal.ToPrincipalInfo()),
					},
					PullReqSegment: PullReqSegment{
						PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
					},
					PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
						TargetRef: ReferenceInfo{
							Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
							Repo: targetRepoInfo,
						},
					},
					ReferenceSegment: ReferenceSegment{
						Ref: ReferenceInfo{
							Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
							Repo: sourceRepoInfo,
						},
					},
					PullReqCheckSegment: PullReqCheckSegment{
						Check: CheckInfo{
							Identifier: event.Payload.Identifier,
							Name:       event.Payload.Identifier,
							Status:     event.Payload.Status,
						},
					},
				}, nil
			})
		if err != nil {
			log.Ctx(ctx).Err(err).Msgf("failed to trigger check completed webhook for PR %d", pr.Number)
		}
	}

	return nil
}
