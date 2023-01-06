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

// BranchBody describes the body of Branch related webhook triggers.
// NOTE: Use a single payload format to make it easier for consumers!
// TODO: move in separate package for small import?
type BranchBody struct {
	Trigger   enum.WebhookTrigger `json:"trigger"`
	Repo      RepositoryInfo      `json:"repo"`
	Principal PrincipalInfo       `json:"principal"`
	Ref       string              `json:"ref"`
	Before    string              `json:"before"`
	After     string              `json:"after"`
	// Forced bool         `json:"forced"` TODO: data has to be calculated explicitly
}

// handleEventBranchCreated handles branch created events
// and triggers branch created webhooks for the source repo.
func (s *Server) handleEventBranchCreated(ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload]) error {
	return s.triggerWebhooksForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchCreated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) interface{} {
			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchCreated,
				Repo:      repositoryInfoFrom(repo, s.urlProvider),
				Principal: principalInfoFrom(principal),
				Ref:       event.Payload.Ref,
				Before:    types.NilSHA,
				After:     event.Payload.SHA,
			}
		})
}

// handleEventBranchUpdated handles branch updated events
// and triggers branch updated webhooks for the source repo.
func (s *Server) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return s.triggerWebhooksForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchUpdated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) interface{} {
			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchUpdated,
				Repo:      repositoryInfoFrom(repo, s.urlProvider),
				Principal: principalInfoFrom(principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.OldSHA,
				After:     event.Payload.NewSHA,
				// Forced: true/false, // TODO: data not available yet
			}
		})
}

// handleEventBranchDeleted handles branch deleted events
// and triggers branch deleted webhooks for the source repo.
func (s *Server) handleEventBranchDeleted(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload]) error {
	return s.triggerWebhooksForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchDeleted,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) interface{} {
			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchDeleted,
				Repo:      repositoryInfoFrom(repo, s.urlProvider),
				Principal: principalInfoFrom(principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.SHA,
				After:     types.NilSHA,
			}
		})
}
