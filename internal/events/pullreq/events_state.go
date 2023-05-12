// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"

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

func (r *Reader) RegisterCreated(fn events.HandlerFunc[*CreatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn, opts...)
}

const ClosedEvent events.EventType = "closed"

type ClosedPayload struct {
	Base
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

func (r *Reader) RegisterClosed(fn events.HandlerFunc[*ClosedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, ClosedEvent, fn, opts...)
}

const ReopenedEvent events.EventType = "reopened"

type ReopenedPayload struct {
	Base
	SourceSHA string `json:"source_sha"`
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

func (r *Reader) RegisterReopened(fn events.HandlerFunc[*ReopenedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, ReopenedEvent, fn, opts...)
}

const MergedEvent events.EventType = "merged"

type MergedPayload struct {
	Base
	MergeMethod gitrpcenum.MergeMethod `json:"merge_method"`
	MergeSHA    string                 `json:"merge_sha"`
	TargetSHA   string                 `json:"target_sha"`
	SourceSHA   string                 `json:"source_sha"`
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

func (r *Reader) RegisterMerged(fn events.HandlerFunc[*MergedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, MergedEvent, fn, opts...)
}
