// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	triggerUID string,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return fmt.Errorf("failed to find parent space: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return fmt.Errorf("failed to find pipeline: %w", err)
	}

	// Trigger permissions are associated with pipeline permissions. If a user has permissions
	// to delete the pipeline, they will have permissions to remove a trigger as well.
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineEdit)
	if err != nil {
		return fmt.Errorf("could not authorize: %w", err)
	}
	err = c.triggerStore.DeleteByUID(ctx, pipeline.ID, triggerUID)
	if err != nil {
		return fmt.Errorf("could not delete trigger: %w", err)
	}
	return nil
}
