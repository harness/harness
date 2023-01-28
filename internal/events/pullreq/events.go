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

const CreatedEvent events.EventType = "created"

type Base struct {
	PullReqID    int64 `json:"pullreq_id"`
	SourceRepoID int64 `json:"source_repo_id"`
	TargetRepoID int64 `json:"repo_id"`
	PrincipalID  int64 `json:"principal_id"`
	Number       int64 `json:"number"`
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

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, CreatedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request created event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request created event with id '%s'", eventID)
}

func (r *Reader) RegisterCreated(fn func(context.Context, *events.Event[*CreatedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, CreatedEvent, fn)
}

const TitleChangedEvent events.EventType = "title-changed"

type TitleChangedPayload struct {
	Base
	OldTitle string `json:"old_title"`
	NewTitle string `json:"new_title"`
}

func (r *Reporter) TitleChanged(ctx context.Context, payload *TitleChangedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, TitleChangedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request title changed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request title changed event with id '%s'", eventID)
}

func (r *Reader) RegisterTitleChanged(fn func(context.Context, *events.Event[*TitleChangedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, TitleChangedEvent, fn)
}

const BranchUpdatedEvent events.EventType = "branch-updated"

type BranchUpdatedPayload struct {
	Base
	OldSHA string `json:"old_sha"`
	NewSHA string `json:"new_sha"`
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

func (r *Reader) RegisterBranchUpdated(fn func(context.Context, *events.Event[*BranchUpdatedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchUpdatedEvent, fn)
}

const StateChangedEvent events.EventType = "state-changed"

type StateChangedPayload struct {
	Base
	OldDraft  bool              `json:"old_draft"`
	OldState  enum.PullReqState `json:"old_state"`
	NewDraft  bool              `json:"new_draft"`
	NewState  enum.PullReqState `json:"new_state"`
	SourceSHA string            `json:"source_sha"` // Only set if the PR is moving from closed state to open
	Message   string            `json:"message"`
}

func (r *Reporter) StateChanged(ctx context.Context, payload *StateChangedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, StateChangedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request state changed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request state changed event with id '%s'", eventID)
}

func (r *Reader) RegisterStateChanged(fn func(context.Context, *events.Event[*StateChangedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, StateChangedEvent, fn)
}

const BranchDeletedEvent events.EventType = "branch-deleted"

type BranchDeletedPayload struct {
	Base
	SHA string `json:"sha"`
}

func (r *Reporter) BranchDeleted(ctx context.Context, payload *BranchDeletedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, BranchDeletedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request branch deleted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request branch deleted event with id '%s'", eventID)
}

func (r *Reader) RegisterBranchDeleted(fn func(context.Context, *events.Event[*BranchDeletedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchDeletedEvent, fn)
}

const ReviewSubmittedEvent events.EventType = "review-submitted"

type ReviewSubmittedPayload struct {
	Base
	Message  string                     `json:"message"`
	Decision enum.PullReqReviewDecision `json:"decision"`
}

func (r *Reporter) ReviewSubmitted(ctx context.Context, payload *ReviewSubmittedPayload) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ReviewSubmittedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request review submitted event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request review submitted with id '%s'", eventID)
}

func (r *Reader) RegisterReviewSubmitted(fn func(context.Context, *events.Event[*ReviewSubmittedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, ReviewSubmittedEvent, fn)
}

const MergedEvent events.EventType = "merged"

type MergedPayload struct {
	Base
	MergeMethod enum.MergeMethod `json:"merge_method"`
	SHA         string           `json:"sha"`
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

func (r *Reader) RegisterMerged(fn func(context.Context, *events.Event[*MergedPayload]) error) error {
	return events.ReaderRegisterEvent(r.innerReader, MergedEvent, fn)
}
