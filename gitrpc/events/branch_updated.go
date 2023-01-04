// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const BranchUpdatedEvent events.EventType = "branchupdated"

type BranchUpdatedPayload struct {
	RepoUID    string `json:"repo_uid"`
	BranchName string `json:"branch_name"`
	FullRef    string `json:"full_ref"`
	OldSHA     string `json:"old_sha"`
	NewSHA     string `json:"new_sha"`
	Forced     bool   `json:"forced"`
}

func (r *Reporter) BranchUpdated(ctx context.Context, payload *BranchUpdatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send branch updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported branch updated event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchUpdated(fn func(context.Context, *events.Event[*BranchUpdatedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchUpdatedEvent, fn)
}
