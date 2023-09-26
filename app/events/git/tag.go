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

const TagCreatedEvent events.EventType = "tag-created"

type TagCreatedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	SHA         string `json:"sha"`
}

func (r *Reporter) TagCreated(ctx context.Context, payload *TagCreatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TagCreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send tag created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported tag created event with id '%s'", eventID)
}

func (r *Reader) RegisterTagCreated(fn events.HandlerFunc[*TagCreatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, TagCreatedEvent, fn, opts...)
}

const TagUpdatedEvent events.EventType = "tag-updated"

type TagUpdatedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	OldSHA      string `json:"old_sha"`
	NewSHA      string `json:"new_sha"`
	Forced      bool   `json:"forced"`
}

func (r *Reporter) TagUpdated(ctx context.Context, payload *TagUpdatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TagUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send tag updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported tag updated event with id '%s'", eventID)
}

func (r *Reader) RegisterTagUpdated(fn events.HandlerFunc[*TagUpdatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, TagUpdatedEvent, fn, opts...)
}

const TagDeletedEvent events.EventType = "tag-deleted"

type TagDeletedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	SHA         string `json:"sha"`
}

func (r *Reporter) TagDeleted(ctx context.Context, payload *TagDeletedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TagDeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send tag deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported tag deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterTagDeleted(fn events.HandlerFunc[*TagDeletedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, TagDeletedEvent, fn, opts...)
}
