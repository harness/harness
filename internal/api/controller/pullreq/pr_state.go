// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type StateInput struct {
	State   enum.PullReqState `json:"state"`
	IsDraft bool              `json:"is_draft"`
	Message string            `json:"message"`
}

func (in *StateInput) Check() error {
	state, ok := in.State.Sanitize() // Sanitize will pass through also merged state, so we must check later for it.
	if !ok {
		return usererror.BadRequest(fmt.Sprintf("Allowed states are: %s and %s",
			enum.PullReqStateOpen, enum.PullReqStateClosed))
	}

	in.State = state
	in.Message = strings.TrimSpace(in.Message)

	if in.State == enum.PullReqStateMerged {
		return usererror.BadRequest("Pull requests can't be merged with this API")
	}

	// TODO: Need to check the length of the message string

	return nil
}

// State updates the pull request's current state.
//
//nolint:gocognit,funlen
func (c *Controller) State(ctx context.Context,
	session *auth.Session, repoRef string, pullreqNum int64, in *StateInput,
) (*types.PullReq, error) {
	if err := in.Check(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	sourceRepo := targetRepo
	if pr.SourceRepoID != pr.TargetRepoID {
		sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get source repo by id: %w", err)
		}

		if err = apiauth.CheckRepo(ctx, c.authorizer, session, sourceRepo,
			enum.PermissionRepoView, false); err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	if pr.State == enum.PullReqStateMerged {
		return nil, usererror.BadRequest("Merged pull requests can't be modified.")
	}

	if pr.State == in.State && in.IsDraft == pr.IsDraft {
		return pr, nil // no changes are necessary: state is the same and is_draft hasn't change
	}

	oldState := pr.State
	oldDraft := pr.IsDraft

	type change int
	const (
		changeReopen change = iota + 1
		changeClose
	)

	var sourceSHA string
	var stateChange change

	if pr.State != enum.PullReqStateOpen && in.State == enum.PullReqStateOpen {
		if sourceSHA, err = c.verifyBranchExistence(ctx, sourceRepo, pr.SourceBranch); err != nil {
			return nil, err
		}

		if _, err = c.verifyBranchExistence(ctx, targetRepo, pr.TargetBranch); err != nil {
			return nil, err
		}

		err = c.checkIfAlreadyExists(ctx, pr.TargetRepoID, pr.SourceRepoID, pr.TargetBranch, pr.SourceBranch)
		if err != nil {
			return nil, err
		}

		stateChange = changeReopen
	} else if pr.State == enum.PullReqStateOpen && in.State != enum.PullReqStateOpen {
		stateChange = changeClose
	}

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.State = in.State
		pr.IsDraft = in.IsDraft
		pr.Edited = time.Now().UnixMilli()
		if in.State == enum.PullReqStateClosed {
			// clear all merge (check) related fields
			pr.MergeCheckStatus = enum.MergeCheckStatusUnchecked
			pr.MergeSHA = nil
			pr.MergeConflicts = nil
		}
		pr.ActivitySeq++ // because we need to add the activity entry
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	payload := &types.PullRequestActivityPayloadStateChange{
		Old:      oldState,
		New:      pr.State,
		OldDraft: oldDraft,
		NewDraft: pr.IsDraft,
		Message:  in.Message,
	}
	if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload); errAct != nil {
		// non-critical error
		log.Ctx(ctx).Err(errAct).Msgf("failed to write pull request activity after state change")
	}

	switch stateChange {
	case changeReopen:
		c.eventReporter.Reopened(ctx, &pullreqevents.ReopenedPayload{
			Base:      eventBase(pr, &session.Principal),
			SourceSHA: sourceSHA,
		})
	case changeClose:
		c.eventReporter.Closed(ctx, &pullreqevents.ClosedPayload{
			Base: eventBase(pr, &session.Principal),
		})
	}

	return pr, nil
}
