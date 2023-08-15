// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

// CreateBranchInput used for branch creation apis.
type CreateBranchInput struct {
	Name string `json:"name"`

	// Target is the commit (or points to the commit) the new branch will be pointing to.
	// If no target is provided, the branch points to the same commit as the default branch of the repo.
	Target string `json:"target"`
}

// CreateBranch creates a new branch for a repo.
func (c *Controller) CreateBranch(ctx context.Context, session *auth.Session,
	repoRef string, in *CreateBranchInput) (*Branch, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoPush, false); err != nil {
		return nil, err
	}

	// set target to default branch in case no target was provided
	if in.Target == "" {
		in.Target = repo.DefaultBranch
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	rpcOut, err := c.gitRPCClient.CreateBranch(ctx, &gitrpc.CreateBranchParams{
		WriteParams: writeParams,
		BranchName:  in.Name,
		Target:      in.Target,
	})
	if err != nil {
		return nil, err
	}

	branch, err := mapBranch(rpcOut.Branch)
	if err != nil {
		return nil, fmt.Errorf("failed to map branch: %w", err)
	}

	return &branch, nil
}
