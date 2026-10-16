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

const (
	ReviewerAddedEvent       events.EventType = "reviewer-added"
	ReviewerRemovedEvent     events.EventType = "reviewer-removed"
	UserGroupReviewerAdded   events.EventType = "usergroup-reviewer-added"
	UserGroupReviewerRemoved events.EventType = "usergroup-reviewer-removed"
)

type ReviewerAddedPayload struct {
	Base
	ReviewerID int64 `json:"reviewer_id"`
}

type ReviewerRemovedPayload struct {
	Base
	ReviewerID int64 `json:"reviewer_id"`
}

type UserGroupReviewerAddedPayload struct {
	Base
	UserGroupReviewerIDs []int64 `json:"usergroup_reviewer_ids,omitempty"`
	ReviewerIDs          []int64 `json:"reviewer_ids,omitempty"`
}

type UserGroupReviewerRemovedPayload struct {
	Base
	UserGroupID int64 `json:"usergroup_id"`
}

func (r *Reporter) ReviewerAdded(
	ctx context.Context,
	payload *ReviewerAddedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ReviewerAddedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request reviewer added event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request reviewer added event with id '%s'", eventID)
}

func (r *Reader) RegisterReviewerAdded(
	fn events.HandlerFunc[*ReviewerAddedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ReviewerAddedEvent, fn, opts...)
}

func (r *Reporter) ReviewerRemoved(
	ctx context.Context,
	payload *ReviewerRemovedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, ReviewerRemovedEvent, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request reviewer removed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request reviewer removed event with id '%s'", eventID)
}

func (r *Reader) RegisterReviewerRemoved(
	fn events.HandlerFunc[*ReviewerRemovedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, ReviewerRemovedEvent, fn, opts...)
}

func (r *Reporter) UserGroupReviewerAdded(
	ctx context.Context,
	payload *UserGroupReviewerAddedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, UserGroupReviewerAdded, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request reviewer added event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request reviewer added event with id '%s'", eventID)
}

func (r *Reader) RegisterUserGroupReviewerAdded(
	fn events.HandlerFunc[*UserGroupReviewerAddedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, UserGroupReviewerAdded, fn, opts...)
}

func (r *Reporter) UserGroupReviewerRemoved(
	ctx context.Context,
	payload *UserGroupReviewerRemovedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(r.innerReporter, ctx, UserGroupReviewerRemoved, payload)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request user group reviewer removed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf("reported pull request user group reviewer removed event with id '%s'", eventID)
}

func (r *Reader) RegisterUserGroupReviewerRemoved(
	fn events.HandlerFunc[*UserGroupReviewerRemovedPayload],
	opts ...events.HandlerOption,
) error {
	return events.ReaderRegisterEvent(r.innerReader, UserGroupReviewerRemoved, fn, opts...)
}
