// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Status string `json:"status"`
}

// Update updates an execution.
func (c *Controller) Update(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	uid string,
	n int64,
	in *UpdateInput) (*types.Execution, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, err
	}

	execution, err := c.executionStore.Find(ctx, pipeline.ID, n)
	if err != nil {
		return nil, err
	}

	if in.Status != "" {
		execution.Status = in.Status
	}

	// TODO: Add auth
	// if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
	// 	return nil, err
	// }

	return c.executionStore.Update(ctx, execution)
}
