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

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) GetAnalysis(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineIdentifier string,
	executionNum int64,
) (*types.AnalyseExecutionOutput, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find repo %s", repoRef)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineIdentifier, enum.PermissionPipelineView)
	if err != nil {
		return nil, usererror.Forbidden(fmt.Sprintf("not allowed to view pipeline %s", pipelineIdentifier))
	}

	pipeline, err := c.pipelineStore.FindByIdentifier(ctx, repo.ID, pipelineIdentifier)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find pipeline: %s", pipelineIdentifier)
	}

	execution, err := c.executionStore.FindByNumber(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, usererror.BadRequestf("failed to find execution %d", executionNum)
	}

	if execution.Status == enum.CIStatusSuccess {
		return nil, usererror.BadRequestf("execution %d is not a failed execution", executionNum)
	}

	return &types.AnalyseExecutionOutput{Yaml: "a:1"}, nil
}
