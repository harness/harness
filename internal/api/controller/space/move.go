// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

// MoveInput is used for moving a space.
type MoveInput struct {
	PathName    *string `json:"pathName"`
	ParentID    *int64  `json:"parentId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

/*
* Move moves a space to a new space and/or name.
 */
func (c *Controller) Move(ctx context.Context, session *auth.Session,
	spaceRef string, in *MoveInput) (*types.Space, error) {
	space, err := findSpaceFromRef(ctx, c.spaceStore, spaceRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckSpace(ctx, c.authorizer, session, space, enum.PermissionSpaceEdit, false); err != nil {
		return nil, err
	}

	// backfill data
	if in.PathName == nil {
		in.PathName = &space.PathName
	}
	if in.ParentID == nil {
		in.ParentID = &space.ParentID
	}

	// convert name to lower case for easy of api use
	*in.PathName = strings.ToLower(*in.PathName)

	// verify input
	if err = check.PathName(*in.PathName); err != nil {
		return nil, err
	}

	// ensure it's not a no-op
	if *in.ParentID == space.ParentID && *in.PathName == space.PathName {
		return nil, err
	}

	// Ensure we can create spaces within the target space (using parent space as scope, similar to create)
	// TODO: restrict top level move
	if *in.ParentID > 0 && *in.ParentID != space.ParentID {
		var newParent *types.Space
		newParent, err = c.spaceStore.Find(ctx, *in.ParentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get target space of move: %w", err)
		}

		scope := &types.Scope{SpacePath: newParent.Path}
		resource := &types.Resource{
			Type: enum.ResourceTypeSpace,
			Name: "",
		}
		if err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionSpaceCreate); err != nil {
			return nil, err
		}

		/*
		 * Validate path length (Due to racing conditions we can't be 100% sure on the path here only best
		 * effort to avoid big transaction failure)
		 * Only needed if we actually change the parent (and can skip top level, as we already validate the name)
		 */
		path := paths.Concatinate(newParent.Path, *in.PathName)
		if err = check.Path(path, true); err != nil {
			return nil, err
		}
	}

	return c.spaceStore.Move(ctx, session.Principal.ID, space.ID, *in.ParentID, *in.PathName, in.KeepAsAlias)
}
