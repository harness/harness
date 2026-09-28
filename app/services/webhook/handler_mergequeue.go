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

	mergequeueevents "github.com/harness/gitness/app/events/mergequeue"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MergeQueueChecksPayload describes the body of merge queue checks requested/canceled triggers.
type MergeQueueChecksPayload struct {
	BaseSegment
	MergeQueue MergeQueueSegment `json:"merge_queue"`
	MergeQueueChecksSegment
	ReferenceDetailsSegment
}

// MergeQueueSegment contains merge queue details.
type MergeQueueSegment struct {
	Branch string `json:"branch"`
}

// MergeQueueChecksSegment contains merge queue checks details.
type MergeQueueChecksSegment struct {
	CommitSHA string `json:"commit_sha"`
}

// handleEventMergeQueueChecksRequested handles merge queue checks requested events.
func (s *Service) handleEventMergeQueueChecksRequested(
	ctx context.Context,
	event *events.Event[*mergequeueevents.ChecksRequestedPayload],
) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerMergeQueueChecksRequested,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, repo.GitUID, repo.Path, event.Payload.CommitSHA, s.urlProvider)
			if err != nil {
				return nil, err
			}
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

			return &MergeQueueChecksPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerMergeQueueChecksRequested,
					Repo:      repoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				MergeQueue: MergeQueueSegment{
					Branch: event.Payload.Branch,
				},
				MergeQueueChecksSegment: MergeQueueChecksSegment{
					CommitSHA: event.Payload.CommitSHA,
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        commitInfo.SHA,
					HeadCommit: &commitInfo,
				},
			}, nil
		})
}

// handleEventMergeQueueChecksCanceled handles merge queue checks canceled events.
func (s *Service) handleEventMergeQueueChecksCanceled(
	ctx context.Context,
	event *events.Event[*mergequeueevents.ChecksCanceledPayload],
) error {
	return s.triggerForEventWithRepo(ctx, enum.WebhookTriggerMergeQueueChecksCanceled,
		event.ID, event.Payload.PrincipalID, event.Payload.RepoID,
		func(principal *types.Principal, repo *types.Repository) (any, error) {
			repoInfo := repositoryInfoFrom(ctx, repo, s.urlProvider)

			return &MergeQueueChecksPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerMergeQueueChecksCanceled,
					Repo:      repoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				MergeQueue: MergeQueueSegment{
					Branch: event.Payload.Branch,
				},
				MergeQueueChecksSegment: MergeQueueChecksSegment{
					CommitSHA: event.Payload.CommitSHA,
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA: event.Payload.CommitSHA,
				},
			}, nil
		})
}
