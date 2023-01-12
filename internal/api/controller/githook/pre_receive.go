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

// PreReceiveInput represents the input of the pre-receive git hook.
type PreReceiveInput struct {
	BaseInput
	// RefUpdates contains all references that are being updated as part of the git operation.
	RefUpdates []ReferenceUpdate `json:"ref_updates"`
}

// PreReceive executes the pre-receive hook for a git repository.
func (c *Controller) PreReceive(
	ctx context.Context,
	session *auth.Session,
	in *PreReceiveInput,
) (*ServerHookOutput, error) {
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

	return &ServerHookOutput{}, nil
}

func (c *Controller) blockDefaultBranchDeletion(repo *types.Repository, in *PreReceiveInput) *ServerHookOutput {
	repoDefaultBranchRef := gitReferenceNamePrefixBranch + repo.DefaultBranch

	for _, refUpdate := range in.RefUpdates {
		if refUpdate.New == types.NilSHA && refUpdate.Ref == repoDefaultBranchRef {
			return &ServerHookOutput{
				Error: ptr.String(usererror.ErrDefaultBranchCantBeDeleted.Error()),
			}
		}
	}
	return nil
}
