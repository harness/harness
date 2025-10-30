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

type PullReqBranchUpdatedPayload struct {
	Base      *BasePullReqPayload
	Committer *types.PrincipalInfo
	NewSHA    string
}

func (s *Service) notifyPullReqBranchUpdated(
	ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	payload, reviewers, err := s.processPullReqBranchUpdatedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.BranchUpdatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if len(reviewers) == 0 {
		return nil
	}

	err = s.notificationClient.SendPullReqBranchUpdated(ctx, reviewers, payload)
	if err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.BranchUpdatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	return nil
}

func (s *Service) processPullReqBranchUpdatedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) (*PullReqBranchUpdatedPayload, []*types.PrincipalInfo, error) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	committer, err := s.principalInfoCache.Get(ctx, event.Payload.PrincipalID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get principal info for %d: %w", event.Payload.PrincipalID, err)
	}

	reviewers, err := s.pullReqReviewersStore.List(ctx, event.Payload.PullReqID)
	if err != nil {
		return nil, nil,
			fmt.Errorf("failed to get reviewers for pull request %d: %w", event.Payload.PullReqID, err)
	}

	reviewerPrincipals := make([]*types.PrincipalInfo, len(reviewers))
	for i, reviewer := range reviewers {
		reviewerPrincipals[i], err = s.principalInfoCache.Get(ctx, reviewer.PrincipalID)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get principal info for %d: %w", reviewer.PrincipalID, err)
		}
	}

	return &PullReqBranchUpdatedPayload{
		Base:      base,
		NewSHA:    event.Payload.NewSHA,
		Committer: committer,
	}, reviewerPrincipals, nil
}
