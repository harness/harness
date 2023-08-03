// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
)

// Find finds a repo.
func (c *Controller) Find(ctx context.Context, session *auth.Session, spaceRef string, uid string) (*types.Pipeline, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}
	// TODO: Add auth
	// if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceDelete, false); err != nil {
	// 	return err
	// }
	// TODO: uncomment when soft delete is implemented
	return c.pipelineStore.FindByUID(ctx, space.ID, uid)
}
