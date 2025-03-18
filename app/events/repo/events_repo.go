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

const CreatedEvent events.EventType = "created"

type CreatedPayload struct {
	Base
	Type string `json:"type"`
}

func (r *Reporter) Created(ctx context.Context, payload *CreatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send repo created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported repo created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(
	fn events.HandlerFunc[*CreatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}

const StateChangedEvent events.EventType = "state-changed"

type StateChangedPayload struct {
	Base
	OldState enum.RepoState `json:"old_state"`
	NewState enum.RepoState `json:"new_state"`
}

func (r *Reporter) StateChanged(ctx context.Context, payload *StateChangedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, StateChangedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send repo srtate change event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported repo state change event with id '%s'", eventID)
}

func (r *Reader) RegisterStateChanged(
	fn events.HandlerFunc[*StateChangedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, StateChangedEvent, fn, opts...)
}

const PublicAccessChangedEvent events.EventType = "public-access-changed"

type PublicAccessChangedPayload struct {
	Base
	OldIsPublic bool `json:"old_is_public"`
	NewIsPublic bool `json:"new_is_public"`
}

func (r *Reporter) PublicAccessChanged(ctx context.Context, payload *PublicAccessChangedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, PublicAccessChangedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send repo public access changed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported repo public access changed event with id '%s'", eventID)
}

func (r *Reader) RegisterPublicAccessChanged(
	fn events.HandlerFunc[*PublicAccessChangedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, PublicAccessChangedEvent, fn, opts...)
}

const SoftDeletedEvent events.EventType = "soft-deleted"

type SoftDeletedPayload struct {
	Base
	RepoPath string `json:"repo_path"`
	Deleted  int64  `json:"deleted"`
}

func (r *Reporter) SoftDeleted(ctx context.Context, payload *SoftDeletedPayload) {
	if payload == nil {
		return
	}
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, SoftDeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send repo soft deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported repo soft deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterSoftDeleted(
	fn events.HandlerFunc[*SoftDeletedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, SoftDeletedEvent, fn, opts...)
}

const DeletedEvent events.EventType = "deleted"

type DeletedPayload struct {
	Base
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

func (r *Reader) RegisterDeleted(
	fn events.HandlerFunc[*DeletedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, DeletedEvent, fn, opts...)
}

const DefaultBranchUpdatedEvent events.EventType = "default-branch-updated"

type DefaultBranchUpdatedPayload struct {
	Base
	OldName string `json:"old_name"`
	NewName string `json:"new_name"`
}

func (r *Reporter) DefaultBranchUpdated(ctx context.Context, payload *DefaultBranchUpdatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, DefaultBranchUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send default branch updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported default branch updated event with id '%s'", eventID)
}

func (r *Reader) RegisterDefaultBranchUpdated(
	fn events.HandlerFunc[*DefaultBranchUpdatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, DefaultBranchUpdatedEvent, fn, opts...)
}
