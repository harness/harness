// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"
	"errors"
	"fmt"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
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
	Commit    *CommitInfo         `json:"commit"`
	// Forced bool         `json:"forced"` TODO: data has to be calculated explicitly
}

// handleEventBranchCreated handles branch created events
// and triggers branch created webhooks for the source repo.
func (s *Server) handleEventBranchCreated(ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchCreated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.SHA)
			if err != nil {
				return nil, err
			}
			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchCreated,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    types.NilSHA,
				After:     event.Payload.SHA,
				Commit:    &commitInfo,
			}, nil
		})
}

// handleEventBranchUpdated handles branch updated events
// and triggers branch updated webhooks for the source repo.
func (s *Server) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchUpdated,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}

			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchUpdated,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.OldSHA,
				After:     event.Payload.NewSHA,
				Commit:    &commitInfo,
				// Forced: true/false, // TODO: data not available yet
			}, nil
		})
}

// handleEventBranchDeleted handles branch deleted events
// and triggers branch deleted webhooks for the source repo.
func (s *Server) handleEventBranchDeleted(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload]) error {
	return s.triggerForEventWithRepoAndPrincipal(ctx, enum.WebhookTriggerBranchDeleted,
		event.ID, event.Payload.RepoID, event.Payload.PrincipalID,
		func(repo *types.Repository, principal *types.Principal) (any, error) {
			return &BranchBody{
				Trigger:   enum.WebhookTriggerBranchDeleted,
				Repo:      repositoryInfoFrom(*repo, s.urlProvider),
				Principal: principalInfoFrom(*principal),
				Ref:       event.Payload.Ref,
				Before:    event.Payload.SHA,
				After:     types.NilSHA,
			}, nil
		})
}

func (s *Server) fetchCommitInfoForEvent(ctx context.Context, repoUID string, sha string) (CommitInfo, error) {
	out, err := s.gitRPCClient.GetCommit(ctx, &gitrpc.GetCommitParams{
		ReadParams: gitrpc.ReadParams{
			RepoUID: repoUID,
		},
		SHA: sha,
	})

	if errors.Is(err, gitrpc.ErrNotFound) {
		// this could happen if the commit has been deleted and garbage collected by now - discard event
		return CommitInfo{}, events.NewDiscardEventErrorf("commit with sha '%s' doesn't exist anymore", sha)
	}

	if err != nil {
		return CommitInfo{}, fmt.Errorf("failed to get commit with sha '%s': %w", sha, err)
	}

	return commitInfoFrom(out.Commit), nil
}
