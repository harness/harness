// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Tail(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	executionNum int64,
	stageNum int,
	stepNum int,
) (<-chan *livelog.Line, <-chan error, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineView)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}
	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	execution, err := c.executionStore.Find(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find execution: %w", err)
	}

	stage, err := c.stageStore.FindByNumber(ctx, execution.ID, stageNum)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find stage: %w", err)
	}

	step, err := c.stepStore.FindByNumber(ctx, stage.ID, stepNum)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find step: %w", err)
	}

	linec, errc := c.logStream.Tail(ctx, step.ID)
	return linec, errc, nil
}
