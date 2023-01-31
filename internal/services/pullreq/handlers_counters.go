// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"

	"github.com/harness/gitness/events"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
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
