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

package trigger

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/bootstrap"
	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/events"
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
