// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// UpdateInput is used for updating a trigger.
type UpdateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
}

func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	triggerUID string,
	in *UpdateInput) (*types.Trigger, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipelineUID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to check auth: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	trigger, err := c.triggerStore.FindByUID(ctx, pipeline.ID, triggerUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find trigger: %w", err)
	}

	return c.triggerStore.UpdateOptLock(ctx,
		trigger, func(original *types.Trigger) error {
			// update values only if provided
			if in.Description != "" {
				original.Description = in.Description
			}
			if in.UID != "" {
				original.UID = in.UID
			}
			return nil
		})
}
