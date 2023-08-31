// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TODO: Add more as needed.
type CreateInput struct {
	Description string `json:"description"`
	UID         string `json:"uid"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	in *CreateInput,
) (*types.Trigger, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	// Trigger permissions are associated with pipeline permissions. If a user has permissions
	// to edit the pipeline, they will have permissions to create a trigger as well.
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineEdit)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	now := time.Now().UnixMilli()
	trigger := &types.Trigger{
		Description: in.Description,
		UID:         in.UID,
		PipelineID:  pipeline.ID,
		Created:     now,
		Updated:     now,
		Version:     0,
	}
	err = c.triggerStore.Create(ctx, trigger)
	if err != nil {
		return nil, fmt.Errorf("trigger creation failed: %w", err)
	}

	return trigger, nil
}
