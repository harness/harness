// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/gitrpc/events"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// BranchBody describes the body of Branch related webhook triggers.
// TODO: move in separate package for small import?
type BranchBody struct {
	Repo   RepoMetadata `json:"repo"`
	Ref    string       `json:"ref"`
	Before string       `json:"before"`
	After  string       `json:"after"`
	Forced bool         `json:"forced"`
}

func getEventHandlerForBranchCreated(server *Server,
	repoStore store.RepoStore) func(context.Context, *events.Event[*gitevents.BranchCreatedPayload]) error {
	return func(ctx context.Context, event *events.Event[*gitevents.BranchCreatedPayload]) error {
		return triggerForEventWithGitUID(ctx, server, repoStore, event.ID, event.Payload.RepoUID,
			enum.WebhookTriggerBranchPushed, func(repo *types.Repository) interface{} {
				return &BranchBody{
					Repo: RepoMetadata{
						ID:            repo.ID,
						Path:          repo.Path,
						UID:           repo.UID,
						DefaultBranch: repo.DefaultBranch,
						GitURL:        "", // TODO: GitURL has to be generated
					},
					Ref:    event.Payload.FullRef,
					Before: types.NilSHA,
					After:  event.Payload.SHA,
				}
			})
	}
}

func getEventHandlerForBranchUpdated(server *Server,
	repoStore store.RepoStore) func(context.Context, *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return func(ctx context.Context, event *events.Event[*gitevents.BranchUpdatedPayload]) error {
		return triggerForEventWithGitUID(ctx, server, repoStore, event.ID, event.Payload.RepoUID,
			enum.WebhookTriggerBranchPushed, func(repo *types.Repository) interface{} {
				return &BranchBody{
					Repo: RepoMetadata{
						ID:            repo.ID,
						Path:          repo.Path,
						UID:           repo.UID,
						DefaultBranch: repo.DefaultBranch,
						GitURL:        "", // TODO: GitURL has to be generated
					},
					Ref:    event.Payload.FullRef,
					Before: event.Payload.OldSHA,
					After:  event.Payload.NewSHA,
					Forced: event.Payload.Forced,
				}
			})
	}
}

func getEventHandlerForBranchDeleted(server *Server,
	repoStore store.RepoStore) func(context.Context, *events.Event[*gitevents.BranchDeletedPayload]) error {
	return func(ctx context.Context, event *events.Event[*gitevents.BranchDeletedPayload]) error {
		return triggerForEventWithGitUID(ctx, server, repoStore, event.ID, event.Payload.RepoUID,
			enum.WebhookTriggerBranchDeleted, func(repo *types.Repository) interface{} {
				return &BranchBody{
					Repo: RepoMetadata{
						ID:            repo.ID,
						Path:          repo.Path,
						UID:           repo.UID,
						DefaultBranch: repo.DefaultBranch,
						GitURL:        "", // TODO: GitURL has to be generated
					},
					Ref:    event.Payload.FullRef,
					Before: event.Payload.SHA,
					After:  types.NilSHA,
				}
			})
	}
}
