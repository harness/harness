// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package execution

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// List lists the executions in a pipeline.
func (c *Controller) List(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	filter *types.ExecutionFilter) ([]types.Execution, int, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}
	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return nil, 0, err
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineView)
	if err != nil {
		return nil, 0, err
	}

	executions, err := c.executionStore.List(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, err
	}
	// TODO: This should be total count, not returned count
	return executions, len(executions), nil
}
