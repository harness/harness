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

package pullreq

import (
	"context"
	"fmt"

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
)

// updatePRCountersOnCreated increments number of PRs and open PRs.
func (s *Service) updatePRCountersOnCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	err := s.updatePRNumbers(ctx, event.Payload.TargetRepoID, 1, 1, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to update repository pull request numbers after PR creation: %w", err)
	}

	return nil
}

// updatePRCountersOnReopened increments number of open PRs and decrements number of closed.
func (s *Service) updatePRCountersOnReopened(ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	err := s.updatePRNumbers(ctx, event.Payload.TargetRepoID, 0, 1, -1, 0)
	if err != nil {
		return fmt.Errorf("failed to update repository pull request numbers after PR reopen: %w", err)
	}

	return nil
}

// updatePRCountersOnClosed increments number of closed PRs and decrements number of open.
func (s *Service) updatePRCountersOnClosed(ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	err := s.updatePRNumbers(ctx, event.Payload.TargetRepoID, 0, -1, 1, 0)
	if err != nil {
		return fmt.Errorf("failed to update repository pull request numbers after PR close: %w", err)
	}

	return nil
}

// updatePRCountersOnMerged increments number of merged PRs and decrements number of open.
func (s *Service) updatePRCountersOnMerged(ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	err := s.updatePRNumbers(ctx, event.Payload.TargetRepoID, 0, -1, 0, 1)
	if err != nil {
		return fmt.Errorf("failed to update repository pull request numbers after PR merge: %w", err)
	}

	return nil
}

func (s *Service) updatePRNumbers(ctx context.Context, repoID int64,
	deltaNew, deltaOpen, deltaClosed, deltaMerged int,
) error {
	repo, err := s.repoStore.Find(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to get repository to update PR numbers: %w", err)
	}

	_, err = s.repoStore.UpdateOptLock(ctx, repo, func(repo *types.Repository) error {
		repo.NumPulls += deltaNew
		repo.NumOpenPulls += deltaOpen
		repo.NumClosedPulls += deltaClosed
		repo.NumMergedPulls += deltaMerged
		return nil
	})

	return err
}
