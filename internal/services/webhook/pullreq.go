// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package webhook

import (
	"context"

	"github.com/harness/gitness/events"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

const (
	// gitReferenceNamePrefixBranch is the prefix of references of type branch.
	gitReferenceNamePrefixBranch = "refs/heads/"
)

// PullReqBranchBody describes the body of the pullreq branch related webhook trigger.
// NOTE: Embed ReferenceBody to make it easier for consumers!
// TODO: move in separate package for small import?
type PullReqBranchBody struct {
	ReferenceBody
	TargetRef ReferenceInfo `json:"target_ref"`
	PullReq   PullReqInfo   `json:"pull_req"`
}

// handleEventPullReqBranchUpdated handles branch updated events for pull requests
// and triggers pullreq branch updated webhooks for the source repo.
func (s *Service) handleEventPullReqBranchUpdated(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload]) error {
	return s.triggerForEventWithPullReq(ctx, enum.WebhookTriggerPullReqBranchUpdated,
		event.ID, event.Payload.PrincipalID, event.Payload.PullReqID,
		func(principal *types.Principal, pr *types.PullReq, targetRepo, sourceRepo *types.Repository) (any, error) {
			commitInfo, err := s.fetchCommitInfoForEvent(ctx, sourceRepo.GitUID, event.Payload.NewSHA)
			if err != nil {
				return nil, err
			}
			targetRepoInfo := repositoryInfoFrom(targetRepo, s.urlProvider)
			sourceRepoInfo := repositoryInfoFrom(sourceRepo, s.urlProvider)

			return &PullReqBranchBody{
				ReferenceBody: ReferenceBody{
					Trigger:   enum.WebhookTriggerPullReqBranchUpdated,
					Repo:      targetRepoInfo,
					Principal: principalInfoFrom(principal),
					Ref: ReferenceInfo{
						Name: gitReferenceNamePrefixBranch + pr.SourceBranch,
						Repo: sourceRepoInfo,
					},
					Before: event.Payload.OldSHA,
					After:  event.Payload.NewSHA,
					Commit: &commitInfo,
					// Forced: true/false, // TODO: data not available yet
				},
				TargetRef: ReferenceInfo{
					Name: gitReferenceNamePrefixBranch + pr.TargetBranch,
					Repo: targetRepoInfo,
				},
				PullReq: pullReqInfoFrom(pr),
			}, nil
		})
}
