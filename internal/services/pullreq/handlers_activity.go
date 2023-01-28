// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/events"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// addActivityBranchUpdate handles the pull request Commit events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityBranchUpdate(ctx context.Context,
	event *events.Event[*pullreqevents.BranchUpdatedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadBranchUpdate{
				Old: event.Payload.OldSHA,
				New: event.Payload.NewSHA,
			}
		})
}

// addActivityBranchDelete handles the pull request BranchDeleted events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityBranchDelete(ctx context.Context,
	event *events.Event[*pullreqevents.BranchDeletedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadBranchDelete{
				SHA: event.Payload.SHA,
			}
		})
}

// addActivityStateChange handles the pull request StateChanged events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityStateChange(ctx context.Context,
	event *events.Event[*pullreqevents.StateChangedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadStateChange{
				Old:     event.Payload.OldState,
				New:     event.Payload.NewState,
				IsDraft: pr.IsDraft,
				Message: event.Payload.Message,
			}
		})
}

// addActivityReviewSubmit handles the pull request ReviewSubmitted events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityReviewSubmit(ctx context.Context,
	event *events.Event[*pullreqevents.ReviewSubmittedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadReviewSubmit{
				Message:  event.Payload.Message,
				Decision: event.Payload.Decision,
			}
		})
}

// addActivityTitleChange handles the pull request TitleChanged events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityTitleChange(ctx context.Context,
	event *events.Event[*pullreqevents.TitleChangedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadTitleChange{
				Old: event.Payload.OldTitle,
				New: event.Payload.NewTitle,
			}
		})
}

// addActivityMerge handles the pull request Merged events and
// adds a new activity to the pull request's timeline with the corresponding type.
func (s *Service) addActivityMerge(ctx context.Context,
	event *events.Event[*pullreqevents.MergedPayload],
) error {
	return s.addActivity(ctx, event.Payload.PullReqID, event.Payload.PrincipalID,
		func(pr *types.PullReq) types.PullReqActivityPayload {
			return &types.PullRequestActivityPayloadMerge{
				MergeMethod: event.Payload.MergeMethod,
				SHA:         event.Payload.SHA,
			}
		})
}

// addActivity is a utility function that finds pull request,
// updates activity sequence number and stores the new pull request timeline activity.
func (s *Service) addActivity(ctx context.Context,
	prID, principalID int64,
	fn func(pr *types.PullReq) types.PullReqActivityPayload,
) error {
	pr, err := s.pullreqStore.Find(ctx, prID)
	if err != nil {
		return fmt.Errorf("failed to get pull request to add activity: %w", err)
	}

	pr, err = s.pullreqStore.UpdateActivitySeq(ctx, pr)
	if err != nil {
		return fmt.Errorf("failed to increment pull request activity number: %w", err)
	}

	payload := fn(pr)

	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		CreatedBy: principalID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     pr.ActivitySeq,
		SubOrder:  0,
		ReplySeq:  0,
		Type:      payload.ActivityType(),
		Kind:      enum.PullReqActivityKindSystem,
		Text:      "",
	}

	_ = act.SetPayload(payload)

	err = s.activityStore.Create(ctx, act)
	if err != nil {
		return fmt.Errorf("failed to create pull request activity: %w", err)
	}

	return nil
}
