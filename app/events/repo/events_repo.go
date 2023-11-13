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

const DeletedEvent events.EventType = "deleted"

type DeletedPayload struct {
	RepoID int64 `json:"repo_id"`
}

func (r *Reporter) Deleted(ctx context.Context, payload *DeletedPayload) {
	if payload == nil {
		return
	}
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, DeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send repo deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported repo deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterRepoDeleted(fn events.HandlerFunc[*DeletedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, DeletedEvent, fn, opts...)
}
