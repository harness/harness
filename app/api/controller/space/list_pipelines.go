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

	apiauth "github.com/harness/gitness/app/api/auth"
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
	filter types.ListQueryFilter,
) ([]*types.Pipeline, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find space: %w", err)
	}
	if err := apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionPipelineView); err != nil {
		return nil, 0, fmt.Errorf("access check failed: %w", err)
	}

	var count int64
	var pipelines []*types.Pipeline

	err = c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.pipelineStore.CountInSpace(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to count pipelines in space: %w", err)
		}

		pipelines, err = c.pipelineStore.ListInSpace(ctx, space.ID, filter)
		if err != nil {
			return fmt.Errorf("failed to list pipelines in space: %w", err)
		}

		return
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return pipelines, count, fmt.Errorf("failed to list pipelines in space: %w", err)
	}

	return pipelines, count, nil
}
