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

// ReferenceBody describes the body of Reference related webhook triggers.
// NOTE: Use a single payload format to make it easier for consumers!
// TODO: move in separate package for small import?
type ReferenceBody struct {
	Trigger   enum.WebhookTrigger `json:"trigger"`
	Repo      RepositoryInfo      `json:"repo"`
	Principal PrincipalInfo       `json:"principal"`
	Ref       ReferenceInfo       `json:"ref"`
	Before    string              `json:"before"`
	After     string              `json:"after"`
	Commit    *CommitInfo         `json:"commit,omitempty"`
	Forced    bool                `json:"forced"`
}

// handleEventBranchCreated handles branch created events
// and triggers branch created webhooks for the source repo.
func (s *Service) handleEventBranchCreated(ctx context.Context,
	event *events.Event[*gitevents.BranchCreatedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerBranchCreated,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.SHA)
			if err != nil {
				return nil, err
			}
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerBranchCreated,
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

// handleEventBranchUpdated handles branch updated events
// and triggers branch updated webhooks for the source repo.
func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerBranchUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerBranchUpdated,
				Repo:      repoInfo,
				Principal: principalInfoFrom(principal),
				Ref: ReferenceInfo{
					Name: event.Payload.Ref,
					Repo: repoInfo,
				},
				Before: event.Payload.OldSHA,
				After:  event.Payload.NewSHA,
				Commit: &commitInfo,
				// Forced: true/false, // TODO: data not available yet
			}, nil
		})
}

// handleEventBranchDeleted handles branch deleted events
// and triggers branch deleted webhooks for the source repo.
func (s *Service) handleEventBranchDeleted(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerBranchDeleted,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			repoInfo := repositoryInfoFrom(repo, s.urlProvider)

			return &ReferenceBody{
				Trigger:   enum.WebhookTriggerBranchDeleted,
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

func (s *Service) fetchCommitInfoForEvent(ctx context.Context, repoUID string, sha string) (CommitInfo, error) {
	out, err := s.gitRPCClient.GetCommit(ctx, &gitrpc.GetCommitParams{
		ReadParams: gitrpc.ReadParams{
			RepoUID: repoUID,
		},
		SHA: sha,
	})

	if errors.Is(err, gitrpc.ErrNotFound) {
		// this could happen if the commit has been deleted and garbage collected by now
		// or if the sha doesn't point to an event - either way discard the event.
		return CommitInfo{}, events.NewDiscardEventErrorf("commit with sha '%s' doesn't exist", sha)
	}

	if err != nil {
		return CommitInfo{}, fmt.Errorf("failed to get commit with sha '%s': %w", sha, err)
	}

	return commitInfoFrom(out.Commit), nil
}
