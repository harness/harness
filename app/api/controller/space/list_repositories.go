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
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// ListRepositories lists the repositories of a space.
func (c *Controller) ListRepositories(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	filter *types.RepoFilter,
) ([]*types.Repository, int64, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionRepoView, true); err != nil {
		return nil, 0, err
	}

	return c.ListRepositoriesNoAuth(ctx, space.ID, filter)
}

// ListRepositoriesNoAuth list repositories WITHOUT checking for PermissionRepoView.
func (c *Controller) ListRepositoriesNoAuth(
	ctx context.Context,
	spaceID int64,
	filter *types.RepoFilter,
) ([]*types.Repository, int64, error) {
	count, err := c.repoStore.Count(ctx, spaceID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count child repos: %w", err)
	}

	repos, err := c.repoStore.List(ctx, spaceID, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list child repos: %w", err)
	}

	// backfill URLs
	for _, repo := range repos {
		repo.GitURL = c.urlProvider.GenerateGITCloneURL(repo.Path)
	}

	return repos, count, nil
}
