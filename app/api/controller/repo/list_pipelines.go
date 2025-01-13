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

package repo

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListPipelines lists the pipelines under a repository.
func (c *Controller) ListPipelines(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	filter *types.ListPipelinesFilter,
) ([]*types.Pipeline, int64, error) {
	repo, err := GetRepo(ctx, c.repoFinder, repoRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find repo: %w", err)
	}

	if err := apiauth.CheckRepoState(ctx, session, repo, enum.PermissionPipelineView); err != nil {
		return nil, 0, err
	}

	if err := apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, "", enum.PermissionPipelineView); err != nil {
		return nil, 0, fmt.Errorf("access check failed: %w", err)
	}

	var count int64
	var pipelines []*types.Pipeline

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.pipelineStore.Count(ctx, repo.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child executions: %w", err)
		}

		if !filter.Latest {
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

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return pipelines, count, fmt.Errorf("failed to list pipelines: %w", err)
	}

	return pipelines, count, nil
}
