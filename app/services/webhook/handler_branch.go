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

package webhook

import (
	"context"
	"fmt"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ReferencePayload describes the payload of Reference related webhook triggers.
// Note: Use same payload for all reference operations to make it easier for consumers.
type ReferencePayload struct {
	BaseSegment
	ReferenceSegment
	ReferenceDetailsSegment
	ReferenceUpdateSegment
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

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerBranchCreated,
					Repo:      repoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: event.Payload.Ref,
						Repo: repoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:    event.Payload.SHA,
					Commit: &commitInfo,
				},
				ReferenceUpdateSegment: ReferenceUpdateSegment{
					OldSHA: types.NilSHA,
					Forced: false,
				},
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

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerBranchUpdated,
					Repo:      repoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: event.Payload.Ref,
						Repo: repoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:    event.Payload.NewSHA,
					Commit: &commitInfo,
				},
				ReferenceUpdateSegment: ReferenceUpdateSegment{
					OldSHA: event.Payload.OldSHA,
					Forced: event.Payload.Forced,
				},
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

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerBranchDeleted,
					Repo:      repoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: event.Payload.Ref,
						Repo: repoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:    types.NilSHA,
					Commit: nil,
				},
				ReferenceUpdateSegment: ReferenceUpdateSegment{
					OldSHA: event.Payload.SHA,
					Forced: false,
				},
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

	if gitrpc.ErrorStatus(err) == gitrpc.StatusNotFound {
		// this could happen if the commit has been deleted and garbage collected by now
		// or if the sha doesn't point to an event - either way discard the event.
		return CommitInfo{}, events.NewDiscardEventErrorf("commit with sha '%s' doesn't exist", sha)
	}

	if err != nil {
		return CommitInfo{}, fmt.Errorf("failed to get commit with sha '%s': %w", sha, err)
	}

	return commitInfoFrom(out.Commit), nil
}
