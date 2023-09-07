// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	executionNum int64,
	stageNum int,
	stepNum int,
) ([]*livelog.Line, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	execution, err := c.executionStore.FindByNumber(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find execution: %w", err)
	}

	stage, err := c.stageStore.FindByNumber(ctx, execution.ID, stageNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find stage: %w", err)
	}

	step, err := c.stepStore.FindByNumber(ctx, stage.ID, stepNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find step: %w", err)
	}

	rc, err := c.logStore.Find(ctx, step.ID)
	if err != nil {
		return nil, fmt.Errorf("could not find logs: %w", err)
	}
	defer rc.Close()

	lines := []*livelog.Line{}
	buf := new(bytes.Buffer)
	buf.ReadFrom(rc)

	err = json.Unmarshal(buf.Bytes(), &lines)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal logs: %w", err)
	}

	return lines, nil
}
