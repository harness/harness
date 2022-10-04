// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
	"github.com/rs/zerolog/log"
)

type CreateInput struct {
	PathName    string `json:"pathName"`
	SpaceID     int64  `json:"spaceId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPublic    bool   `json:"isPublic"`
	ForkID      int64  `json:"forkId"`
}

/*
 * Create creates a new repository.
 */
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Repository, error) {
	log := log.Ctx(ctx)
	// ensure we reference a space
	if in.SpaceID <= 0 {
		return nil, usererror.BadRequest("A repository can't exist by itself.")
	}

	parentSpace, err := c.spaceStore.Find(ctx, in.SpaceID)
	if err != nil {
		log.Err(err).Msgf("Failed to get space with id '%d'.", in.SpaceID)

		return nil, usererror.BadRequest("Parent not found'")
	}

	/*
	 * AUTHORIZATION
	 * Create is a special case - check permission without specific resource
	 */
	scope := &types.Scope{SpacePath: parentSpace.Path}
	resource := &types.Resource{
		Type: enum.ResourceTypeRepo,
		Name: "",
	}

	err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionRepoCreate)
	if err != nil {
		return nil, fmt.Errorf("auth check failed: %w", err)
	}

	// create new repo object
	repo := &types.Repository{
		PathName:    strings.ToLower(in.PathName),
		SpaceID:     in.SpaceID,
		Name:        in.Name,
		Description: in.Description,
		IsPublic:    in.IsPublic,
		CreatedBy:   session.Principal.ID,
		Created:     time.Now().UnixMilli(),
		Updated:     time.Now().UnixMilli(),
		ForkID:      in.ForkID,
	}

	// validate repo
	if err = check.Repo(repo); err != nil {
		return nil, err
	}

	// create in store
	err = c.repoStore.Create(ctx, repo)
	if err != nil {
		log.Error().Err(err).
			Msg("Repository creation failed.")

		return nil, err
	}

	return repo, nil
}
