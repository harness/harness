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
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store"

	"github.com/rs/zerolog/log"
)

// handlerUpdated is invoked whenever a merge queue is modified (PR has been added or removed).
func (s *Service) handlerUpdated(
	ctx context.Context,
	event *events.Event[*mergequeueevents.UpdatedPayload],
) error {
	log.Ctx(ctx).Debug().
		Int64("repo_id", event.Payload.RepoID).
		Str("branch", event.Payload.Branch).
		Msg("merge queue updated event received")

	repoID := event.Payload.RepoID
	branch := event.Payload.Branch

	repo, err := s.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to find repo: %w", err)
	}

	branchProtection, err := s.protectionManager.ListRepoBranchRules(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to list repo branch rules: %w", err)
	}

	mergeQueueSetup, err := branchProtection.GetMergeQueueSetup(protection.MergeQueueSetupInput{
		Repo:         repo,
		TargetBranch: branch,
	})
	if err != nil {
		return fmt.Errorf("failed to get merge queue setup: %w", err)
	}

	if !mergeQueueSetup.IsActive() {
		return nil
	}

	q, err := s.mergeQueueStore.FindByRepoAndBranch(ctx, repoID, branch)
	if errors.Is(err, store.ErrResourceNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to find merge queue: %w", err)
	}

	err = s.reprocess(ctx, repo, q, mergeQueueSetup)
	if err != nil {
		return fmt.Errorf("failed to process merge queue after being updated: %w", err)
	}

	return nil
}
