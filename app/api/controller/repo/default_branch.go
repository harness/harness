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
	"time"

	"github.com/harness/gitness/app/api/controller"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/contextutil"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

type UpdateDefaultBranchInput struct {
	Name string `json:"name"`
}

// TODO: handle the racing condition between update/delete default branch requests for a repo.
func (c *Controller) UpdateDefaultBranch(
	ctx context.Context,
	session *auth.Session,
	repoRef string,
	in *UpdateDefaultBranchInput,
) (*types.Repository, error) {
	repo, err := c.getRepoCheckAccess(ctx, session, repoRef, enum.PermissionRepoEdit, false)
	if err != nil {
		return nil, err
	}

	// the max time we give an update default branch to succeed
	const timeout = 2 * time.Minute

	// lock concurrent requests for updating the default branch of a repo
	// requests will wait for previous ones to compelete before proceed
	unlock, err := c.lockDefaultBranch(
		ctx,
		repo.GitUID,
		in.Name, // branch name only used for logging (lock is on repo)
		timeout+30*time.Second,
	)
	if err != nil {
		return nil, err
	}
	defer unlock()

	writeParams, err := controller.CreateRPCInternalWriteParams(ctx, c.urlProvider, session, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC write params: %w", err)
	}

	// create new, time-restricted context to guarantee update completion, even if request is canceled.
	// TODO: a proper error handling solution required.
	ctx, cancel := context.WithTimeout(
		contextutil.WithNewValues(context.Background(), ctx),
		timeout,
	)
	defer cancel()

	err = c.git.UpdateDefaultBranch(ctx, &git.UpdateDefaultBranchParams{
		WriteParams: writeParams,
		BranchName:  in.Name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update the repo default branch: %w", err)
	}

	repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
		r.DefaultBranch = in.Name
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update the repo default branch on db:%w", err)
	}

	err = c.indexer.Index(ctx, repo)
	if err != nil {
		log.Ctx(ctx).Warn().Err(err).Int64("repo_id", repo.ID).
			Msgf("failed to index repo with the updated default branch %s", in.Name)
	}

	return repo, nil
}
