// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListPipelines lists the pipelines under a repository.
func (c *Controller) ListPipelines(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter types.ListQueryFilter,
) ([]*types.Pipeline, int64, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find repo: %w", err)
	}

	err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionPipelineView, false)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize: %w", err)
	}

	var count int64
	var pipelines []*types.Pipeline

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.pipelineStore.Count(ctx, repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		pipelines, err = c.pipelineStore.List(ctx, repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}
		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return pipelines, count, fmt.Errorf("failed to list pipelines: %w", err)
	}

	return pipelines, count, nil
}
