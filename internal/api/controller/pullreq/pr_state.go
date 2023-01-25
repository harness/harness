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

	return nil
}

// State updates the pull request's current state.
//
//nolint:gocognit
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

	event := &pullreqevents.StateChangePayload{
		Base:     eventBase(pr, targetRepo, &session.Principal),
		OldDraft: pr.IsDraft,
		OldState: pr.State,
		NewDraft: in.IsDraft,
		NewState: in.State,
	}

	if pr.State != enum.PullReqStateOpen && in.State == enum.PullReqStateOpen {
		var sourceSHA string

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

		event.SourceSHA = sourceSHA
	}

	pr.State = in.State
	pr.IsDraft = in.IsDraft
	pr.Edited = time.Now().UnixMilli()

	err = c.pullreqStore.Update(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("failed to update pull request: %w", err)
	}

	// Write a row to the pull request activity
	err = c.writeActivity(ctx, pr, getStateActivity(session, pr, in))
	if err != nil {
		// non-critical error
		log.Err(err).Msg("failed to write pull req activity")
	}

	c.eventReporter.StateChange(ctx, event)

	return pr, nil
}

func getStateActivity(session *auth.Session, pr *types.PullReq, in *StateInput) *types.PullReqActivity {
	now := time.Now().UnixMilli()
	act := &types.PullReqActivity{
		ID:         0, // Will be populated in the data layer
		Version:    0,
		CreatedBy:  session.Principal.ID,
		Created:    now,
		Updated:    now,
		Edited:     now,
		Deleted:    nil,
		RepoID:     pr.TargetRepoID,
		PullReqID:  pr.ID,
		Order:      0, // Will be filled in writeActivity
		SubOrder:   0,
		ReplySeq:   0,
		Type:       enum.PullReqActivityTypeStateChange,
		Kind:       enum.PullReqActivityKindSystem,
		Text:       "",
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
	}

	_ = act.SetPayload(&types.PullRequestActivityPayloadStateChange{
		Old:     pr.State,
		New:     in.State,
		IsDraft: in.IsDraft,
		Message: in.Message,
	})

	return act
}
