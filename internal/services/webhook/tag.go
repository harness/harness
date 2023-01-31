// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// handleEventTagCreated handles tag created events
// and triggers tag created webhooks for the source repo.
func (s *Service) handleEventTagCreated(ctx context.Context,
	event *events.Event[*gitevents.TagCreatedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerTagCreated,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.SHA)
			if err != nil {
				return nil, err
			}
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerTagCreated,
				Repo:      repoInfo,
				Principal: principalInfoFrom(principal),
				Ref: ReferenceInfo{
					Name: event.Payload.Ref,
					Repo: repoInfo,
				},
				Before: types.NilSHA,
				After:  event.Payload.SHA,
				Commit: &commitInfo,
			}, nil
		})
}

// handleEventTagUpdated handles tag updated events
// and triggers tag updated webhooks for the source repo.
func (s *Service) handleEventTagUpdated(ctx context.Context,
	event *events.Event[*gitevents.TagUpdatedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerTagUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerTagUpdated,
				Repo:      repoInfo,
				Principal: principalInfoFrom(principal),
				Ref: ReferenceInfo{
					Name: event.Payload.Ref,
					Repo: repoInfo,
				},
				Before: event.Payload.OldSHA,
				After:  event.Payload.NewSHA,
				Forced: event.Payload.Forced,
				Commit: &commitInfo,
			}, nil
		})
}

// handleEventTagDeleted handles tag deleted events
// and triggers tag deleted webhooks for the source repo.
func (s *Service) handleEventTagDeleted(ctx context.Context,
	event *events.Event[*gitevents.TagDeletedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerTagDeleted,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerTagDeleted,
				Repo:      repoInfo,
				Principal: principalInfoFrom(principal),
				Ref: ReferenceInfo{
					Name: event.Payload.Ref,
					Repo: repoInfo,
				},
				Before: event.Payload.SHA,
				After:  types.NilSHA,
			}, nil
		})
}
