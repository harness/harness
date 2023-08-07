// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes a pipeline.
func (c *Controller) Delete(ctx context.Context, session *auth.Session, spaceRef string, uid string, n int64) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}

	// TODO: uncomment when soft delete is implemented
	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return err
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineDelete)
	if err != nil {
		return err
	}
	err = c.executionStore.Delete(ctx, pipeline.ID, n)
	if err != nil {
		return fmt.Errorf("could not delete execution: %w", err)
	}
	return nil
}
