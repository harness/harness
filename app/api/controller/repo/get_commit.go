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

package repo

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// GetCommit gets a repo commit.
func (c *Controller) GetCommit(ctx context.Context,
	session *auth.Session,
	repoRef string,
	sha string,
) (*types.Commit, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoView)
	if err != nil {
		return nil, err
	}

	rpcOut, err := c.git.GetCommit(ctx, &git.GetCommitParams{
		ReadParams: git.CreateReadParams(repo),
		Revision:   sha,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit: %w", err)
	}

	rpcCommit := rpcOut.Commit
	commit, err := controller.MapCommit(&rpcCommit)
	if err != nil {
		return nil, fmt.Errorf("failed to map commit: %w", err)
	}

	return commit, nil
}
