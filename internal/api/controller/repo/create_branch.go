// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// CreateBranchInput used for branch creation apis.
type CreateBranchInput struct {
	Name string `json:"name"`

	// Target is the commit (or points to the commit) the new branch will be pointing to.
	// If no target is provided, the branch points to the same commit as the default branch of the repo.
	Target *string `json:"target"`
}

/*
* Creates a new branch for a repo.
 */
func (c *Controller) CreateBranch(ctx context.Context, session *auth.Session,
	repoRef string, in *CreateBranchInput) (*Branch, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	// set target to default branch in case no target was provided
	if in.Target == nil || *in.Target == "" {
		in.Target = &repo.DefaultBranch
	}

	err = checkBranchName(in.Name)
	if err != nil {
		return nil, err
	}

	rpcOut, err := c.gitRPCClient.CreateBranch(ctx, &gitrpc.CreateBranchParams{
		RepoUID:    repo.GitUID,
		BranchName: in.Name,
		Target:     *in.Target,
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

// checkBranchName does some basic branch validation
// We only ensure there are no control characters, the rest is up to git.
// TODO: Do we need some more validation here?
func checkBranchName(name string) error {
	// fail fast on missing name
	if len(name) == 0 {
		return usererror.ErrBadRequest
	}

	return check.ForControlCharacters(name)
}
