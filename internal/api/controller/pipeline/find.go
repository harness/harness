// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Find finds a pipeline.
func (c *Controller) Find(ctx context.Context, session *auth.Session, spaceRef string, uid string) (*types.Pipeline, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, uid, enum.PermissionPipelineView)
	if err != nil {
		return nil, err
	}
	return c.pipelineStore.FindByUID(ctx, space.ID, uid)
}
