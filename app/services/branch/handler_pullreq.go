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

package branch

import (
	"context"
	"fmt"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

// handleEventPullReqCreated handles the pull request created event by updating the source branch
// with the ID of the newly created pull request.
func (s *Service) handleEventPullReqCreated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	payload := event.Payload
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	sourceRepoID := payload.SourceRepoID
	sourceBranch := payload.SourceBranch
	pullReqID := payload.PullReqID

	if sourceRepoID == nil {
		return events.NewDiscardEventErrorf("pullreq %d event missing sourceRepoID", pullReqID)
	}

	logger := log.Ctx(ctx).With().
		Int64("source_repo_id", *sourceRepoID).
		Str("source_branch", sourceBranch).
		Int64("pullreq_id", pullReqID).
		Logger()

	err := s.branchStore.UpdateLastPR(ctx, *sourceRepoID, sourceBranch, &pullReqID)
	if err != nil {
		return fmt.Errorf("failed to update last PR: %w", err)
	}

	logger.Debug().Msg("successfully updated branch's last created pullreq")
	return nil
}

// handleEventPullReqClosed handles the pull request closed event.
func (s *Service) handleEventPullReqClosed(
	ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	payload := event.Payload
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	if payload.SourceRepoID == nil {
		return events.NewDiscardEventErrorf("pullreq %d event missing sourceRepoID", payload.PullReqID)
	}

	logger := log.Ctx(ctx).With().
		Int64("source_repo_id", *payload.SourceRepoID).
		Int64("pullreq_id", payload.PullReqID).
		Str("source_branch", payload.SourceBranch).
		Logger()

	sourceRepoID := payload.SourceRepoID
	sourceBranch := payload.SourceBranch

	err := s.branchStore.UpdateLastPR(ctx, *sourceRepoID, sourceBranch, nil)
	if err != nil {
		return fmt.Errorf("failed to update last PR: %w", err)
	}

	logger.Debug().Msg("successfully updated branch's last created pullreq for closed pullreq")
	return nil
}

// handleEventPullReqReopened handles the pull request reopened event.
func (s *Service) handleEventPullReqReopened(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	payload := event.Payload
	if payload == nil {
		return fmt.Errorf("payload is nil")
	}

	if payload.SourceRepoID == nil {
		return events.NewDiscardEventErrorf("pullreq %d event missing sourceRepoID", payload.PullReqID)
	}

	logger := log.Ctx(ctx).With().
		Int64("source_repo_id", *payload.SourceRepoID).
		Int64("pullreq_id", payload.PullReqID).
		Str("source_branch", payload.SourceBranch).
		Logger()

	sourceRepoID := payload.SourceRepoID
	sourceBranch := payload.SourceBranch
	pullReqID := payload.PullReqID

	err := s.branchStore.UpdateLastPR(ctx, *sourceRepoID, sourceBranch, &pullReqID)
	if err != nil {
		return fmt.Errorf("failed to update last PR: %w", err)
	}

	logger.Debug().Msg("successfully updated branch's last created pullreq for reopened pullreq")
	return nil
}
