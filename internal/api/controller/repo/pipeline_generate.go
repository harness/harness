// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// PipelineGenerate returns automatically generate pipeline YAML for a repository.
func (c *Controller) PipelineGenerate(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
) ([]byte, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, true)
	if err != nil {
		return nil, err
	}

	result, err := c.gitRPCClient.GeneratePipeline(ctx, &gitrpc.GeneratePipelineParams{
		ReadParams: CreateRPCReadParams(repo),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate pipeline: %s", err)
	}

	return result.PipelineYAML, nil
}
