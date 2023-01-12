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

// TagBody describes the body of Tag related webhook triggers.
// NOTE: Use a single payload format (and keep it similar to BranchBody) to make it easier for consumers!
// TODO: move in separate package for small import?
type TagBody struct {
	Trigger   enum.WebhookTrigger `json:"trigger"`
	Repo      RepositoryInfo      `json:"repo"`
	Principal PrincipalInfo       `json:"principal"`
	Ref       string              `json:"ref"`
	Before    string              `json:"before"`
	After     string              `json:"after"`
	Forced    bool                `json:"forced"` // tags can only be force-updated, include to be explicit.
}

// handleEventTagCreated handles tag created events
// and triggers tag created webhooks for the source repo.
func (s *Server) handleEventTagCreated(ctx context.Context,
	event *events.Event[*gitevents.TagCreatedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerTagCreated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			return &TagBody{
				Trigger:   enum.WebhookTriggerTagCreated,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    types.NilSHA,
				After:     event.Payload.SHA,
			}, nil
		})
}

// handleEventTagUpdated handles tag updated events
// and triggers tag updated webhooks for the source repo.
func (s *Server) handleEventTagUpdated(ctx context.Context,
	event *events.Event[*gitevents.TagUpdatedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerTagUpdated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			return &TagBody{
				Trigger:   enum.WebhookTriggerTagUpdated,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.OldSHA,
				After:     event.Payload.NewSHA,
				Forced:    event.Payload.Forced,
			}, nil
		})
}

// handleEventTagDeleted handles tag deleted events
// and triggers tag deleted webhooks for the source repo.
func (s *Server) handleEventTagDeleted(ctx context.Context,
	event *events.Event[*gitevents.TagDeletedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerTagDeleted,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			return &TagBody{
				Trigger:   enum.WebhookTriggerTagDeleted,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.SHA,
				After:     types.NilSHA,
			}, nil
		})
}
