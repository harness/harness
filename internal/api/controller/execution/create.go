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

// TODO: Add more as needed.
type CreateInput struct {
	Status string `json:"status"`
}

func (c *Controller) Create(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	in *CreateInput,
) (*types.Execution, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path,
		pipelineUID, enum.PermissionPipelineExecute)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	pipeline, err = c.pipelineStore.IncrementSeqNum(ctx, pipeline)
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
