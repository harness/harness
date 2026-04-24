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
	"strings"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
)

type UserGroupReviewerAddedPayload struct {
	Base          *BasePullReqPayload
	ReviewerCount int
	ReviewerNames string
}

func (s *Service) notifyUserGroupReviewerAdded(
	ctx context.Context,
	event *events.Event[*pullreqevents.UserGroupReviewerAddedPayload],
) error {
	base, members, err := s.processUserGroupReviewerAddedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.UserGroupReviewerAdded,
			event.Payload.PullReqID,
			err,
		)
	}

	if len(members) == 0 {
		return nil
	}

	reviewerNames := make([]string, 0, len(members))
	for _, member := range members {
		reviewerNames = append(reviewerNames, member.DisplayName)
	}

	// Send ONE email to the author listing all reviewers
	authorPayload := &UserGroupReviewerAddedPayload{
		Base:          base,
		ReviewerCount: len(members),
		ReviewerNames: strings.Join(reviewerNames, ", "),
	}

	err = s.notificationClient.SendUserGroupReviewerAdded(ctx, []*types.PrincipalInfo{base.Author}, authorPayload)
	if err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.UserGroupReviewerAdded,
			event.Payload.PullReqID,
			err,
		)
	}

	// Send individual emails to each reviewer (like they were individually added)
	for _, member := range members {
		reviewerPayload := &ReviewerAddedPayload{
			Base:     base,
			Reviewer: member,
		}

		err = s.notificationClient.SendReviewerAdded(ctx, []*types.PrincipalInfo{member}, reviewerPayload)
		if err != nil {
			return fmt.Errorf(
				"failed to send email for event %s for pullReqID %d: %w",
				pullreqevents.UserGroupReviewerAdded,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	return nil
}

func (s *Service) processUserGroupReviewerAddedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.UserGroupReviewerAddedPayload],
) (*BasePullReqPayload, []*types.PrincipalInfo, error) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	members := make([]*types.PrincipalInfo, 0, len(event.Payload.ReviewerIDs))
	for _, reviewerID := range event.Payload.ReviewerIDs {
		principal, err := s.principalInfoCache.Get(ctx, reviewerID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get reviewer from principalInfoCache: %w", err)
		}

		if principal.ID == base.Author.ID {
			continue
		}

		members = append(members, principal)
	}

	return base, members, nil
}
