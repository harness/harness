// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const branchDeletedEvent events.EventType = "branchdeleted"

type BranchDeletedPayload struct {
	RepoUID    string `json:"repo_uid"`
	BranchName string `json:"branch_name"`
}

func (r *Reporter) BranchDeleted(ctx context.Context, payload *BranchDeletedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, branchDeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send branch deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported branch deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchDeleted(fn func(context.Context, *events.Event[*BranchDeletedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, branchDeletedEvent, fn)
}
