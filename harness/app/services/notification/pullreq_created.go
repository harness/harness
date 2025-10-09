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

	for _, reviewer := range reviewers {
		payload := &ReviewerAddedPayload{
			Base:     base,
			Reviewer: reviewer,
		}
		if err := s.notificationClient.SendReviewerAdded(
			ctx,
			[]*types.PrincipalInfo{base.Author, reviewer},
			payload,
		); err != nil {
			return fmt.Errorf(
				"failed to send email for event %s for pullReqID %d: %w",
				pullreqevents.CreatedEvent,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	return nil
}
