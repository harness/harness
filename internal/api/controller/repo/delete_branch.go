// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	"github.com/harness/gitness/gitrpc"
	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types/enum"
)

/*
* DeletePath deletes a repo branch.
 */
func (c *Controller) DeleteBranch(ctx context.Context, session *auth.Session, repoRef string, branchName string) error {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return err
	}

	// make sure user isn't deleting the default branch
	// ASSUMPTION: lower layer calls explicit branch api
	// and 'refs/heads/branch1' would fail if 'branch1' exists.
	// TODO: Add functional test to ensure the scenario is covered!
	if branchName == repo.DefaultBranch {
		return usererror.ErrDefaultBranchCantBeDeleted
	}

	err = c.gitRPCClient.DeleteBranch(ctx, &gitrpc.DeleteBranchParams{
		RepoUID:    repo.GitUID,
		BranchName: branchName,
	})
	if err != nil {
		return err
	}

	return nil
}
