// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"context"
	"time"

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

	SourceRepoRef string `json:"sourceRepoRef"`
	SourceBranch  string `json:"sourceBranch"`
	TargetBranch  string `json:"targetBranch"`
}

// Create creates a new repository.
func (c *Controller) Create(ctx context.Context, session *auth.Session, repoRef string, in *CreateInput) (*types.PullReq, error) {
	var pr *types.PullReq
	now := time.Now().UnixMilli()

	err := dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		var (
			sourceRepo *types.Repository
			targetRepo *types.Repository
			err        error
		)

		targetRepo, err = c.repoStore.FindRepoFromRef(ctx, repoRef)
		if err != nil {
			return err
		}

		if in.SourceRepoRef != "" {
			sourceRepo, err = c.repoStore.FindRepoFromRef(ctx, repoRef)
			if err != nil {
				return err
			}
		} else {
			sourceRepo = targetRepo
		}

		if sourceRepo.ID == targetRepo.ID && in.TargetBranch == in.SourceBranch {
			return usererror.BadRequest("target and source branch can't be the same")
		}

		if err = apiauth.CheckRepo(ctx, c.authorizer, session, targetRepo, enum.PermissionRepoEdit, false); err != nil {
			return err
		}

		lastNumber, err := c.pullreqStore.LastNumber(ctx, targetRepo.ID)
		if err != nil {
			return err
		}

		// create new repo object
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

	return pr, nil
}
