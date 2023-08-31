// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	pagination types.Pagination,
) ([]*types.Execution, int64, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pipeline: %w", err)
	}

	var count int64
	var executions []*types.Execution

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.executionStore.Count(ctx, pipeline.ID)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		executions, err = c.executionStore.List(ctx, pipeline.ID, pagination)
		if err != nil {
			return fmt.Errorf("failed to list child executions: %w", err)
		}

		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return executions, count, fmt.Errorf("failed to fetch list: %w", err)
	}

	return executions, count, nil
}
