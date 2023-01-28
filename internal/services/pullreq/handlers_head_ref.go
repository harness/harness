// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strconv"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcenum "github.com/harness/gitness/gitrpc/enum"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types/enum"
)

// createHeadRefCreated handles pull request Created events.
// It creates the PR head git ref.
func (s *Service) createHeadRefCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	repoGit, err := s.repoGitInfoCache.Get(ctx, event.Payload.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo).
	// This is because commits from the source repository must be first pulled into the target repository.
	err = s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{RepoUID: repoGit.GitUID},
		Name:        strconv.Itoa(int(event.Payload.Number)),
		Type:        gitrpcenum.RefTypePullReqHead,
		NewValue:    event.Payload.SourceSHA,
		OldValue:    gitrpc.NilSHA, // this is a new pull request, so we expect that the ref doesn't exist
	})
	if err != nil {
		return fmt.Errorf("failed to update PR head ref: %w", err)
	}

	return nil
}

// updateHeadRefBranchUpdate handles pull request Branch Updated events.
// It updates the PR head git ref to point to the latest commit.
func (s *Service) updateHeadRefBranchUpdate(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	repoGit, err := s.repoGitInfoCache.Get(ctx, event.Payload.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo)
	// This is because commits from the source repository must be first pulled into the target repository.
	err = s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{RepoUID: repoGit.GitUID},
		Name:        strconv.Itoa(int(event.Payload.Number)),
		Type:        gitrpcenum.RefTypePullReqHead,
		NewValue:    event.Payload.NewSHA,
		OldValue:    event.Payload.OldSHA,
	})
	if err != nil {
		return fmt.Errorf("failed to update PR head ref after new commit: %w", err)
	}

	return nil
}

// updateHeadRefStateChange handles pull request StateChanged events.
// It updates the PR head git ref to point to the source branch commit SHA.
func (s *Service) updateHeadRefStateChange(ctx context.Context,
	event *events.Event[*pullreqevents.StateChangedPayload],
) error {
	// this handler need to execute only if the PR is being reopened (closed->open)
	if event.Payload.OldState != enum.PullReqStateClosed || event.Payload.NewState != enum.PullReqStateOpen {
		return nil
	}

	repoGit, err := s.repoGitInfoCache.Get(ctx, event.Payload.TargetRepoID)
	if err != nil {
		return fmt.Errorf("failed to get repo git info: %w", err)
	}

	// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo)
	// This is because commits from the source repository must be first pulled into the target repository.
	err = s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{RepoUID: repoGit.GitUID},
		Name:        strconv.Itoa(int(event.Payload.Number)),
		Type:        gitrpcenum.RefTypePullReqHead,
		NewValue:    event.Payload.SourceSHA,
		OldValue:    "", // the request is re-opened, so anything can be the old value
	})
	if err != nil {
		return fmt.Errorf("failed to update PR head ref after PR state change: %w", err)
	}

	return nil
}
