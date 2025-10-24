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
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) handleEventPullReqCreated(ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionPullReqCreated,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		After:       event.Payload.SourceSHA,
	}
	err := s.augmentPullReqInfo(ctx, hook, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("could not augment pull request info: %w", err)
	}
	return s.trigger(ctx, event.Payload.TargetRepoID, enum.TriggerActionPullReqCreated, hook)
}

func (s *Service) handleEventPullReqReopened(ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionPullReqReopened,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		After:       event.Payload.SourceSHA,
	}
	err := s.augmentPullReqInfo(ctx, hook, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("could not augment pull request info: %w", err)
	}
	return s.trigger(ctx, event.Payload.TargetRepoID, enum.TriggerActionPullReqReopened, hook)
}

func (s *Service) handleEventPullReqBranchUpdated(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionPullReqBranchUpdated,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		After:       event.Payload.NewSHA,
	}
	err := s.augmentPullReqInfo(ctx, hook, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("could not augment pull request info: %w", err)
	}
	return s.trigger(ctx, event.Payload.TargetRepoID, enum.TriggerActionPullReqBranchUpdated, hook)
}

func (s *Service) handleEventPullReqClosed(ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionPullReqClosed,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		After:       event.Payload.SourceSHA,
	}
	err := s.augmentPullReqInfo(ctx, hook, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("could not augment pull request info: %w", err)
	}
	return s.trigger(ctx, event.Payload.TargetRepoID, enum.TriggerActionPullReqClosed, hook)
}

func (s *Service) handleEventPullReqMerged(
	ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionPullReqMerged,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		After:       event.Payload.SourceSHA,
	}
	err := s.augmentPullReqInfo(ctx, hook, event.Payload.PullReqID)
	if err != nil {
		return fmt.Errorf("could not augment pull request info: %w", err)
	}
	return s.trigger(ctx, event.Payload.TargetRepoID, enum.TriggerActionPullReqMerged, hook)
}

// augmentPullReqInfo adds in information into the hook pertaining to the pull request
// by querying the database.
func (s *Service) augmentPullReqInfo(
	ctx context.Context,
	hook *triggerer.Hook,
	pullReqID int64,
) error {
	pullreq, err := s.pullReqStore.Find(ctx, pullReqID)
	if err != nil {
		return fmt.Errorf("could not find pull request: %w", err)
	}
	hook.Title = pullreq.Title
	hook.Timestamp = pullreq.Created
	hook.AuthorLogin = pullreq.Author.UID
	hook.AuthorName = pullreq.Author.DisplayName
	hook.AuthorEmail = pullreq.Author.Email
	hook.Message = pullreq.Description
	hook.Before = pullreq.MergeBaseSHA
	hook.Target = pullreq.TargetBranch
	hook.Source = pullreq.SourceBranch
	// expand the branch to a git reference.
	hook.Ref = fmt.Sprintf("refs/pullreq/%d/head", pullreq.Number)
	return nil
}
