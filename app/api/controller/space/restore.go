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
	"errors"
	"fmt"
	"math"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/api/controller/limiter"
	"github.com/harness/gitness/app/api/usererror"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

type RestoreInput struct {
	NewIdentifier *string `json:"new_identifier,omitempty"`
	NewParentRef  *string `json:"new_parent_ref,omitempty"` // Reference of the new parent space
}

var errSpacePathInvalid = usererror.BadRequest("Space ref or identifier is invalid.")

func (c *Controller) Restore(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	deletedAt int64,
	in *RestoreInput,
) (*SpaceOutput, error) {
	if err := c.sanitizeRestoreInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize restore input: %w", err)
	}

	space, err := c.spaceStore.FindByRefAndDeletedAt(ctx, spaceRef, deletedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to find the space: %w", err)
	}

	// check view permission on the original ref.
	err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceView)
	if err != nil {
		return nil, fmt.Errorf("failed to authorize on space restore: %w", err)
	}

	parentSpace, err := c.getParentSpace(ctx, space, in.NewParentRef)
	if err != nil {
		return nil, fmt.Errorf("failed to get space parent: %w", err)
	}

	// check create permissions within the parent space scope.
	if err = apiauth.CheckSpaceScope(
		ctx,
		c.authorizer,
		session,
		parentSpace,
		enum.ResourceTypeSpace,
		enum.PermissionSpaceEdit,
	); err != nil {
		return nil, fmt.Errorf("authorization failed on space restore: %w", err)
	}

	spacePath := paths.Concatenate(parentSpace.Path, space.Identifier)
	if in.NewIdentifier != nil {
		spacePath = paths.Concatenate(parentSpace.Path, *in.NewIdentifier)
	}

	err = c.tx.WithTx(ctx, func(ctx context.Context) error {
		space, err = c.restoreSpaceInnerInTx(
			ctx,
			space,
			deletedAt,
			in.NewIdentifier,
			&parentSpace.ID,
			spacePath)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("failed to restore space in a tnx: %w", err)
	}

	// restored spaces will be private since public access data has deleted upon deletion.
	return &SpaceOutput{
		Space:    *space,
		IsPublic: false,
	}, nil
}

func (c *Controller) restoreSpaceInnerInTx(
	ctx context.Context,
	space *types.Space,
	deletedAt int64,
	newIdentifier *string,
	newParentID *int64,
	spacePath string,
) (*types.Space, error) {
	// restore the target space
	restoredSpace, err := c.restoreNoAuth(ctx, space, newIdentifier, newParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to restore space: %w", err)
	}

	if err = check.PathDepth(restoredSpace.Path, true); err != nil {
		return nil, fmt.Errorf("path is invalid: %w", err)
	}

	repoCount, err := c.repoStore.Count(
		ctx,
		space.ID,
		&types.RepoFilter{DeletedAt: &deletedAt, Recursive: true},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to count repos in space recursively: %w", err)
	}

	if err := c.resourceLimiter.RepoCount(ctx, space.ID, int(repoCount)); err != nil {
		return nil, fmt.Errorf("resource limit exceeded: %w", limiter.ErrMaxNumReposReached)
	}

	filter := &types.SpaceFilter{
		Page:      1,
		Size:      math.MaxInt,
		Query:     "",
		Order:     enum.OrderDesc,
		Sort:      enum.SpaceAttrCreated,
		DeletedAt: &deletedAt,
		Recursive: true,
	}
	subSpaces, err := c.spaceStore.List(ctx, space.ID, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list space sub spaces recursively: %w", err)
	}

	var subspacePath string
	for _, subspace := range subSpaces {
		// check the path depth before restore nested subspaces.
		subspacePath = subspace.Path[len(space.Path):]

		if err = check.PathDepth(paths.Concatenate(spacePath, subspacePath), true); err != nil {
			return nil, fmt.Errorf("path is invalid: %w", err)
		}

		// identifier and parent ID of sub spaces shouldn't change.
		_, err = c.restoreNoAuth(ctx, subspace, nil, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to restore subspace: %w", err)
		}
	}

	if err := c.restoreRepositoriesNoAuth(ctx, space.ID, deletedAt); err != nil {
		return nil, fmt.Errorf("failed to restore space repositories: %w", err)
	}

	return restoredSpace, nil
}

func (c *Controller) restoreNoAuth(
	ctx context.Context,
	space *types.Space,
	newIdentifier *string,
	newParentID *int64,
) (*types.Space, error) {
	space, err := c.spaceStore.Restore(ctx, space, newIdentifier, newParentID)
	if err != nil {
		return nil, fmt.Errorf("failed to restore the space: %w", err)
	}

	now := time.Now().UnixMilli()
	pathSegment := &types.SpacePathSegment{
		Identifier: space.Identifier,
		IsPrimary:  true,
		SpaceID:    space.ID,
		ParentID:   space.ParentID,
		CreatedBy:  space.CreatedBy,
		Created:    now,
		Updated:    now,
	}
	err = c.spacePathStore.InsertSegment(ctx, pathSegment)
	if errors.Is(err, store.ErrDuplicate) {
		return nil, usererror.BadRequest(fmt.Sprintf("A primary path already exists for %s.",
			space.Identifier))
	}
	if err != nil {
		return nil, fmt.Errorf("failed to insert space path on restore: %w", err)
	}

	return space, nil
}

func (c *Controller) restoreRepositoriesNoAuth(
	ctx context.Context,
	spaceID int64,
	deletedAt int64,
) error {
	filter := &types.RepoFilter{
		Page:      1,
		Size:      int(math.MaxInt),
		Query:     "",
		Order:     enum.OrderAsc,
		Sort:      enum.RepoAttrNone,
		DeletedAt: &deletedAt,
		Recursive: true,
	}
	repos, err := c.repoStore.List(ctx, spaceID, filter)
	if err != nil {
		return fmt.Errorf("failed to list space repositories: %w", err)
	}

	for _, repo := range repos {
		_, err = c.repoCtrl.RestoreNoAuth(ctx, repo, nil, repo.ParentID)
		if err != nil {
			return fmt.Errorf("failed to restore repository: %w", err)
		}
	}
	return nil
}

func (c *Controller) getParentSpace(
	ctx context.Context,
	space *types.Space,
	newParentRef *string,
) (*types.Space, error) {
	var parentSpace *types.Space
	var err error

	if newParentRef == nil {
		if space.ParentID == 0 {
			return &types.Space{}, nil
		}

		parentSpace, err = c.spaceStore.Find(ctx, space.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to find the space parent %d", space.ParentID)
		}

		return parentSpace, nil
	}

	// the provided new reference for space parent must exist.
	parentSpace, err = c.spaceStore.FindByRef(ctx, *newParentRef)
	if err != nil {
		return nil, fmt.Errorf("failed to find the parent space by ref '%s': %w - returning usererror %w",
			*newParentRef, err, errSpacePathInvalid)
	}

	return parentSpace, nil
}

func (c *Controller) sanitizeRestoreInput(in *RestoreInput) error {
	if in.NewParentRef == nil {
		return nil
	}

	if len(*in.NewParentRef) > 0 && !c.nestedSpacesEnabled {
		return errNestedSpacesNotSupported
	}

	return nil
}
