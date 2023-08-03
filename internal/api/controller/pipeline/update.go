// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pipeline

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
)

// UpdateInput is used for updating a repo.
type UpdateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
	ConfigPath  string `json:"config_path"`
}

// Update updates a repository.
func (c *Controller) Update(ctx context.Context, session *auth.Session,
	spaceRef string, uid string, in *UpdateInput) (*types.Pipeline, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, err
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, err
	}

	if in.Description != "" {
		pipeline.Description = in.Description
	}
	if in.UID != "" {
		pipeline.UID = in.UID
	}
	if in.ConfigPath != "" {
		pipeline.ConfigPath = in.ConfigPath
	}

	// TODO: Add auth
	// if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
	// 	return nil, err
	// }

	return c.pipelineStore.Update(ctx, pipeline)
}
