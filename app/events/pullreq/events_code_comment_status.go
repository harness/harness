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

package events

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const CommentStatusUpdatedEvent events.EventType = "comment-status-updated"

type CommentStatusUpdatedPayload struct {
	Base
	ActivityID    int64                     `json:"activity_id"`
	OldStatus     enum.PullReqCommentStatus `json:"old_status"`
	NewStatus     enum.PullReqCommentStatus `json:"new_status"`
	OldResolvedBy *int64                    `json:"old_resolved_by"`
	NewResolvedBy *int64                    `json:"new_resolved_by"`
}

func (r *Reporter) CommentStatusUpdated(
	ctx context.Context,
	payload *CommentStatusUpdatedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CommentStatusUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf(
			"failed to send pull request comment status updated event for event id '%s'", eventID)
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request comment status update event with id '%s'", eventID)
}

func (r *Reader) RegisterCommentStatusUpdated(
	fn events.HandlerFunc[*CommentStatusUpdatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CommentStatusUpdatedEvent, fn, opts...)
}
