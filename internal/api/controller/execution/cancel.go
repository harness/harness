// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/pipeline/checks"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

func (c *Controller) Cancel(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	executionNum int64,
) (*types.Execution, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineExecute)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return nil, fmt.Errorf("failed to find pipeline: %w", err)
	}

	execution, err := c.executionStore.FindByNumber(ctx, pipeline.ID, executionNum)
	if err != nil {
		return nil, fmt.Errorf("failed to find execution %d: %w", executionNum, err)
	}

	err = c.canceler.Cancel(ctx, repo, execution)
	if err != nil {
		return nil, fmt.Errorf("unable to cancel execution: %w", err)
	}

	// Write to the checks store, log and ignore on errors
	err = checks.Write(ctx, c.checkStore, execution, pipeline)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Msg("could not update status check")
	}

	return execution, nil
}
