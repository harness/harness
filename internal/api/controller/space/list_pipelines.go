// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.
package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListPipelines lists the pipelines in a space.
func (c *Controller) ListPipelines(ctx context.Context, session *auth.Session,
	spaceRef string, filter *types.PipelineFilter) ([]types.Pipeline, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	count, err := c.pipelineStore.Count(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count child pipelnes: %w", err)
	}

	err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView, true)
	if err != nil {
		return nil, 0, err
	}
	pipelines, err := c.pipelineStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, 0, err
	}

	return pipelines, count, nil
}
