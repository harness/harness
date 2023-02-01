// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

const BranchCreatedEvent events.EventType = "branch-created"

type BranchCreatedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	SHA         string `json:"sha"`
}

func (r *Reporter) BranchCreated(ctx context.Context, payload *BranchCreatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchCreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send branch created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported branch created event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchCreated(fn events.HandlerFunc[*BranchCreatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchCreatedEvent, fn, opts...)
}

const BranchUpdatedEvent events.EventType = "branch-updated"

type BranchUpdatedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	OldSHA      string `json:"old_sha"`
	NewSHA      string `json:"new_sha"`
	Forced      bool   `json:"forced"`
}

func (r *Reporter) BranchUpdated(ctx context.Context, payload *BranchUpdatedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchUpdatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send branch updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported branch updated event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchUpdated(fn events.HandlerFunc[*BranchUpdatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchUpdatedEvent, fn, opts...)
}

const BranchDeletedEvent events.EventType = "branch-deleted"

type BranchDeletedPayload struct {
	RepoID      int64  `json:"repo_id"`
	PrincipalID int64  `json:"principal_id"`
	Ref         string `json:"ref"`
	SHA         string `json:"sha"`
}

func (r *Reporter) BranchDeleted(ctx context.Context, payload *BranchDeletedPayload) {
	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchDeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send branch deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported branch deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchDeleted(fn events.HandlerFunc[*BranchDeletedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchDeletedEvent, fn, opts...)
}
