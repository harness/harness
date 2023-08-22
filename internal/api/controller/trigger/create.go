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
	spaceRef string,
	uid string,
	in *CreateInput,
) (*types.Trigger, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("could not find space: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, uid)
	if err != nil {
		return nil, fmt.Errorf("could not find pipeline: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineExecute)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err = c.pipelineStore.IncrementSeqNum(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to increment sequence number: %w", err)
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
