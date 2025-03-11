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

	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// gitReferenceNamePrefixBranch is the prefix of references of type branch.
	gitReferenceNamePrefixBranch = "refs/heads/"
)

// PullReqCreatedPayload describes the body of the pullreq created trigger.
// TODO: move in separate package for small import?
type PullReqCreatedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	ReferenceDetailsSegment
}

// handleEventPullReqCreated handles created events for pull requests
// and triggers pullreq created webhooks for the source repo.
func (s *Service) handleEventPullReqCreated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CreatedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqCreated,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.SourceSHA)
			if err != nil {
				return nil, err
			}
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqCreatedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqCreated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        event.Payload.SourceSHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
			}, nil
		})
}

// PullReqReopenedPayload describes the body of the pullreq reopened trigger.
// Note: same as payload for created.
type PullReqReopenedPayload PullReqCreatedPayload

// handleEventPullReqReopened handles reopened events for pull requests
// and triggers pullreq reopened webhooks for the source repo.
func (s *Service) handleEventPullReqReopened(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReopenedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqReopened,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.SourceSHA)
			if err != nil {
				return nil, err
			}
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqReopenedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqReopened,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        event.Payload.SourceSHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
			}, nil
		})
}

// PullReqBranchUpdatedPayload describes the body of the pullreq branch updated trigger.
// TODO: move in separate package for small import?
type PullReqBranchUpdatedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	ReferenceDetailsSegment
	ReferenceUpdateSegment
}

// handleEventPullReqBranchUpdated handles branch updated events for pull requests
// and triggers pullreq branch updated webhooks for the source repo.
func (s *Service) handleEventPullReqBranchUpdated(
	ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqBranchUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitsInfo, totalCommits, err := s.fetchCommitsInfoForEvent(ctx, sourceRepo.GitUID,
				event.Payload.OldSHA, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}

			commitInfo := commitsInfo[0]
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqBranchUpdatedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqBranchUpdated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
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

// PullReqClosedPayload describes the body of the pullreq closed trigger.
type PullReqClosedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	ReferenceDetailsSegment
}

func (s *Service) handleEventPullReqClosed(
	ctx context.Context,
	event *events.Event[*pullreqevents.ClosedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqClosed,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.SourceSHA)
			if err != nil {
				return nil, err
			}
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqClosedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqClosed,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        event.Payload.SourceSHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
			}, nil
		})
}

// PullReqMergedPayload describes the body of the pullreq merged trigger.
type PullReqMergedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	ReferenceDetailsSegment
}

func (s *Service) handleEventPullReqMerged(
	ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqMerged,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.SourceSHA)
			if err != nil {
				return nil, err
			}
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqClosedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqMerged,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        event.Payload.SourceSHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
			}, nil
		})
}

// PullReqCommentPayload describes the body of the pullreq comment create trigger.
type PullReqCommentPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	ReferenceDetailsSegment
	PullReqCommentSegment
}

func (s *Service) handleEventPullReqComment(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentCreatedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqCommentCreated,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)
			activity, err := s.activityStore.Find(ctx, event.Payload.ActivityID)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to get activity by id for acitivity id %d: %w",
					event.Payload.ActivityID,
					err,
				)
			}
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.SourceSHA)
			if err != nil {
				return nil, err
			}
			return &PullReqCommentPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqCommentCreated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				ReferenceDetailsSegment: ReferenceDetailsSegment{
					SHA:        event.Payload.SourceSHA,
					Commit:     &commitInfo,
					HeadCommit: &commitInfo,
				},
				PullReqCommentSegment: PullReqCommentSegment{
					CommentInfo: CommentInfo{
						Text:     activity.Text,
						ID:       activity.ID,
						ParentID: activity.ParentID,
						Kind:     activity.Kind,
						Created:  activity.Created,
						Updated:  activity.Updated,
					},
					CodeCommentInfo: extractCodeCommentInfoIfAvailable(activity),
				},
			}, nil
		})
}

// handleEventPullReqCommentUpdated handles updated events for pull request comments.
func (s *Service) handleEventPullReqCommentUpdated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentUpdatedPayload],
) error {
	return s.triggerForEventWithPullReq(
		ctx,
		enum.WebhookTriggerPullReqCommentUpdated,
		event.ID,
		event.Payload.PrincipalID,
		event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)
			activity, err := s.activityStore.Find(ctx, event.Payload.ActivityID)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to get activity by id for acitivity id %d: %w",
					event.Payload.ActivityID,
					err,
				)
			}
			return &PullReqCommentPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqCommentUpdated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				PullReqCommentSegment: PullReqCommentSegment{
					CommentInfo: CommentInfo{
						Text:     activity.Text,
						ID:       activity.ID,
						ParentID: activity.ParentID,
						Created:  activity.Created,
						Updated:  activity.Updated,
						Kind:     activity.Kind,
					},
					CodeCommentInfo: extractCodeCommentInfoIfAvailable(activity),
				},
			}, nil
		})
}

func extractCodeCommentInfoIfAvailable(
	activity *types.PullReqActivity,
) *CodeCommentInfo {
	if !activity.IsValidCodeComment() {
		return nil
	}
	return (*CodeCommentInfo)(activity.CodeComment)
}

// PullReqLabelAssignedPayload describes the body of the pullreq label assignment trigger.
type PullReqLabelAssignedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqLabelSegment
}

func (s *Service) handleEventPullReqLabelAssigned(
	ctx context.Context,
	event *events.Event[*pullreqevents.LabelAssignedPayload],
) error {
	return s.triggerForEventWithPullReq(
		ctx,
		enum.WebhookTriggerPullReqLabelAssigned,
		event.ID, event.Payload.PrincipalID,
		event.Payload.PullReqID,
		func(
			principal *types.Principal,
			pr *types.PullReq,
			targetRepo,
			_ *types.Repository,
		) (any, error) {
			label, err := s.labelStore.FindByID(ctx, event.Payload.LabelID)
			if err != nil {
				return nil, fmt.Errorf("failed to find label by id: %w", err)
			}

			var labelValue *string
			if event.Payload.ValueID != nil {
				value, err := s.labelValueStore.FindByID(ctx, *event.Payload.ValueID)
				if err != nil {
					return nil, fmt.Errorf("failed to find label value by id: %d %w", *event.Payload.ValueID, err)
				}
				labelValue = &value.Value
			}

			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			return &PullReqLabelAssignedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqLabelAssigned,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqLabelSegment: PullReqLabelSegment{
					LabelInfo: LabelInfo{
						ID:      event.Payload.LabelID,
						Key:     label.Key,
						ValueID: event.Payload.ValueID,
						Value:   labelValue,
					},
				},
			}, nil
		})
}

// PullReqUpdatedPayload describes the body of the pullreq updated trigger.
type PullReqUpdatedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	PullReqUpdateSegment
}

// handleEventPullReqUpdated handles updated events for pull requests
// and triggers pullreq updated webhooks for the target repo.
func (s *Service) handleEventPullReqUpdated(
	ctx context.Context,
	event *events.Event[*pullreqevents.UpdatedPayload],
) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			return &PullReqUpdatedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqUpdated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				PullReqUpdateSegment: PullReqUpdateSegment{
					TitleChanged:       event.Payload.TitleChanged,
					TitleOld:           event.Payload.TitleOld,
					TitleNew:           event.Payload.TitleNew,
					DescriptionChanged: event.Payload.DescriptionChanged,
					DescriptionOld:     event.Payload.DescriptionOld,
					DescriptionNew:     event.Payload.DescriptionNew,
				},
			}, nil
		})
}

// PullReqActivityStatusUpdatedPayload describes the body of the pullreq comment updated trigger.
type PullReqActivityStatusUpdatedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	PullReqCommentSegment
	PullReqCommentStatusUpdatedSegment
}

// handleEventPullReqCommentUpdated handles status updated events for pull request comments.
func (s *Service) handleEventPullReqCommentStatusUpdated(
	ctx context.Context,
	event *events.Event[*pullreqevents.CommentStatusUpdatedPayload],
) error {
	return s.triggerForEventWithPullReq(
		ctx,
		enum.WebhookTriggerPullReqCommentStatusUpdated,
		event.ID,
		event.Payload.PrincipalID,
		event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)
			activity, err := s.activityStore.Find(ctx, event.Payload.ActivityID)
			if err != nil {
				return nil, fmt.Errorf(
					"failed to get activity by id for acitivity id %d: %w",
					event.Payload.ActivityID,
					err,
				)
			}

			status := enum.PullReqCommentStatusActive
			if activity.Resolved != nil {
				status = enum.PullReqCommentStatusResolved
			}

			return &PullReqActivityStatusUpdatedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqCommentStatusUpdated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				PullReqCommentSegment: PullReqCommentSegment{
					CommentInfo: CommentInfo{
						ID:       activity.ID,
						Text:     activity.Text,
						Kind:     activity.Kind,
						ParentID: activity.ParentID,
						Created:  activity.Created,
						Updated:  activity.Updated,
					},
					CodeCommentInfo: extractCodeCommentInfoIfAvailable(activity),
				},
				PullReqCommentStatusUpdatedSegment: PullReqCommentStatusUpdatedSegment{
					Status: status,
				},
			}, nil
		})
}

// PullReqReviewSubmittedPayload describes the body of the pullreq review submitted trigger.
type PullReqReviewSubmittedPayload struct {
	BaseSegment
	PullReqSegment
	PullReqTargetReferenceSegment
	ReferenceSegment
	PullReqReviewSegment
}

// handleEventPullReqReviewSubmitted handles review events for pull requests
// and triggers pullreq review submitted webhooks for the target repo.
func (s *Service) handleEventPullReqReviewSubmitted(
	ctx context.Context,
	event *events.Event[*pullreqevents.ReviewSubmittedPayload],
) error {
	return s.triggerForEventWithPullReq(
		ctx,
		enum.WebhookTriggerPullReqReviewSubmitted,
		event.ID, event.Payload.PrincipalID,
		event.Payload.PullReqID,
		func(
			principal *types.Principal,
			pr *types.PullReq,
			targetRepo,
			sourceRepo *types.Repository,
		) (any, error) {
			targetRepoInfo := repositoryInfoFrom(ctx, targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(ctx, sourceRepo, s.urlProvider)

			reviewer, err := s.WebhookExecutor.FindPrincipalForEvent(ctx, event.Payload.ReviewerID)
			if err != nil {
				return nil, fmt.Errorf("failed to get reviewer by id for reviewer id %d: %w", event.Payload.ReviewerID, err)
			}

			return &PullReqReviewSubmittedPayload{
				BaseSegment: BaseSegment{
					Trigger:   enum.WebhookTriggerPullReqReviewSubmitted,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal.ToPrincipalInfo()),
				},
				PullReqSegment: PullReqSegment{
					PullReq: pullReqInfoFrom(ctx, pr, targetRepo, s.urlProvider),
				},
				PullReqTargetReferenceSegment: PullReqTargetReferenceSegment{
					TargetRef: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
						Repo: targetRepoInfo,
					},
				},
				ReferenceSegment: ReferenceSegment{
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
				},
				PullReqReviewSegment: PullReqReviewSegment{
					ReviewDecision: event.Payload.Decision,
					ReviewerInfo:   principalInfoFrom(reviewer.ToPrincipalInfo()),
				},
			}, nil
		})
}
