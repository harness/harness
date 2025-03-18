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

const (
	GitspaceDeleteEvent events.EventType = "gitspace_delete_event"
)

type (
	GitspaceDeleteEventPayload struct {
		GitspaceConfigIdentifier string `json:"gitspace_config_identifier"`
		SpaceID                  int64  `json:"space_id"`
	}
)

func (r *Reporter) EmitGitspaceDeleteEvent(
	ctx context.Context,
	event events.EventType,
	payload *GitspaceDeleteEventPayload,
) {
	if payload == nil {
		return
	}

	if event != GitspaceDeleteEvent {
		log.Ctx(ctx).Error().Msgf("event type should be %s, got %s, aborting emission, payload: %+v",
			GitspaceDeleteEvent, event, payload)
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, event, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send %s event", event)
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported %s event with id '%s'", event, eventID)
}

func (r *Reader) RegisterGitspaceDeleteEvent(
	fn events.HandlerFunc[*GitspaceDeleteEventPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, GitspaceDeleteEvent, fn, opts...)
}
