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

const CreatedEvent events.EventType = "created"

type CreatedPayload struct {
	PipelineID int64 `json:"pipeline_id"`
	RepoID     int64 `json:"repo_id"`
}

func (r *Reporter) Created(ctx context.Context, payload *CreatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pipeline created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pipeline created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(fn events.HandlerFunc[*CreatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}
