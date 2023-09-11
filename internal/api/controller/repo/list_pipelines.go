// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

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
	latest bool,
	filter types.ListQueryFilter,
) ([]*types.Pipeline, int64, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, true)
	if err != nil {
		return nil, 0, err
	}

	var count int64
	var pipelines []*types.Pipeline

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.pipelineStore.Count(ctx, repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		if !latest {
			pipelines, err = c.pipelineStore.List(ctx, repo.ID, filter)
			if err != nil {
				return fmt.Errorf("failed to list pipelines: %w", err)
			}
		} else {
			pipelines, err = c.pipelineStore.ListLatest(ctx, repo.ID, filter)
			if err != nil {
				return fmt.Errorf("failed to list latest pipelines: %w", err)
			}
		}

		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return pipelines, count, fmt.Errorf("failed to list pipelines: %w", err)
	}

	return pipelines, count, nil
}
