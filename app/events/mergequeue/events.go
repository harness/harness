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

type Base struct {
	RepoID int64  `json:"repo_id"`
	Branch string `json:"branch"`
}

const UpdatedEvent events.EventType = "updated"

type UpdatedPayload struct {
	Base
}

func (r *Reporter) Updated(ctx context.Context, payload *UpdatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, UpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to send merge queue updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported merge queue updated event with id '%s'", eventID)
}

func (r *Reader) RegisterUpdated(
	fn events.HandlerFunc[*UpdatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, UpdatedEvent, fn, opts...)
}

const ChecksRequestedEvent events.EventType = "checks_requested"

type ChecksRequestedPayload struct {
	Base
	PrincipalID int64  `json:"principal_id"`
	CommitSHA   string `json:"commit_sha"`
}

func (r *Reporter) ChecksRequested(ctx context.Context, payload *ChecksRequestedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ChecksRequestedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to send merge queue checks requested event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported merge queue checks requested event with id '%s'", eventID)
}

func (r *Reader) RegisterChecksRequested(
	fn events.HandlerFunc[*ChecksRequestedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ChecksRequestedEvent, fn, opts...)
}

const ChecksCanceledEvent events.EventType = "checks_canceled"

type ChecksCanceledPayload struct {
	Base
	PrincipalID int64  `json:"principal_id"`
	CommitSHA   string `json:"commit_sha"`
}

func (r *Reporter) ChecksCanceled(ctx context.Context, payload *ChecksCanceledPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ChecksCanceledEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to send merge queue checks canceled event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported merge queue checks canceled event with id '%s'", eventID)
}

func (r *Reader) RegisterChecksCanceled(
	fn events.HandlerFunc[*ChecksCanceledPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ChecksCanceledEvent, fn, opts...)
}
