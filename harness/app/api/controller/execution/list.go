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

package execution

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	pagination types.Pagination,
) ([]*types.Execution, int64, error) {
	repo, err := c.getRepoCheckPipelineAccess(
		ctx,
		session,
		repoRef,
		pipelineIdentifier,
		enum.PermissionPipelineView,
	)
	if err != nil {
		return nil, 0, err
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pipeline: %w", err)
	}

	var count int64
	var executions []*types.Execution

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
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
