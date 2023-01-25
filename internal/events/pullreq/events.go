// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package events

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

const TypeCreated events.EventType = "created"

type Base struct {
	PullReqID        int64  `json:"pullreq_id"`
	SourceRepoID     int64  `json:"source_repo_id"`
	TargetRepoID     int64  `json:"repo_id"`
	TargetRepoGitUID string `json:"repo_git_uid"`
	PrincipalID      int64  `json:"principal_id"`
	Number           int64  `json:"number"`
}

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

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TypeCreated, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(fn func(context.Context, *events.Event[*CreatedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, TypeCreated, fn)
}

const TypeUpdated events.EventType = "updated"

type UpdatedPayload struct {
	Base
	OldTitle       string
	OldDescription string
	NewTitle       string
	NewDescription string
}

func (r *Reporter) Updated(ctx context.Context, payload *UpdatedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TypeUpdated, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request updated event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request updated event with id '%s'", eventID)
}

func (r *Reader) RegisterUpdated(fn func(context.Context, *events.Event[*UpdatedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, TypeUpdated, fn)
}

const TypeStateChange events.EventType = "state_change"

type StateChangePayload struct {
	Base
	OldDraft  bool              `json:"old_draft"`
	OldState  enum.PullReqState `json:"old_state"`
	NewDraft  bool              `json:"new_draft"`
	NewState  enum.PullReqState `json:"new_state"`
	SourceSHA string            `json:"source_sha"` // Only set if the PR is moving from closed state to open
}

func (r *Reporter) StateChange(ctx context.Context, payload *StateChangePayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TypeStateChange, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request state change event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request state change event with id '%s'", eventID)
}

func (r *Reader) RegisterStateChange(fn func(context.Context, *events.Event[*StateChangePayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, TypeStateChange, fn)
}

const TypeMerged events.EventType = "merged"

type MergedPayload struct {
	Base
	// TODO: Add more fields
}

func (r *Reporter) Merged(ctx context.Context, payload *MergedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TypeMerged, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request merge event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request merge event with id '%s'", eventID)
}

func (r *Reader) RegisterMerged(fn func(context.Context, *events.Event[*MergedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, TypeMerged, fn)
}
