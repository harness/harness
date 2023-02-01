// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const BranchUpdatedEvent events.EventType = "branch-updated"

type BranchUpdatedPayload struct {
	Base
	OldSHA string `json:"old_sha"`
	NewSHA string `json:"new_sha"`
	Forced bool   `json:"forced"`
}

func (r *Reporter) BranchUpdated(ctx context.Context, payload *BranchUpdatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request branch updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request branch updated event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchUpdated(fn events.HandlerFunc[*BranchUpdatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchUpdatedEvent, fn, opts...)
}
