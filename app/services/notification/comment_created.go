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
)

type CommentCreatedPayload struct {
	Base      *BasePullReqPayload
	Commenter *types.PrincipalInfo
	Text      string
}

func (s *Service) notifyCommentCreated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentCreatedPayload],
) error {
	payload, recipients, err := s.processCommentCreatedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.CommentCreatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	err = s.notificationClient.SendCommentCreated(ctx, recipients, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.CommentCreatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}
	return nil
}

func (s *Service) processCommentCreatedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentCreatedPayload],
) (*CommentCreatedPayload, []*types.PrincipalInfo, error) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	activity, err := s.pullReqActivityStore.Find(ctx, event.Payload.ActivityID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch activity from pullReqActivityStore: %w", err)
	}

	commenter, err := s.principalInfoView.Find(ctx, activity.CreatedBy)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch commenter from principalInfoView: %w", err)
	}
	recipients := []*types.PrincipalInfo{
		base.Author,
	}

	return &CommentCreatedPayload{
		Base:      base,
		Commenter: commenter,
		Text:      activity.Text,
	}, recipients, nil
}
