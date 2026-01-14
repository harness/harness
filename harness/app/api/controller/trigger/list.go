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

package trigger

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	filter types.ListQueryFilter,
) ([]*types.Trigger, int64, error) {
	repo, err := c.getRepoCheckPipelineAccess(ctx, session, repoRef, pipelineIdentifier, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, err
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find pipeline: %w", err)
	}

	count, err := c.triggerStore.Count(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count triggers in space: %w", err)
	}

	triggers, err := c.triggerStore.List(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list triggers: %w", err)
	}

	return triggers, count, nil
}
