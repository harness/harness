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

// ReviewerAddedReason encodes how a reviewer was added and whether via a user
// group, in one value the template switches on. Empty is the generic fallback.
type ReviewerAddedReason string

const (
	// Direct reviewers.
	ReviewerAddedReasonRequested  ReviewerAddedReason = "requested"
	ReviewerAddedReasonCodeOwners ReviewerAddedReason = "code_owners"
	ReviewerAddedReasonDefault    ReviewerAddedReason = "default"

	// Reviewers added as a member of a reviewer group.
	ReviewerAddedReasonUserGroupRequested  ReviewerAddedReason = "usergroup_requested"
	ReviewerAddedReasonUserGroupCodeOwners ReviewerAddedReason = "usergroup_code_owners"
	ReviewerAddedReasonUserGroupDefault    ReviewerAddedReason = "usergroup_default"
)

// reviewerAddedReason maps a reviewer type + group flag to the rendered reason;
// unknown types fall back to generic ("") direct, or the generic group reason.
func reviewerAddedReason(t enum.PullReqReviewerType, viaUserGroup bool) ReviewerAddedReason {
	if viaUserGroup {
		switch t {
		case enum.PullReqReviewerTypeCodeOwners:
			return ReviewerAddedReasonUserGroupCodeOwners
		case enum.PullReqReviewerTypeDefault:
			return ReviewerAddedReasonUserGroupDefault
		case enum.PullReqReviewerTypeRequested,
			enum.PullReqReviewerTypeAssigned,
			enum.PullReqReviewerTypeSelfAssigned:
			return ReviewerAddedReasonUserGroupRequested
		default:
			return ReviewerAddedReasonUserGroupRequested
		}
	}
	switch t {
	case enum.PullReqReviewerTypeCodeOwners:
		return ReviewerAddedReasonCodeOwners
	case enum.PullReqReviewerTypeDefault:
		return ReviewerAddedReasonDefault
	case enum.PullReqReviewerTypeRequested:
		return ReviewerAddedReasonRequested
	case enum.PullReqReviewerTypeAssigned,
		enum.PullReqReviewerTypeSelfAssigned:
		return ReviewerAddedReason("")
	default:
		return ReviewerAddedReason("")
	}
}

type ReviewerAddedPayload struct {
	Base     *BasePullReqPayload
	Reviewer *types.PrincipalInfo
	// Reason is how the reviewer was added, incl. via-group; empty = generic.
	Reason ReviewerAddedReason
}

func (s *Service) notifyReviewerAdded(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewerAddedPayload],
) error {
	payload, recipients, err := s.processReviewerAddedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.ReviewerAddedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if len(recipients) == 0 {
		return nil
	}

	err = s.notificationClient.SendReviewerAdded(ctx, recipients, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.ReviewerAddedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	return nil
}

func (s *Service) processReviewerAddedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewerAddedPayload],
) (*ReviewerAddedPayload, []*types.PrincipalInfo, error) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	reviewerPrincipal, err := s.principalInfoCache.Get(ctx, event.Payload.ReviewerID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get reviewer from principalInfoCache: %w", err)
	}

	recipients := reviewerAddedRecipients(base.Author, reviewerPrincipal, event.Payload.PrincipalID)

	return &ReviewerAddedPayload{
		Base:     base,
		Reviewer: reviewerPrincipal,
	}, recipients, nil
}

// reviewerAddedRecipients returns who to mail: the author, plus the reviewer
// unless they added themselves. The actor is never mailed about their own action.
func reviewerAddedRecipients(
	author *types.PrincipalInfo,
	reviewer *types.PrincipalInfo,
	actorID int64,
) []*types.PrincipalInfo {
	selfAdded := reviewer.ID == actorID

	var recipients []*types.PrincipalInfo

	// Mail the author, unless they added themselves (nothing to tell them).
	if !selfAdded || author.ID != reviewer.ID {
		recipients = append(recipients, author)
	}

	// Mail the reviewer, unless they added themselves or are the author (already mailed above).
	if !selfAdded && reviewer.ID != author.ID {
		recipients = append(recipients, reviewer)
	}

	return recipients
}
