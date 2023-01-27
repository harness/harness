// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

// PreReceive executes the pre-receive hook for a git repository.
func (c *Controller) PreReceive(
	ctx context.Context,
	session *auth.Session,
	in *types.PreReceiveInput,
) (*types.ServerHookOutput, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, in.RepoID, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	branchOutput := c.blockDefaultBranchDeletion(repo, in)
	if branchOutput != nil {
		return branchOutput, nil
	}

	// TODO: Branch Protection, Block non-brach/tag refs (?), ...

	return &types.ServerHookOutput{}, nil
}

func (c *Controller) blockDefaultBranchDeletion(repo *types.Repository,
	in *types.PreReceiveInput) *types.ServerHookOutput {
	repoDefaultBranchRef := gitReferenceNamePrefixBranch + repo.DefaultBranch

	for _, refUpdate := range in.RefUpdates {
		if refUpdate.New == types.NilSHA && refUpdate.Ref == repoDefaultBranchRef {
			return &types.ServerHookOutput{
				Error: ptr.String(usererror.ErrDefaultBranchCantBeDeleted.Error()),
			}
		}
	}
	return nil
}
