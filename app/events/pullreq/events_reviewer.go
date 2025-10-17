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
	ReviewerAddedEvent     events.EventType = "reviewer-added"
	UserGroupReviewerAdded events.EventType = "usergroup-reviewer-added"
)

type ReviewerAddedPayload struct {
	Base
	ReviewerID int64 `json:"reviewer_id"`
}

type UserGroupReviewerAddedPayload struct {
	Base
	UserGroupReviewerID int64 `json:"usergroup_reviewer_id"`
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
	// TODO: Start using this for sending out notifications
	return events.ReaderRegisterEvent(r.innerReader, UserGroupReviewerAdded, fn, opts...)
}
