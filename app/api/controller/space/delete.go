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
	"math"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// Delete deletes a space.
func (c *Controller) Delete(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) error {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return err
	}
	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceDelete, false); err != nil {
		return err
	}

	return c.DeleteNoAuth(ctx, session, space.ID)
}

// DeleteNoAuth deletes the space - no authorization is verified.
// WARNING this is meant for internal calls only.
func (c *Controller) DeleteNoAuth(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
) error {
	filter := &types.SpaceFilter{
		Page:  1,
		Size:  math.MaxInt,
		Query: "",
		Order: enum.OrderAsc,
		Sort:  enum.SpaceAttrNone,
	}
	subSpaces, _, err := c.ListSpacesNoAuth(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space %d sub spaces: %w", spaceID, err)
	}
	for _, space := range subSpaces {
		err = c.DeleteNoAuth(ctx, session, space.ID)
		if err != nil {
			return fmt.Errorf("failed to delete space %d: %w", space.ID, err)
		}
	}
	err = c.deleteRepositoriesNoAuth(ctx, session, spaceID)
	if err != nil {
		return fmt.Errorf("failed to delete repositories of space %d: %w", spaceID, err)
	}
	err = c.spaceStore.Delete(ctx, spaceID)
	if err != nil {
		return fmt.Errorf("spaceStore failed to delete space %d: %w", spaceID, err)
	}
	return nil
}

// deleteRepositoriesNoAuth deletes all repositories in a space - no authorization is verified.
// WARNING this is meant for internal calls only.
func (c *Controller) deleteRepositoriesNoAuth(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
) error {
	filter := &types.RepoFilter{
		Page:              1,
		Size:              int(math.MaxInt),
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.RepoAttrNone,
		DeletedBeforeOrAt: nil,
	}

	repos, _, err := c.ListRepositoriesNoAuth(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space repositories: %w", err)
	}

	// TEMPORARY until we support space delete/restore CODE-1413
	recent := time.Now().Add(+time.Hour * 24).UnixMilli()
	filter.DeletedBeforeOrAt = &recent
	alreadyDeletedRepos, _, err := c.ListRepositoriesNoAuth(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list delete repositories for space %d: %w", spaceID, err)
	}
	repos = append(repos, alreadyDeletedRepos...)

	for _, repo := range repos {
		err = c.repoCtrl.PurgeNoAuth(ctx, session, repo)
		if err != nil {
			return fmt.Errorf("failed to delete repository %d: %w", repo.ID, err)
		}
	}
	return nil
}
