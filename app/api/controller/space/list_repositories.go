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
	"github.com/harness/gitness/app/api/controller/repo"
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
) ([]*repo.Repository, int64, error) {
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
		c.publicAccess,
		true,
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
) ([]*repo.Repository, int64, error) {
	var repos []*repo.Repository
	var count int64

	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.repoStore.Count(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child repos: %w", err)
		}

		reposBase, err := c.repoStore.List(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child repos: %w", err)
		}

		for _, repoBase := range reposBase {
			// backfill URLs
			repoBase.GitURL = c.urlProvider.GenerateGITCloneURL(repoBase.Path)

			// backfill public access mode
			isPublic, err := c.publicAccess.Get(ctx,
				&types.PublicResource{
					Type:       enum.PublicResourceTypeRepository,
					ResourceID: repoBase.ID,
				})
			if err != nil {
				return fmt.Errorf("failed to get resource public access mode: %w", err)
			}

			repos = append(repos, &repo.Repository{
				Repository: *repoBase,
				IsPublic:   isPublic,
			})
		}
		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	return repos, count, nil
}
