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

type SoftDeleteResponse struct {
	DeletedAt int64 `json:"deleted_at"`
}

// SoftDelete marks deleted timestamp for the space and all its subspaces and repositories inside.
func (c *Controller) SoftDelete(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
) (*SoftDeleteResponse, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find space for soft delete: %w", err)
	}

	if err = apiauth.CheckSpace(
		ctx,
		c.authorizer,
		session,
		space,
		enum.PermissionSpaceDelete,
	); err != nil {
		return nil, fmt.Errorf("failed to check access: %w", err)
	}

	return c.SoftDeleteNoAuth(ctx, session, space)
}

// SoftDeleteNoAuth soft deletes the space - no authorization is verified.
// WARNING For internal calls only.
func (c *Controller) SoftDeleteNoAuth(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
) (*SoftDeleteResponse, error) {
	err := c.publicAccess.Delete(ctx, enum.PublicResourceTypeSpace, space.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to delete public access for space: %w", err)
	}

	var softDelRes *SoftDeleteResponse
	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		softDelRes, err = c.softDeleteInnerInTx(ctx, session, space)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to soft delete the space: %w", err)
	}

	return softDelRes, nil
}

func (c *Controller) softDeleteInnerInTx(
	ctx context.Context,
	session *auth.Session,
	space *types.Space,
) (*SoftDeleteResponse, error) {
	_, err := c.spaceStore.FindForUpdate(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock the space for update: %w", err)
	}
	filter := &types.SpaceFilter{
		Page:              1,
		Size:              math.MaxInt,
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.SpaceAttrCreated,
		DeletedBeforeOrAt: nil, // only filter active subspaces
		Recursive:         true,
	}
	subSpaces, err := c.spaceStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list space %d sub spaces recursively: %w", space.ID, err)
	}

	now := time.Now().UnixMilli()

	for _, space := range subSpaces {
		_, err := c.spaceStore.FindForUpdate(ctx, space.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to lock the space for update: %w", err)
		}

		if err := c.spaceStore.SoftDelete(ctx, space, now); err != nil {
			return nil, fmt.Errorf("failed to soft delete subspace: %w", err)
		}
	}

	err = c.softDeleteRepositoriesNoAuth(ctx, session, space.ID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to soft delete repositories of space %d: %w", space.ID, err)
	}

	if err = c.spaceStore.SoftDelete(ctx, space, now); err != nil {
		return nil, fmt.Errorf("spaceStore failed to soft delete space: %w", err)
	}

	err = c.spacePathStore.DeletePathsAndDescendandPaths(ctx, space.ID)
	if err != nil {
		return nil, fmt.Errorf("spacePathStore failed to delete descendant paths of %d: %w", space.ID, err)
	}

	return &SoftDeleteResponse{DeletedAt: now}, nil
}

// softDeleteRepositoriesNoAuth soft deletes all repositories in a space - no authorization is verified.
// WARNING For internal calls only.
func (c *Controller) softDeleteRepositoriesNoAuth(
	ctx context.Context,
	session *auth.Session,
	spaceID int64,
	deletedAt int64,
) error {
	filter := &types.RepoFilter{
		Page:              1,
		Size:              int(math.MaxInt),
		Query:             "",
		Order:             enum.OrderAsc,
		Sort:              enum.RepoAttrNone,
		DeletedBeforeOrAt: nil, // only filter active repos
		Recursive:         true,
	}
	repos, err := c.repoStore.List(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space repositories: %w", err)
	}

	for _, repo := range repos {
		err = c.repoCtrl.SoftDeleteNoAuth(ctx, session, repo, deletedAt)
		if err != nil {
			return fmt.Errorf("failed to soft delete repository: %w", err)
		}
	}
	return nil
}
