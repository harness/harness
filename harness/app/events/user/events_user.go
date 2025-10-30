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

const RegisteredEvent events.EventType = "registered"

type RegisteredPayload struct {
	Base
}

func (r *Reporter) Registered(ctx context.Context, payload *RegisteredPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, RegisteredEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send user registered event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported user registered event with id '%s'", eventID)
}

func (r *Reader) RegisterRegistered(
	fn events.HandlerFunc[*RegisteredPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, RegisteredEvent, fn, opts...)
}

const CreatedEvent events.EventType = "created"

type CreatedPayload struct {
	Base

	// CreatedPrincipalID is ID of the created user.
	CreatedPrincipalID int64 `json:"created_principal_id"`
}

func (r *Reporter) Created(ctx context.Context, payload *CreatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send user created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported user created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(
	fn events.HandlerFunc[*CreatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}

const LoggedInEvent events.EventType = "logged-in"

type LoggedInPayload struct {
	Base
}

func (r *Reporter) LoggedIn(ctx context.Context, payload *LoggedInPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, LoggedInEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send user logged-in event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported user logged-in event with id '%s'", eventID)
}

func (r *Reader) RegisterLoggedIn(
	fn events.HandlerFunc[*LoggedInPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, LoggedInEvent, fn, opts...)
}
