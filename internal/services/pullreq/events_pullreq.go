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
)

func (s *Service) handleEventPullReqCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	payload := event.Payload

	// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo)
	err := s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{RepoUID: payload.TargetRepoGitUID},
		Name:        strconv.Itoa(int(payload.Number)),
		Type:        gitrpcenum.RefTypePullReqHead,
		NewValue:    payload.SourceSHA,
		OldValue:    gitrpc.NilSHA, // this is a new pull request, so we expect that the ref doesn't exist
	})
	if err != nil {
		return fmt.Errorf("failed to update PR head ref: %w", err)
	}

	return nil
}

func (s *Service) handleEventPullReqUpdated(ctx context.Context,
	event *events.Event[*pullreqevents.UpdatedPayload],
) error {
	return nil
}

func (s *Service) handleEventPullReqStateChange(ctx context.Context,
	event *events.Event[*pullreqevents.StateChangePayload],
) error {
	if event.Payload.SourceSHA == "" {
		return nil
	}

	// TODO: This doesn't work for forked repos (only works when sourceRepo==targetRepo)
	err := s.gitRPCClient.UpdateRef(ctx, gitrpc.UpdateRefParams{
		WriteParams: gitrpc.WriteParams{RepoUID: event.Payload.TargetRepoGitUID},
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

func (s *Service) handleEventPullReqMerged(ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	return nil
}
