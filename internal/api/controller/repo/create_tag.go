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
type CreateTagInput struct {
	Name string `json:"name"`
	// Target is the commit (or points to the commit) the new branch will be pointing to.
	// If no target is provided, the branch points to the same commit as the default branch of the repo.
	Target  *string `json:"target"`
	Message *string `json:"message"`
}

// CreateTag creates a new tag for a repo.
func (c *Controller) CreateTag(ctx context.Context, session *auth.Session,
	repoRef string, in *CreateTagInput) (*CommitTag, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoPush, false); err != nil {
		return nil, err
	}

	// set target to default branch in case no branch or commit was provided
	if in.Target == nil || *in.Target == "" {
		in.Target = &repo.DefaultBranch
	}

	writeParams, err := CreateRPCWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	rpcOut, err := c.gitRPCClient.CreateTag(ctx, &gitrpc.CreateTagParams{
		WriteParams: writeParams,
		Name:        in.Name,
		SHA:         *in.Target,
		Message:     *in.Message,
	})

	if err != nil {
		return nil, err
	}
	commitTag, err := mapCommitTag(rpcOut.CommitTag)

	if err != nil {
		return nil, fmt.Errorf("failed to map tag recieved from service output: %w", err)
	}
	return &commitTag, nil
}
