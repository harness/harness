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
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
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

	return c.ListRepositoriesNoAuth(ctx, session.Principal.ID, space.ID, filter)
}

// ListRepositoriesNoAuth list repositories WITHOUT checking for PermissionRepoView.
func (c *Controller) ListRepositoriesNoAuth(
	ctx context.Context,
	principalID int64,
	spaceID int64,
	filter *types.RepoFilter,
) ([]*repoCtrl.RepositoryOutput, int64, error) {
	var (
		repos []*types.Repository
		count int64
	)

	err := c.tx.WithTx(ctx, func(ctx context.Context) (err error) {
		count, err = c.repoStore.Count(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to count child repos for space %d: %w", spaceID, err)
		}

		repos, err = c.repoStore.List(ctx, spaceID, filter)
		if err != nil {
			return fmt.Errorf("failed to list child repos for space %d: %w", spaceID, err)
		}

		return nil
	}, dbtx.TxDefaultReadOnly)
	if err != nil {
		return nil, 0, err
	}

	if len(repos) == 0 {
		return []*repoCtrl.RepositoryOutput{}, 0, nil
	}

	favoritesMap := make(map[int64]bool)
	// We will initialize favoritesMap only in the case when favorites filter is not applied
	// and use the session's principal id in the case to populate the favoritesMap.
	// TODO: [CODE-4005] fix the filters to either add OnlyFavorites as boolean or use OnlyFavoritesFor everywhere.
	if filter.OnlyFavoritesFor == nil {
		// Get repo IDs
		repoIDs := make([]int64, len(repos))
		for i, repo := range repos {
			repoIDs[i] = repo.ID
		}
		// Get favorites
		favoritesMap, err = c.favoriteStore.Map(ctx, principalID, enum.ResourceTypeRepo, repoIDs)
		if err != nil {
			return nil, 0, fmt.Errorf("fetch favorite repos for principal %d failed: %w", principalID, err)
		}
	}

	reposOut := make([]*repoCtrl.RepositoryOutput, 0, len(repos))
	for _, repo := range repos {
		// backfill URLs
		repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
		repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

		repoOut, err := repoCtrl.GetRepoOutput(ctx, c.publicAccess, repo)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get repo %q output: %w", repo.Path, err)
		}

		// We will populate the IsFavorite as true if the favorites filter is applied
		// otherwise take the value out from the favoritesMap.
		repoOut.IsFavorite = filter.OnlyFavoritesFor != nil || favoritesMap[repo.ID]

		reposOut = append(reposOut, repoOut)
	}

	return reposOut, count, nil
}
