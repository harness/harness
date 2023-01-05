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

	existing, err := c.pullreqStore.Count(ctx, targetRepo.ID, &types.PullReqFilter{
		SourceRepoID: sourceRepo.ID,
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		States:       []enum.PullReqState{enum.PullReqStateOpen},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count existing pull requests: %w", err)
	}
	if existing > 0 {
		return nil, usererror.BadRequest("a pull request for this target and source branch already exists")
	}

	targetRepo, err = c.repoStore.UpdateOptLock(ctx, targetRepo, func(repo *types.Repository) error {
		repo.PullReqSeq++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to aquire PullReqSeq number: %w", err)
	}

	pr := newPullReq(session, targetRepo.PullReqSeq, sourceRepo, targetRepo, in)

	err = c.pullreqStore.Create(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("pullreq creation failed: %w", err)
	}

	return pr, nil
}

// newPullReq creates new pull request object.
func newPullReq(session *auth.Session, number int64,
	sourceRepo, targetRepo *types.Repository, in *CreateInput) *types.PullReq {
	now := time.Now().UnixMilli()
	return &types.PullReq{
		ID:            0, // the ID will be populated in the data layer
		Version:       0,
		Number:        number,
		CreatedBy:     session.Principal.ID,
		Created:       now,
		Updated:       now,
		Edited:        now,
		State:         enum.PullReqStateOpen,
		Title:         in.Title,
		Description:   in.Description,
		SourceRepoID:  sourceRepo.ID,
		SourceBranch:  in.SourceBranch,
		TargetRepoID:  targetRepo.ID,
		TargetBranch:  in.TargetBranch,
		ActivitySeq:   0,
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
}
