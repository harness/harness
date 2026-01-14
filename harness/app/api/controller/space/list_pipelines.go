// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package space

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListPipelines lists the pipelines in a space.
func (c *Controller) ListPipelines(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter types.ListPipelinesFilter,
) ([]*types.Pipeline, int64, error) {
	space, err := c.getSpaceCheckAuth(ctx, session, spaceRef, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to acquire access to space: %w", err)
	}

	var count int64
	var pipelines []*types.Pipeline

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		pipelines, err = c.pipelineStore.ListInSpace(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list pipelines in space: %w", err)
		}

		pipelineIDs := make([]int64, len(pipelines))
		for i, pipeline := range pipelines {
			pipelineIDs[i] = pipeline.ID
		}
		execs, err := c.executionStore.ListByPipelineIDs(ctx, pipelineIDs, filter.LastExecutions)
		if err != nil {
			return fmt.Errorf("failed to list executions by pipeline IDs: %w", err)
		}
		for _, pipeline := range pipelines {
			pipeline.LastExecutions = execs[pipeline.ID]
		}

		if filter.Page == 1 && len(pipelines) < filter.Size {
			count = int64(len(pipelines))
			return nil
		}

		count, err = c.pipelineStore.CountInSpace(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count pipelines in space: %w", err)
		}

		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return pipelines, count, fmt.Errorf("failed to list pipelines in space: %w", err)
	}

	return pipelines, count, nil
}
