// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type CreateInput struct {
	IsDraft bool `json:"is_draft"`

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

	targetRepo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoPush)
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

	var sourceSHA string

	if sourceSHA, err = c.verifyBranchExistence(ctx, sourceRepo, in.SourceBranch); err != nil {
		return nil, err
	}

	if _, err = c.verifyBranchExistence(ctx, targetRepo, in.TargetBranch); err != nil {
		return nil, err
	}

	if err = c.checkIfAlreadyExists(ctx, targetRepo.ID, sourceRepo.ID, in.TargetBranch, in.SourceBranch); err != nil {
		return nil, err
	}

	mergeBaseResult, err := c.gitRPCClient.MergeBase(ctx, gitrpc.MergeBaseParams{
		ReadParams: gitrpc.ReadParams{RepoUID: sourceRepo.GitUID},
		Ref1:       in.SourceBranch,
		Ref2:       in.TargetBranch,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find merge base: %w", err)
	}

	mergeBaseSHA := mergeBaseResult.MergeBaseSHA

	if mergeBaseSHA == sourceSHA {
		return nil, usererror.BadRequest("The source branch doesn't contain any new commits")
	}

	targetRepo, err = c.repoStore.UpdateOptLock(ctx, targetRepo, func(repo *types.Repository) error {
		repo.PullReqSeq++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to aquire PullReqSeq number: %w", err)
	}

	pr := newPullReq(session, targetRepo.PullReqSeq, sourceRepo, targetRepo, in, sourceSHA, mergeBaseSHA)

	err = c.pullreqStore.Create(ctx, pr)
	if err != nil {
		return nil, fmt.Errorf("pullreq creation failed: %w", err)
	}

	c.eventReporter.Created(ctx, &pullreqevents.CreatedPayload{
		Base:         eventBase(pr, &session.Principal),
		SourceBranch: in.SourceBranch,
		TargetBranch: in.TargetBranch,
		SourceSHA:    sourceSHA,
	})

	return pr, nil
}

// newPullReq creates new pull request object.
func newPullReq(
	session *auth.Session,
	number int64,
	sourceRepo *types.Repository,
	targetRepo *types.Repository,
	in *CreateInput,
	sourceSHA, mergeBaseSHA string,
) *types.PullReq {
	now := time.Now().UnixMilli()
	return &types.PullReq{
		ID:               0, // the ID will be populated in the data layer
		Version:          0,
		Number:           number,
		CreatedBy:        session.Principal.ID,
		Created:          now,
		Updated:          now,
		Edited:           now,
		State:            enum.PullReqStateOpen,
		IsDraft:          in.IsDraft,
		Title:            in.Title,
		Description:      in.Description,
		SourceRepoID:     sourceRepo.ID,
		SourceBranch:     in.SourceBranch,
		SourceSHA:        sourceSHA,
		TargetRepoID:     targetRepo.ID,
		TargetBranch:     in.TargetBranch,
		ActivitySeq:      0,
		MergedBy:         nil,
		Merged:           nil,
		MergeCheckStatus: enum.MergeCheckStatusUnchecked,
		MergeMethod:      nil,
		MergeBaseSHA:     mergeBaseSHA,
		Author:           *session.Principal.ToPrincipalInfo(),
		Merger:           nil,
	}
}
