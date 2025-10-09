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
	"strings"

	"github.com/harness/gitness/app/bootstrap"
	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"
)

// TODO: This can be moved to SCM library
func ExtractBranch(ref string) string {
	return strings.TrimPrefix(ref, "refs/heads/")
}

func (s *Service) handleEventBranchCreated(ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionBranchCreated,
		Ref:         event.Payload.Ref,
		Source:      ExtractBranch(event.Payload.Ref),
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		Target:      ExtractBranch(event.Payload.Ref),
		After:       event.Payload.SHA,
	}
	err := s.augmentCommitInfo(ctx, hook, event.Payload.RepoID, event.Payload.SHA)
	if err != nil {
		return fmt.Errorf("could not augment commit info: %w", err)
	}
	return s.trigger(ctx, event.Payload.RepoID, enum.TriggerActionBranchCreated, hook)
}

func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	hook := &triggerer.Hook{
		Trigger:     enum.TriggerHook,
		Action:      enum.TriggerActionBranchUpdated,
		Ref:         event.Payload.Ref,
		Before:      event.Payload.OldSHA,
		After:       event.Payload.NewSHA,
		TriggeredBy: bootstrap.NewSystemServiceSession().Principal.ID,
		Source:      ExtractBranch(event.Payload.Ref),
		Target:      ExtractBranch(event.Payload.Ref),
	}
	err := s.augmentCommitInfo(ctx, hook, event.Payload.RepoID, event.Payload.NewSHA)
	if err != nil {
		return fmt.Errorf("could not augment commit info: %w", err)
	}
	return s.trigger(ctx, event.Payload.RepoID, enum.TriggerActionBranchUpdated, hook)
}

// augmentCommitInfo adds information about the commit to the hook by interacting with
// the commit service.
func (s *Service) augmentCommitInfo(
	ctx context.Context,
	hook *triggerer.Hook,
	repoID int64,
	sha string,
) error {
	repo, err := s.repoFinder.FindByID(ctx, repoID)
	if err != nil {
		return fmt.Errorf("could not find repo: %w", err)
	}

	commit, err := s.commitSvc.FindCommit(ctx, repo, sha)
	if err != nil {
		return fmt.Errorf("could not find commit info")
	}

	hook.AuthorName = commit.Author.Identity.Name
	hook.Title = commit.Title
	hook.Timestamp = commit.Committer.When.UnixMilli()
	hook.AuthorLogin = commit.Author.Identity.Name
	hook.AuthorEmail = commit.Author.Identity.Email
	hook.Message = commit.Message

	return nil
}
