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
	"github.com/harness/gitness/errors"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const MaxWebhookCommitFileStats = 20

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
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

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

// handleEventBranchUpdated handles branch updated events
// and triggers branch updated webhooks for the source repo.
func (s *Service) handleEventBranchUpdated(ctx context.Context,
	event *events.Event[*gitevents.BranchUpdatedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerBranchUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitsInfo, totalCommits, err := s.fetchCommitsInfoForEvent(ctx, repo.GitUID,
				event.Payload.OldSHA, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}

			commitInfo := commitsInfo[0]
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

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

// handleEventBranchDeleted handles branch deleted events
// and triggers branch deleted webhooks for the source repo.
func (s *Service) handleEventBranchDeleted(ctx context.Context,
	event *events.Event[*gitevents.BranchDeletedPayload]) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerBranchDeleted,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

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

func (s *Service) fetchCommitInfoForEvent(ctx context.Context, repoUID string, commitSHA string) (CommitInfo, error) {
	out, err := s.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: git.ReadParams{
			RepoUID: repoUID,
		},
		Revision: commitSHA,
	})

	if errors.AsStatus(err) == errors.StatusNotFound {
		// this could happen if the commit has been deleted and garbage collected by now
		// or if the targetSha doesn't point to an event - either way discard the event.
		return CommitInfo{}, events.NewDiscardEventErrorf("commit with targetSha '%s' doesn't exist", commitSHA)
	}

	if err != nil {
		return CommitInfo{}, fmt.Errorf("failed to get commit with targetSha '%s': %w", commitSHA, err)
	}

	return commitInfoFrom(out.Commit), nil
}

func (s *Service) fetchCommitsInfoForEvent(
	ctx context.Context,
	repoUID string,
	oldSHA string,
	newSHA string,
) ([]CommitInfo, int, error) {
	listCommitsParams := git.ListCommitsParams{
		ReadParams:   git.ReadParams{RepoUID: repoUID},
		GitREF:       newSHA,
		After:        oldSHA,
		Page:         0,
		Limit:        MaxWebhookCommitFileStats,
		IncludeStats: true,
	}
	listCommitsOutput, err := s.git.ListCommits(ctx, &listCommitsParams)

	if errors.AsStatus(err) == errors.StatusNotFound {
		// this could happen if the commit has been deleted and garbage collected by now
		// or if the targetSha doesn't point to an event - either way discard the event.
		return []CommitInfo{}, 0, events.NewDiscardEventErrorf("commit with targetSha '%s' doesn't exist", newSHA)
	}

	if err != nil {
		return []CommitInfo{}, 0, fmt.Errorf("failed to get commit with targetSha '%s': %w", newSHA, err)
	}

	if len(listCommitsOutput.Commits) == 0 {
		return nil, 0, fmt.Errorf("no commit found between %s and %s", oldSHA, newSHA)
	}

	return commitsInfoFrom(listCommitsOutput.Commits), listCommitsOutput.TotalCommits, nil
}
