// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"strings"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	PathName    *string `json:"pathName"`
	SpaceID     *int64  `json:"spaceId"`
	KeepAsAlias bool    `json:"keepAsAlias"`
}

/*
* Move moves a repository to a new space and/or name.
 */
func (c *Controller) Move(ctx context.Context, session *auth.Session,
	repoRef string, in *MoveInput) (*types.Repository, error) {
	repo, err := findRepoFromRef(ctx, c.repoStore, repoRef)
	if err != nil {
		return nil, err
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, enum.PermissionRepoEdit, false); err != nil {
		return nil, err
	}

	// backfill data
	if in.PathName == nil {
		in.PathName = &repo.PathName
	}
	if in.SpaceID == nil {
		in.SpaceID = &repo.SpaceID
	}

	// convert name to lower case for easy of api use
	*in.PathName = strings.ToLower(*in.PathName)

	// verify input
	if err = check.PathName(*in.PathName); err != nil {
		return nil, err
	}

	// ensure it's not a no-op
	if *in.SpaceID == repo.SpaceID && *in.PathName == repo.PathName {
		return nil, err
	}

	// ensure we move to another space
	if *in.SpaceID <= 0 {
		return nil, err
	}

	// Ensure we have access to the target space (if it's a space move)
	if *in.SpaceID != repo.SpaceID {
		var newSpace *types.Space
		newSpace, err = c.spaceStore.Find(ctx, *in.SpaceID)
		if err != nil {
			log.Err(err).Msgf("Failed to get target space with id %d for the move.", *in.SpaceID)

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

	return c.repoStore.Move(ctx, session.Principal.ID, repo.ID, *in.SpaceID, *in.PathName, in.KeepAsAlias)
}
