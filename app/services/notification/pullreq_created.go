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

	"golang.org/x/exp/maps"
)

func (s *Service) notifyPullReqCreated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return fmt.Errorf("failed to get base payload: %w", err)
	}

	reviewers, err := s.principalInfoCache.Map(ctx, event.Payload.ReviewerIDs)
	if err != nil {
		return fmt.Errorf("failed to get principal infos from cache: %w", err)
	}

	// Filter out the author from reviewers
	delete(reviewers, base.Author.ID)

	if len(reviewers) == 0 {
		return nil
	}

	// Send ONE batched email to the author listing all reviewers
	reviewerList := maps.Values(reviewers)
	reviewerNames := make([]string, 0, len(reviewerList))
	for _, reviewer := range reviewerList {
		reviewerNames = append(reviewerNames, reviewer.DisplayName)
	}

	authorPayload := &ReviewersAddedPayload{
		Base:          base,
		ReviewerCount: len(reviewers),
		ReviewerNames: strings.Join(reviewerNames, ", "),
	}

	err = s.notificationClient.SendReviewersAdded(ctx, []*types.PrincipalInfo{base.Author}, authorPayload)
	if err != nil {
		return fmt.Errorf(
			"failed to send batched email to author for event %s for pullReqID %d: %w",
			pullreqevents.CreatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	// Send individual emails to each reviewer
	for _, reviewer := range reviewers {
		payload := &ReviewerAddedPayload{
			Base:     base,
			Reviewer: reviewer,
		}
		if err := s.notificationClient.SendReviewerAdded(
			ctx,
			[]*types.PrincipalInfo{reviewer},
			payload,
		); err != nil {
			return fmt.Errorf(
				"failed to send email to reviewer for event %s for pullReqID %d: %w",
				pullreqevents.CreatedEvent,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	return nil
}
