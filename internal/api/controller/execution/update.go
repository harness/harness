// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type UpdateInput struct {
	Status string `json:"status"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	executionNum int64,
	in *UpdateInput) (*types.Execution, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	execution, err := c.executionStore.Find(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	return c.executionStore.UpdateOptLock(ctx,
		execution, func(original *types.Execution) error {
			// update values only if provided
			if in.Status != "" {
				original.Status = in.Status
			}

			return nil
		})
}
