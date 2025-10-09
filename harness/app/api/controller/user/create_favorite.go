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

package user

import (
	"context"
	"fmt"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (c *Controller) CreateFavorite(
	ctx context.Context,
	session *auth.Session,
	in *types.FavoriteResource,
) (*types.FavoriteResource, error) {
	switch in.Type { // nolint:exhaustive
	case enum.ResourceTypeRepo:
		repo, err := c.repoFinder.FindByID(ctx, in.ID)
		if err != nil {
			return nil, fmt.Errorf("couldn't fetch repo for the user: %w", err)
		}
		if err = apiauth.CheckRepo(
			ctx,
			c.authorizer,
			session,
			repo,
			enum.PermissionRepoView); err != nil {
			return nil, err
		}
		in.ID = repo.ID
	default:
		return nil, fmt.Errorf("resource not onboarded to favorites: %s", in.Type)
	}

	if err := c.favoriteStore.Create(ctx, session.Principal.ID, in); err != nil {
		return nil, fmt.Errorf("failed to mark %s %d as favorite: %w", in.Type, in.ID, err)
	}

	return in, nil
}
