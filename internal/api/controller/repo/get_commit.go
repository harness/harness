// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// GetCommit gets a repo commit.
func (c *Controller) GetCommit(ctx context.Context,
	session *auth.Session,
	repoRef string,
	sha string,
) (*types.Commit, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView, true)
	if err != nil {
		return nil, err
	}

	rpcOut, err := c.gitRPCClient.GetCommit(ctx, &gitrpc.GetCommitParams{
		ReadParams: CreateRPCReadParams(repo),
		SHA:        sha,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit from gitrpc: %w", err)
	}

	rpcCommit := rpcOut.Commit
	commit, err := controller.MapCommit(&rpcCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map commit: %w", err)
	}

	return commit, nil
}
