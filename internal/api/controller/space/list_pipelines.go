// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package space

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
)

// ListRepositories lists the repositories of a space.
// TODO: move to different file
func (c *Controller) ListPipelines(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.PipelineFilter) ([]types.Pipeline, int, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Add auth
	// if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionRepoView, true); err != nil {
	// 	return nil, 0, err
	// }
	pipelines, err := c.pipelineStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, err
	}
	return pipelines, len(pipelines), nil
}
