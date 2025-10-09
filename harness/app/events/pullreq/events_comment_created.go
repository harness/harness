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

	"github.com/rs/zerolog/log"
)

const CommentCreatedEvent events.EventType = "comment-created"

type CommentCreatedPayload struct {
	Base
	ActivityID int64  `json:"activity_id"`
	SourceSHA  string `json:"source_sha"`
	IsReply    bool   `json:"is_reply"`
}

func (r *Reporter) CommentCreated(
	ctx context.Context,
	payload *CommentCreatedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CommentCreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request comment created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request comment created event with id '%s'", eventID)
}

func (r *Reader) RegisterCommentCreated(
	fn events.HandlerFunc[*CommentCreatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CommentCreatedEvent, fn, opts...)
}
