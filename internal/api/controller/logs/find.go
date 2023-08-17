// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"context"
	"fmt"
	"io"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	pipelineUID string,
	executionNum int64,
	stageNum int,
	stepNum int,
) (io.ReadCloser, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find parent space: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, space.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("could not find pipeline: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, space.Path, pipeline.UID, enum.PermissionPipelineView)
	if err != nil {
		return nil, fmt.Errorf("could not authorize: %w", err)
	}

	execution, err := c.executionStore.Find(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, fmt.Errorf("could not find execution: %w", err)
	}

	stage, err := c.stageStore.FindByNumber(ctx, execution.ID, stageNum)
	if err != nil {
		return nil, fmt.Errorf("could not find stage: %w", err)
	}

	step, err := c.stepStore.FindByNumber(ctx, stage.ID, stepNum)
	if err != nil {
		return nil, fmt.Errorf("could not find step: %w", err)
	}

	return c.logStore.Find(ctx, step.ID)
}
