// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package execution

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
)

// ListRepositories lists the repositories of a space.
// TODO: move to different file
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

	// TODO: Add auth
	// if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionRepoView, true); err != nil {
	// 	return nil, 0, err
	// }
	executions, err := c.executionStore.List(ctx, pipeline.ID, filter)
	if err != nil {
		return nil, 0, err
	}
	return executions, len(executions), nil
}
