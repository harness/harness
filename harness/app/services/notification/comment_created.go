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
	"strconv"
	"strings"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	gitnessenum "github.com/harness/gitness/types/enum"

	"golang.org/x/exp/maps"
)

type CommentPayload struct {
	Base      *BasePullReqPayload
	Commenter *types.PrincipalInfo
	Text      string
}

func (s *Service) notifyCommentCreated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentCreatedPayload],
) error {
	payload, mentions, participants, author, err := s.processCommentCreatedEvent(ctx, event)
	if err != nil {
		return fmt.Errorf(
			"failed to process %s event for pullReqID %d: %w",
			pullreqevents.CommentCreatedEvent,
			event.Payload.PullReqID,
			err,
		)
	}

	if len(mentions) > 0 {
		err = s.notificationClient.SendCommentMentions(ctx, mentions, payload)
		if err != nil {
			return fmt.Errorf(
				"failed to send notification to mentions for event %s for pullReqID %d: %w",
				pullreqevents.CommentCreatedEvent,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	if len(participants) > 0 {
		err = s.notificationClient.SendCommentParticipants(ctx, participants, payload)
		if err != nil {
			return fmt.Errorf(
				"failed to send notification to participants for event %s for pullReqID %d: %w",
				pullreqevents.CommentCreatedEvent,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	if author != nil {
		err = s.notificationClient.SendCommentPRAuthor(
			ctx,
			[]*types.PrincipalInfo{author},
			payload,
		)
		if err != nil {
			return fmt.Errorf(
				"failed to send notification to author for event %s for pullReqID %d: %w",
				pullreqevents.CommentCreatedEvent,
				event.Payload.PullReqID,
				err,
			)
		}
	}

	return nil
}

func (s *Service) processCommentCreatedEvent(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentCreatedPayload],
) (
	payload *CommentPayload,
	mentions []*types.PrincipalInfo,
	participants []*types.PrincipalInfo,
	author *types.PrincipalInfo,
	err error,
) {
	base, err := s.getBasePayload(ctx, event.Payload.Base)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to get base payload: %w", err)
	}

	activity, err := s.pullReqActivityStore.Find(ctx, event.Payload.ActivityID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch activity from pullReqActivityStore: %w", err)
	}

	if activity.Type != gitnessenum.PullReqActivityTypeComment {
		return nil, nil, nil, nil, fmt.Errorf("code-comments are not supported currently")
	}

	commenter, err := s.principalInfoView.Find(ctx, activity.CreatedBy)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch commenter from principalInfoView: %w", err)
	}

	payload = &CommentPayload{
		Base:      base,
		Commenter: commenter,
		Text:      activity.Text,
	}

	seen := make(map[int64]bool)
	seen[commenter.ID] = true

	// process mentions
	mentionsMap, err := s.processMentions(ctx, activity.Metadata, seen)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	for i, mention := range mentionsMap {
		payload.Text = strings.ReplaceAll(
			payload.Text, "@["+strconv.FormatInt(i, 10)+"]", mention.DisplayName,
		)
	}

	// process participants
	participants, err = s.processParticipants(
		ctx, event.Payload.IsReply, seen, event.Payload.PullReqID, activity.Order)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	// process author
	if !seen[base.Author.ID] {
		author = base.Author
	}

	return payload, maps.Values(mentionsMap), participants, author, nil
}

func (s *Service) processMentions(
	ctx context.Context,
	metadata *types.PullReqActivityMetadata,
	seen map[int64]bool,
) (map[int64]*types.PrincipalInfo, error) {
	if metadata == nil || metadata.Mentions == nil {
		return map[int64]*types.PrincipalInfo{}, nil
	}

	var ids []int64
	for _, id := range metadata.Mentions.IDs {
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) == 0 {
		return map[int64]*types.PrincipalInfo{}, nil
	}

	mentions, err := s.principalInfoCache.Map(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch thread mentions from principalInfoView: %w", err)
	}

	return mentions, nil
}

func (s *Service) processParticipants(
	ctx context.Context,
	isReply bool,
	seen map[int64]bool,
	prID int64,
	order int64,
) ([]*types.PrincipalInfo, error) {
	var participants []*types.PrincipalInfo

	if !isReply {
		return participants, nil
	}

	authorIDs, err := s.pullReqActivityStore.ListAuthorIDs(
		ctx,
		prID,
		order,
	)
	if err != nil {
		return participants, fmt.Errorf("failed to fetch thread participant IDs from pullReqActivityStore: %w", err)
	}

	var participantIDs []int64
	for _, authorID := range authorIDs {
		if !seen[authorID] {
			participantIDs = append(participantIDs, authorID)
			seen[authorID] = true
		}
	}
	if len(participantIDs) > 0 {
		participants, err = s.principalInfoView.FindMany(ctx, participantIDs)
		if err != nil {
			return participants, fmt.Errorf("failed to fetch thread participants from principalInfoView: %w", err)
		}
	}

	return participants, nil
}
