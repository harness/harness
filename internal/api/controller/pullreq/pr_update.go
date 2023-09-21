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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type UpdateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

func (in *UpdateInput) Check() error {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return usererror.BadRequest("pull request title can't be empty")
	}

	in.Description = strings.TrimSpace(in.Description)

	// TODO: Check the length of the input strings

	return nil
}

// Update updates an pull request.
func (c *Controller) Update(ctx context.Context,
	session *auth.Session, repoRef string, pullreqNum int64, in *UpdateInput,
) (*types.PullReq, error) {
	if err := in.Check(); err != nil {
		return nil, err
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access to target repo: %w", err)
	}

	pr, err := c.pullreqStore.FindByNumber(ctx, targetRepo.ID, pullreqNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull request by number: %w", err)
	}

	if pr.SourceRepoID != pr.TargetRepoID {
		var sourceRepo *types.Repository

		sourceRepo, err = c.repoStore.Find(ctx, pr.SourceRepoID)
		if err != nil {
			return nil, fmt.Errorf("failed to get source repo by id: %w", err)
		}

		if err = apiauth.CheckRepo(ctx, c.authorizer, session, sourceRepo,
			enum.PermissionRepoView, false); err != nil {
			return nil, fmt.Errorf("failed to acquire access to source repo: %w", err)
		}
	}

	if pr.Title == in.Title && pr.Description == in.Description {
		return pr, nil
	}

	needToWriteActivity := in.Title != pr.Title
	oldTitle := pr.Title

	pr, err = c.pullreqStore.UpdateOptLock(ctx, pr, func(pr *types.PullReq) error {
		pr.Title = in.Title
		pr.Description = in.Description
		pr.Edited = time.Now().UnixMilli()
		if needToWriteActivity {
			pr.ActivitySeq++
		}
		return nil
	})
	if err != nil {
		return pr, fmt.Errorf("failed to update pull request: %w", err)
	}

	if needToWriteActivity {
		payload := &types.PullRequestActivityPayloadTitleChange{
			Old: oldTitle,
			New: pr.Title,
		}
		if _, errAct := c.activityStore.CreateWithPayload(ctx, pr, session.Principal.ID, payload); errAct != nil {
			// non-critical error
			log.Ctx(ctx).Err(errAct).Msgf("failed to write pull request activity after title change")
		}
	}

	if err = c.sseStreamer.Publish(ctx, targetRepo.ParentID, enum.SSETypePullrequesUpdated, pr); err != nil {
		log.Ctx(ctx).Warn().Msg("failed to publish PR changed event")
	}

	return pr, nil
}
