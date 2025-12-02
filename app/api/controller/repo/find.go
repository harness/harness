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

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types/enum"
)

// Find finds a repo.
func (c *Controller) Find(ctx context.Context, session *auth.Session, repoRef string) (*RepositoryOutput, error) {
	// note: can't use c.getRepoCheckAccess because even repositories that are currently being imported can be fetched.
	repoCore, err := c.repoFinder.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repoCore, enum.PermissionRepoView); err != nil {
		return nil, err
	}

	repo, err := c.repoStore.Find(ctx, repoCore.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch repo by ID: %w", err)
	}

	// backfill clone url
	repo.GitURL = c.urlProvider.GenerateGITCloneURL(ctx, repo.Path)
	repo.GitSSHURL = c.urlProvider.GenerateGITCloneSSHURL(ctx, repo.Path)

	repoOut, err := GetRepoOutput(ctx, c.publicAccess, c.repoFinder, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repo output for repo %q: %w", repo.Path, err)
	}

	favoritesMap, err := c.favoriteStore.Map(ctx, session.Principal.ID, enum.ResourceTypeRepo, []int64{repo.ID})
	if err != nil {
		return nil, fmt.Errorf("failed to check if repo %q is marked as favorite: %w", repo.Path, err)
	}
	repoOut.IsFavorite = favoritesMap[repo.ID]

	return repoOut, nil
}
