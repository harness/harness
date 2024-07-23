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

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/events"
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
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerTagCreated,
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
					SHA:        event.Payload.SHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
				ReferenceUpdateSegment: ReferenceUpdateSegment{
					OldSHA: types.NilSHA,
					Forced: false,
				},
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
			commitsInfo, totalCommits, err := s.fetchCommitsInfoForEvent(ctx, repo.GitUID,
				event.Payload.OldSHA, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}

			commitInfo := CommitInfo{}
			if len(commitsInfo) > 0 {
				commitInfo = commitsInfo[0]
			}
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerTagUpdated,
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
					SHA:               event.Payload.NewSHA,
					Commit:            &commitInfo,
					HeadCommit:        &commitInfo,
					Commits:           &commitsInfo,
					TotalCommitsCount: totalCommits,
				},
				ReferenceUpdateSegment: ReferenceUpdateSegment{
					OldSHA: event.Payload.OldSHA,
					Forced: event.Payload.Forced,
				},
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
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

			return &ReferencePayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerTagDeleted,
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
