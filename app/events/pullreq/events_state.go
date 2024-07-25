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
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	SourceSHA    string `json:"source_sha"`
}

func (r *Reporter) Created(ctx context.Context, payload *CreatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(
	fn events.HandlerFunc[*CreatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}

const ClosedEvent events.EventType = "closed"

type ClosedPayload struct {
	Base
	SourceSHA string `json:"source_sha"`
}

func (r *Reporter) Closed(ctx context.Context, payload *ClosedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ClosedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request closed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request closed event with id '%s'", eventID)
}

func (r *Reader) RegisterClosed(
	fn events.HandlerFunc[*ClosedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ClosedEvent, fn, opts...)
}

const ReopenedEvent events.EventType = "reopened"

type ReopenedPayload struct {
	Base
	SourceSHA    string `json:"source_sha"`
	MergeBaseSHA string `json:"merge_base_sha"`
}

func (r *Reporter) Reopened(ctx context.Context, payload *ReopenedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ReopenedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request reopened event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request reopened event with id '%s'", eventID)
}

func (r *Reader) RegisterReopened(
	fn events.HandlerFunc[*ReopenedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ReopenedEvent, fn, opts...)
}

const MergedEvent events.EventType = "merged"

type MergedPayload struct {
	Base
	MergeMethod enum.MergeMethod `json:"merge_method"`
	MergeSHA    string           `json:"merge_sha"`
	TargetSHA   string           `json:"target_sha"`
	SourceSHA   string           `json:"source_sha"`
}

func (r *Reporter) Merged(ctx context.Context, payload *MergedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, MergedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request merged event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request merged event with id '%s'", eventID)
}

func (r *Reader) RegisterMerged(
	fn events.HandlerFunc[*MergedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, MergedEvent, fn, opts...)
}

const UpdatedEvent events.EventType = "updated"

type UpdatedPayload struct {
	Base
	TitleChanged       bool   `json:"title_changed"`
	TitleOld           string `json:"title_old"`
	TitleNew           string `json:"title_new"`
	DescriptionChanged bool   `json:"description_changed"`
	DescriptionOld     string `json:"description_old"`
	DescriptionNew     string `json:"description_new"`
}

func (r *Reporter) Updated(ctx context.Context, payload *UpdatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, UpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request created event with id '%s'", eventID)
}

func (r *Reader) RegisterUpdated(
	fn events.HandlerFunc[*UpdatedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, UpdatedEvent, fn, opts...)
}
