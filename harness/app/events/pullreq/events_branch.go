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

const BranchUpdatedEvent events.EventType = "branch-updated"

type BranchUpdatedPayload struct {
	Base
	OldSHA          string `json:"old_sha"`
	NewSHA          string `json:"new_sha"`
	OldMergeBaseSHA string `json:"old_merge_base_sha"`
	NewMergeBaseSHA string `json:"new_merge_base_sha"`
	Forced          bool   `json:"forced"`
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

func (r *Reader) RegisterBranchUpdated(
	fn events.HandlerFunc[*BranchUpdatedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, BranchUpdatedEvent, fn, opts...)
}

const TargetBranchChangedEvent events.EventType = "target-branch-changed"

type TargetBranchChangedPayload struct {
	Base
	SourceSHA       string `json:"source_sha"`
	OldTargetBranch string `json:"old_target_branch"`
	NewTargetBranch string `json:"new_target_branch"`
	OldMergeBaseSHA string `json:"old_merge_base_sha"`
	NewMergeBaseSHA string `json:"new_merge_base_sha"`
}

func (r *Reporter) TargetBranchChanged(
	ctx context.Context,
	payload *TargetBranchChangedPayload,
) {
	if payload == nil {
		return
	}

	eventID, err := events.ReporterSendEvent(
		r.innerReporter, ctx, TargetBranchChangedEvent, payload,
	)
	if err != nil {
		log.Ctx(ctx).Err(err).Msgf("failed to send pull request target branch changed event")
		return
	}

	log.Ctx(ctx).Debug().Msgf(
		"reported pull request target branch changed event with id '%s'", eventID,
	)
}

func (r *Reader) RegisterTargetBranchChanged(
	fn events.HandlerFunc[*TargetBranchChangedPayload],
	opts ...events.HandlerOption) error {
	return events.ReaderRegisterEvent(r.innerReader, TargetBranchChangedEvent, fn, opts...)
}
