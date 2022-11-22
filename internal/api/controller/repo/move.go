// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	UID         *string `json:"uid"`
	ParentID    *int64  `json:"parentId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

/*
* Move moves a repository to a new space and/or uid.
 */
func (c *Controller) Move(ctx context.Context, session *auth.Session,
	repoRef string, in *MoveInput) (*types.Repository, error) {
	repo, err := c.repoStore.FindRepoFromRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	// backfill data
	if in.UID == nil {
		in.UID = &repo.UID
	}
	if in.ParentID == nil {
		in.ParentID = &repo.ParentID
	}

	// verify input
	if err = check.UID(*in.UID); err != nil {
		return nil, err
	}

	// ensure we move to another space
	if *in.ParentID <= 0 {
		return nil, usererror.ErrBadRequest
	}

	// ensure it's not a no-op
	if *in.ParentID == repo.ParentID && *in.UID == repo.UID {
		return nil, usererror.ErrNoChange
	}

	// Ensure we have access to the target space (if it's a space move)
	if *in.ParentID != repo.ParentID {
		var newSpace *types.Space
		newSpace, err = c.spaceStore.Find(ctx, *in.ParentID)
		if err != nil {
			log.Err(err).Msgf("Failed to get target space with id %d for the move.", *in.ParentID)

			return nil, err
		}

		// Ensure we can create repos within the space (using space as scope, similar to create)
		scope := &types.Scope{SpacePath: newSpace.Path}
		resource := &types.Resource{
			Type: enum.ResourceTypeRepo,
			Name: "",
		}
		if err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoCreate); err != nil {
			return nil, err
		}
	}

	return c.repoStore.Move(ctx, session.Principal.ID, repo.ID, *in.ParentID, *in.UID, in.KeepAsAlias)
}
