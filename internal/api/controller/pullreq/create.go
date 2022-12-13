// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description"`

	SourceRepoRef string `json:"source_repo_ref"`
	SourceBranch  string `json:"source_branch"`
	TargetBranch  string `json:"target_branch"`
}

// Create creates a new pull request.
func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *CreateInput,
) (*types.PullReq, error) {
	now := time.Now().UnixMilli()

	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return nil, usererror.BadRequest("pull request title can't be empty")
	}

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to acquire access access to target repo: %w", err)
	}

	sourceRepo := targetRepo
	if in.SourceRepoRef != "" {
		sourceRepo, err = c.getRepoCheckAccess(ctx, session, in.SourceRepoRef, enum.PermissionRepoView)
		if err != nil {
			return nil, fmt.Errorf("failed to acquire access access to source repo: %w", err)
		}
	}

	if sourceRepo.ID == targetRepo.ID && in.TargetBranch == in.SourceBranch {
		return nil, usererror.BadRequest("target and source branch can't be the same")
	}

	if errBranch := c.verifyBranchExistence(ctx, sourceRepo, in.SourceBranch); errBranch != nil {
		return nil, errBranch
	}
	if errBranch := c.verifyBranchExistence(ctx, targetRepo, in.TargetBranch); errBranch != nil {
		return nil, errBranch
	}

	var pr *types.PullReq

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		var existing int64
		existing, err = c.pullreqStore.Count(ctx, targetRepo.ID, &types.PullReqFilter{
			SourceRepoID: sourceRepo.ID,
			SourceBranch: in.SourceBranch,
			TargetBranch: in.TargetBranch,
			States:       []enum.PullReqState{enum.PullReqStateOpen},
		})
		if err != nil {
			return fmt.Errorf("failed to count existing pull requests: %w", err)
		}

		if existing > 0 {
			return usererror.BadRequest("a pull request for this target and source branch already exists")
		}

		var lastNumber int64

		lastNumber, err = c.pullreqStore.LastNumber(ctx, targetRepo.ID)
		if err != nil {
			return err
		}

		// create new pull request object
		pr = &types.PullReq{
			ID:            0, // the ID will be populated in the data layer
			CreatedBy:     session.Principal.ID,
			Created:       now,
			Updated:       now,
			Number:        lastNumber + 1,
			State:         enum.PullReqStateOpen,
			Title:         in.Title,
			Description:   in.Description,
			SourceRepoID:  sourceRepo.ID,
			SourceBranch:  in.SourceBranch,
			TargetRepoID:  targetRepo.ID,
			TargetBranch:  in.TargetBranch,
			MergedBy:      nil,
			Merged:        nil,
			MergeStrategy: nil,
			Author: types.PrincipalInfo{
				ID:    session.Principal.ID,
				UID:   session.Principal.UID,
				Name:  session.Principal.DisplayName,
				Email: session.Principal.Email,
			},
			Merger: nil,
		}

		return c.pullreqStore.Create(ctx, pr)
	})
	if err != nil {
		return nil, err
	}

	return pr, nil
}
