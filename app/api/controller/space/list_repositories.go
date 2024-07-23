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

package space

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	repoCtrl "github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListRepositories lists the repositories of a space.
func (c *Controller) ListRepositories(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter *types.RepoFilter,
) ([]*repoCtrl.RepositoryOutput, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpaceScope(
		ctx,
		c.authorizer,
		session,
		space,
		enum.ResourceTypeRepo,
		enum.PermissionRepoView,
	); err != nil {
		return nil, 0, err
	}

	return c.ListRepositoriesNoAuth(ctx, space.ID, filter)
}

// ListRepositoriesNoAuth list repositories WITHOUT checking for PermissionRepoView.
func (c *Controller) ListRepositoriesNoAuth(
	ctx context.Context,
	spaceID int64,
	filter *types.RepoFilter,
) ([]*repoCtrl.RepositoryOutput, int64, error) {
	var repos []*types.Repository
	var count int64

	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.repoStore.Count(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child repos: %w", err)
		}

		repos, err = c.repoStore.List(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child repos: %w", err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	var reposOut []*repoCtrl.RepositoryOutput
	for _, repo := range repos {
		// backfill URLs
		repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
		repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

		repoOut, err := repoCtrl.GetRepoOutput(ctx, c.publicAccess, repo)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get repo %q output: %w", repo.Path, err)
		}

		reposOut = append(reposOut, repoOut)
	}

	return reposOut, count, nil
}
