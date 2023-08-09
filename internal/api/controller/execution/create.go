// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// TODO: Add more as needed
type CreateInput struct {
	Status string `json:"status"`
}

// Create creates a new execution
func (c *Controller) Create(ctx context.Context, session *auth.Session, spaceRef string, uid string, in *CreateInput) (*types.Execution, error) {
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

	pipeline, err = c.pipelineStore.Increment(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to increment sequence number: %w", err)
	}

	now := time.Now().UnixMilli()
	execution := &types.Execution{
		Number:     pipeline.Seq,
		Status:     in.Status,
		RepoID:     pipeline.RepoID,
		PipelineID: pipeline.ID,
		Created:    now,
		Updated:    now,
		Version:    0,
	}
	err = c.executionStore.Create(ctx, execution)
	if err != nil {
		return nil, fmt.Errorf("execution creation failed: %w", err)
	}

	return execution, nil
}
