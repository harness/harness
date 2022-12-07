// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
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
) (*types.PullReqInfo, error) {
	now := time.Now().UnixMilli()

	var (
		sourceRepo *types.Repository
		targetRepo *types.Repository
		err        error
	)

	targetRepo, err = c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, targetRepo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	sourceRepo = targetRepo
	if in.SourceRepoRef != "" {
		sourceRepo, err = c.repoStore.FindRepoFromRef(ctx, repoRef)
		if err != nil {
			return nil, err
		}

		if err = apiauth.CheckRepo(ctx, c.authorizer, session, sourceRepo, enum.PermissionRepoView, false); err != nil {
			return nil, err
		}
	}

	if sourceRepo.ID == targetRepo.ID && in.TargetBranch == in.SourceBranch {
		return nil, usererror.BadRequest("target and source branch can't be the same")
	}

	_, err = c.gitRPCClient.GetRef(ctx,
		&gitrpc.GetRefParams{RepoUID: sourceRepo.GitUID, Name: in.SourceBranch, Type: gitrpc.RefTypeBranch})
	if errors.Is(err, gitrpc.ErrNotFound) {
		return nil, usererror.BadRequest(
			fmt.Sprintf("branch %s does not exist in the repository %s", in.SourceBranch, sourceRepo.UID))
	}
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check existence of the branch %s in the repository %s: %w",
			in.SourceBranch, sourceRepo.UID, err)
	}

	_, err = c.gitRPCClient.GetRef(ctx,
		&gitrpc.GetRefParams{RepoUID: targetRepo.GitUID, Name: in.TargetBranch, Type: gitrpc.RefTypeBranch})
	if errors.Is(err, gitrpc.ErrNotFound) {
		return nil, usererror.BadRequest(
			fmt.Sprintf("branch %s does not exist in the repository %s", in.TargetBranch, targetRepo.UID))
	}
	if err != nil {
		return nil, fmt.Errorf(
			"failed to check existence of the branch %s in the repository %s: %w",
			in.TargetBranch, targetRepo.UID, err)
	}

	var pr *types.PullReq

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
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
		}

		return c.pullreqStore.Create(ctx, pr)
	})
	if err != nil {
		return nil, err
	}

	pri := &types.PullReqInfo{
		PullReq:     *pr,
		AuthorID:    session.Principal.ID,
		AuthorName:  session.Principal.DisplayName,
		AuthorEmail: session.Principal.Email,
	}

	return pri, nil
}
