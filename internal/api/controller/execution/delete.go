// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package execution

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	pipelineUID string,
	executionNum int64,
) error {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return fmt.Errorf("failed to find repo by ref: %w", err)
	}
	err = apiauth.CheckPipeline(ctx, c.authorizer, session, repo.Path, pipelineUID, enum.PermissionPipelineDelete)
	if err != nil {
		return fmt.Errorf("failed to authorize: %w", err)
	}

	pipeline, err := c.pipelineStore.FindByUID(ctx, repo.ID, pipelineUID)
	if err != nil {
		return fmt.Errorf("failed to find pipeline: %w", err)
	}
	err = c.executionStore.Delete(ctx, pipeline.ID, executionNum)
	if err != nil {
		return fmt.Errorf("could not delete execution: %w", err)
	}
	return nil
}
