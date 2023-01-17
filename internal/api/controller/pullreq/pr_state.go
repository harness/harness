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
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type StateInput struct {
	State   enum.PullReqState `json:"state"`
	IsDraft bool              `json:"is_draft"`
	Message string            `json:"message"`
}

// State updates the pull request's current state.
//
//nolint:gocognit
func (c *Controller) State(ctx context.Context,
	session *auth.Session, repoRef string, pullreqNum int64, in *StateInput) (*types.PullReq, error) {
	var pr *types.PullReq

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	state, ok := in.State.Sanitize()
	if !ok {
		return nil, usererror.BadRequest(fmt.Sprintf("Allowed states are: %s and %s",
			enum.PullReqStateOpen, enum.PullReqStateClosed))
	}

	in.State = state
	in.Message = strings.TrimSpace(in.Message)

	if in.State == enum.PullReqStateMerged {
		return nil, usererror.BadRequest("Pull requests can't be merged with this API")
	}

	var activity *types.PullReqActivity

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		pr, err = c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
		if err != nil {
			return fmt.Errorf("failed to get pull request by number: %w", err)
		}

		if pr.SourceRepoID != pr.TargetRepoID {
			var sourceRepo *types.Repository

			sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
			if err != nil {
				return fmt.Errorf("failed to get source repo by id: %w", err)
			}

			if err = apiauth.CheckRepo(ctx, c.authorizer, session, sourceRepo,
				enum.PermissionRepoView, false); err != nil {
				return fmt.Errorf("failed to acquire access to source repo: %w", err)
			}
		}

		if pr.State == enum.PullReqStateMerged {
			return usererror.BadRequest("Merged pull requests can't be modified.")
		}

		if pr.State == in.State && in.IsDraft == pr.IsDraft {
			return nil // no changes are necessary: state is the same and is_draft hasn't change
		}

		if pr.State != enum.PullReqStateOpen && in.State == enum.PullReqStateOpen {
			err = c.checkIfAlreadyExists(ctx, pr.TargetRepoID, pr.SourceRepoID, pr.TargetBranch, pr.SourceBranch)
			if err != nil {
				return err
			}
		}

		activity = getStateActivity(session, pr, in)

		pr.State = in.State
		pr.IsDraft = in.IsDraft
		pr.Edited = time.Now().UnixMilli()

		err = c.pullreqStore.Update(ctx, pr)
		if err != nil {
			return fmt.Errorf("failed to update pull request: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Write a row to the pull request activity
	if activity != nil {
		err = c.writeActivity(ctx, pr, activity)
		if err != nil {
			// non-critical error
			log.Err(err).Msg("failed to write pull req activity")
		}
	}

	return pr, nil
}

func getStateActivity(session *auth.Session, pr *types.PullReq, in *StateInput) *types.PullReqActivity {
	now := time.Now().UnixMilli()
	payload := map[string]interface{}{
		"old":      pr.State,
		"new":      in.State,
		"is_draft": in.IsDraft,
	}
	if len(in.Message) != 0 {
		payload["message"] = in.Message
	}

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
		Payload:    payload,
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
	}

	return act
}
