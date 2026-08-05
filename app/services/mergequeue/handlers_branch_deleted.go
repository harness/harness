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

package mergequeue

import (
	"context"
	"fmt"

	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// handlerBranchDeleted is invoked whenever a merge queue protected branch is deleted.
func (s *Service) handlerBranchDeleted(
	ctx context.Context,
	event *events.Event[*mergequeueevents.BranchDeletedPayload],
) error {
	log.Ctx(ctx).Debug().
		Int64("repo_id", event.Payload.RepoID).
		Str("branch", event.Payload.Branch).
		Msg("merge queue branch deleted event received")

	repoID := event.Payload.RepoID
	branch := event.Payload.Branch

	repo, err := s.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo: %w", err)
	}

	if err := s.RemoveAll(ctx, repo, branch, types.PullRequestActivityPayloadMergeQueueRemove{
		Reason: enum.MergeQueueRemovalReasonTargetDeleted,
	}); err != nil {
		return fmt.Errorf("failed to clear merge queue after target branch %q was deleted: %w", branch, err)
	}

	return nil
}
