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

type PullReqState string

const (
	PullReqStateMerged   PullReqState = "merged"
	PullReqStateClosed   PullReqState = "closed"
	PullReqStateReopened PullReqState = "reopened"
)

type PullReqStateChangedPayload struct {
	Base      *BasePullReqPayload
	ChangedBy *types.PrincipalInfo
	State     PullReqState
}

func (s *Service) notifyPullReqStateMerged(
	ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	payload, recipients, err := s.processPullReqStateChangedEvent(ctx, event.Payload.Base, PullReqStateMerged)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.MergedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if err = s.notificationClient.SendPullReqStateChanged(
		ctx,
		recipients,
		payload,
	); err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.MergedEvent,
			payload.Base.PullReq.ID,
			err,
		)
	}
	return nil
}

func (s *Service) notifyPullReqStateClosed(
	ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	payload, recipients, err := s.processPullReqStateChangedEvent(ctx, event.Payload.Base, PullReqStateClosed)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.ClosedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if err = s.notificationClient.SendPullReqStateChanged(
		ctx,
		recipients,
		payload,
	); err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.ClosedEvent,
			payload.Base.PullReq.ID,
			err,
		)
	}
	return nil
}

func (s *Service) notifyPullReqStateReOpened(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	payload, recipients, err := s.processPullReqStateChangedEvent(ctx, event.Payload.Base, PullReqStateReopened)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.ReopenedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if err = s.notificationClient.SendPullReqStateChanged(
		ctx,
		recipients,
		payload,
	); err != nil {
		return fmt.Errorf(
			"failed to send email for event %s for pullReqID %d: %w",
			pullreqevents.ReopenedEvent,
			payload.Base.PullReq.ID,
			err,
		)
	}
	return nil
}

func (s *Service) processPullReqStateChangedEvent(
	ctx context.Context,
	baseEvent pullreqevents.Base,
	state PullReqState,
) (*PullReqStateChangedPayload, []*types.PrincipalInfo, error) {
	basePayload, err := s.getBasePayload(ctx, baseEvent)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	author, err := s.principalInfoCache.Get(ctx, basePayload.PullReq.CreatedBy)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to get author from principalInfoCache for pullReqID %d: %w",
			baseEvent.PullReqID,
			err,
		)
	}

	stateModifierPrincipal, err := s.principalInfoCache.Get(ctx, baseEvent.PrincipalID)
	if err != nil {
		return nil, nil,
			fmt.Errorf(
				"failed to get principal information about principal that changed PR state for pullReqID %d: %w",
				baseEvent.PullReqID,
				err,
			)
	}

	reviewers, err := s.pullReqReviewersStore.List(ctx, baseEvent.PullReqID)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to get reviewers from pullReqReviewersStore for pullReqID %d: %w",
			baseEvent.PullReqID,
			err,
		)
	}

	userGroupReviewers, err := s.userGroupReviewerStore.List(ctx, baseEvent.PullReqID)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"failed to get user group reviewers from userGroupReviewerStore for pullReqID %d: %w",
			baseEvent.PullReqID,
			err,
		)
	}

	seen := make(map[int64]struct{}, len(reviewers)+1)
	recipients := make([]*types.PrincipalInfo, 0, len(reviewers)+1)

	addRecipient := func(p *types.PrincipalInfo) {
		if _, ok := seen[p.ID]; !ok {
			seen[p.ID] = struct{}{}
			recipients = append(recipients, p)
		}
	}

	for i := range reviewers {
		addRecipient(&reviewers[i].Reviewer)
	}

	if len(userGroupReviewers) > 0 {
		groupIDs := make([]int64, len(userGroupReviewers))
		for i, ugr := range userGroupReviewers {
			groupIDs[i] = ugr.UserGroupID
		}

		memberIDs, err := s.userGroupService.ListUserIDsByGroupIDs(ctx, groupIDs)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to list user group member IDs for pullReqID %d: %w",
				baseEvent.PullReqID,
				err,
			)
		}

		memberInfos, err := s.principalInfoCache.Map(ctx, memberIDs)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"failed to load principal infos for user group members for pullReqID %d: %w",
				baseEvent.PullReqID,
				err,
			)
		}

		for _, info := range memberInfos {
			addRecipient(info)
		}
	}

	addRecipient(author)

	return &PullReqStateChangedPayload{
		Base:      basePayload,
		ChangedBy: stateModifierPrincipal,
		State:     state,
	}, recipients, nil
}
