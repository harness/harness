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

// ReportedEvent is the event type for check status reporting.
const ReportedEvent events.EventType = "reported"

type ReportedPayload struct {
	Base
	Identifier string           `json:"identifier"`
	Status     enum.CheckStatus `json:"status"`
}

func (r *Reporter) Reported(ctx context.Context, payload *ReportedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ReportedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send check status event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported check status event with id '%s'", eventID)
}

func (r *Reader) RegisterReported(
	fn events.HandlerFunc[*ReportedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ReportedEvent, fn, opts...)
}
