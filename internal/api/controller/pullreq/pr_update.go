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

type UpdateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// Update updates an pull request.
//
//nolint:gocognit
func (c *Controller) Update(ctx context.Context,
	session *auth.Session, repoRef string, pullreqNum int64, in *UpdateInput) (*types.PullReq, error) {
	var pr *types.PullReq

	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, usererror.BadRequest("pull request title can't be empty")
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
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

		if pr.Title == in.Title && pr.Description == in.Description {
			return nil
		}

		activity = getUpdateActivity(session, pr, in)

		pr.Title = in.Title
		pr.Description = in.Description
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

func getUpdateActivity(session *auth.Session, pr *types.PullReq, in *UpdateInput) *types.PullReqActivity {
	if pr.Title == in.Title {
		return nil
	}

	now := time.Now().UnixMilli()

	act := &types.PullReqActivity{
		ID:        0, // Will be populated in the data layer
		Version:   0,
		CreatedBy: session.Principal.ID,
		Created:   now,
		Updated:   now,
		Edited:    now,
		Deleted:   nil,
		RepoID:    pr.TargetRepoID,
		PullReqID: pr.ID,
		Order:     0, // Will be filled in writeActivity
		SubOrder:  0,
		ReplySeq:  0,
		Type:      enum.PullReqActivityTypeTitleChange,
		Kind:      enum.PullReqActivityKindSystem,
		Text:      "",
		Payload: map[string]interface{}{
			"old": pr.Title,
			"new": in.Title,
		},
		Metadata:   nil,
		ResolvedBy: nil,
		Resolved:   nil,
	}

	return act
}
