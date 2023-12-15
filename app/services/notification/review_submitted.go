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

package notification

import (
	"context"
	"fmt"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type ReviewSubmittedPayload struct {
	Base     *BasePullReqPayload
	Author   *types.PrincipalInfo
	Reviewer *types.PrincipalInfo
	Decision enum.PullReqReviewDecision
}

func (s *Service) notifyReviewSubmitted(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewSubmittedPayload],
) error {
	notificationPayload, recipients, err := s.processReviewSubmittedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.ReviewSubmittedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	err = s.notificationClient.SendReviewSubmitted(
		ctx,
		recipients,
		notificationPayload,
	)

	if err != nil {
		return fmt.Errorf(
			"failed to send notification for event %s for pullReqID %d: %w",
			pullreqevents.ReviewSubmittedEvent,
			event.Payload.PullReqID,
			err,
		)
	}
	return nil
}

func (s *Service) processReviewSubmittedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewSubmittedPayload],
) (*ReviewSubmittedPayload, []*types.PrincipalInfo, error) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	authorPrincipal, err := s.principalInfoCache.Get(ctx, base.PullReq.CreatedBy)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to get author from principalInfoCache on %s event for pullReqID %d: %w",
			pullreqevents.ReviewSubmittedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	reviewerPrincipal, err := s.principalInfoCache.Get(ctx, event.Payload.ReviewerID)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to get reviewer from principalInfoCache on event %s for pullReqID %d: %w",
			pullreqevents.ReviewSubmittedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	return &ReviewSubmittedPayload{
		Base:     base,
		Author:   authorPrincipal,
		Decision: event.Payload.Decision,
		Reviewer: reviewerPrincipal,
	}, []*types.PrincipalInfo{authorPrincipal}, nil
}
