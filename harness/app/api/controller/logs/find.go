// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package logs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Find(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	executionNum int64,
	stageNum int,
	stepNum int,
) ([]*livelog.Line, error) {
	repo, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}

	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineIdentifier, enum.PermissionPipelineView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize pipeline: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
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
	_, err = buf.ReadFrom(rc)
	if err != nil {
		return nil, fmt.Errorf("failed to read from buffer: %w", err)
	}

	err = json.Unmarshal(buf.Bytes(), &lines)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal logs: %w", err)
	}

	return lines, nil
}
