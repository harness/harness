// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/internal/bootstrap"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/internal/pipeline/triggerer"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) handleEventTagCreated(ctx context.Context,
	event *events.Event[*gitevents.TagCreatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionTagCreated,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		Ref:         event.Payload.Ref,
		Before:      event.Payload.SHA,
		After:       event.Payload.SHA,
		Source:      event.Payload.Ref,
		Target:      event.Payload.Ref,
	}
	err := s.augmentCommitInfo(ctx, hook, event.Payload.RepoID, event.Payload.SHA)
	if err != nil {
		return fmt.Errorf("could not augment commit info: %w", err)
	}
	return s.trigger(ctx, event.Payload.RepoID, enum.TriggerActionTagCreated, hook)
}

func (s *Service) handleEventTagUpdated(ctx context.Context,
	event *events.Event[*gitevents.TagUpdatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionTagUpdated,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		Ref:         event.Payload.Ref,
		Before:      event.Payload.OldSHA,
		After:       event.Payload.NewSHA,
		Source:      event.Payload.Ref,
		Target:      event.Payload.Ref,
	}
	err := s.augmentCommitInfo(ctx, hook, event.Payload.RepoID, event.Payload.NewSHA)
	if err != nil {
		return fmt.Errorf("could not augment commit info: %w", err)
	}
	return s.trigger(ctx, event.Payload.RepoID, enum.TriggerActionTagUpdated, hook)
}
