// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package githook

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/githook"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/gotidy/ptr"
)

// PreReceive executes the pre-receive hook for a git repository.
//
//nolint:revive // not yet fully implemented
func (c *Controller) PreReceive(
	ctx context.Context,
	session *auth.Session,
	repoID int64,
	principalID int64,
	in *githook.PreReceiveInput,
) (*githook.Output, error) {
	if in == nil {
		return nil, fmt.Errorf("input is nil")
	}

	repo, err := c.getRepoCheckAccess(ctx, session, repoID, enum.PermissionRepoEdit)
	if err != nil {
		return nil, err
	}

	branchOutput := c.blockDefaultBranchDeletion(repo, in)
	if branchOutput != nil {
		return branchOutput, nil
	}

	// TODO: Branch Protection, Block non-brach/tag refs (?), ...

	return &githook.Output{}, nil
}

func (c *Controller) blockDefaultBranchDeletion(repo *types.Repository,
	in *githook.PreReceiveInput) *githook.Output {
	repoDefaultBranchRef := gitReferenceNamePrefixBranch + repo.DefaultBranch

	for _, refUpdate := range in.RefUpdates {
		if refUpdate.New == types.NilSHA && refUpdate.Ref == repoDefaultBranchRef {
			return &githook.Output{
				Error: ptr.String(usererror.ErrDefaultBranchCantBeDeleted.Error()),
			}
		}
	}
	return nil
}
