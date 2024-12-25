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

	apiauth "github.com/harness/gitness/app/api/auth"
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
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	// Trigger permissions are associated with pipeline permissions. If a user has permissions
	// to view the pipeline, they will have permissions to list triggers as well.
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineIdentifier, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize pipeline: %w", err)
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
