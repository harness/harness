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

const LabelAssignedEvent events.EventType = "label-assigned"

type LabelAssignedPayload struct {
	Base
	LabelID int64  `json:"label_id"`
	ValueID *int64 `json:"value_id"`
}

func (r *Reporter) LabelAssigned(
	ctx context.Context,
	payload *LabelAssignedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, LabelAssignedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request label assigned event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request label assigned event with id '%s'", eventID)
}

func (r *Reader) RegisterLabelAssigned(
	fn events.HandlerFunc[*LabelAssignedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, LabelAssignedEvent, fn, opts...)
}
