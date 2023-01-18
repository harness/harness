// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package space

import (
	"context"
	"fmt"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/api/usererror"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"
)

var (
	errParentIDNegative = usererror.BadRequest(
		"Parent ID has to be either zero for a root space or greater than zero for a child space.")
)

type CreateInput struct {
	ParentID    int64  `json:"parent_id"`
	UID         string `json:"uid"`
	Description string `json:"description"`
	IsPublic    bool   `json:"is_public"`
}

// Create creates a new space.
func (c *Controller) Create(ctx context.Context, session *auth.Session, in *CreateInput) (*types.Space, error) {
	if err := c.checkAuthSpaceCreation(ctx, session, in.ParentID); err != nil {
		return nil, err
	}

	if err := c.sanitizeCreateInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	var space *types.Space
	err := dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		spacePath := in.UID
		if in.ParentID > 0 {
			// lock parent space path to ensure it doesn't get updated while we setup new space
			parentPath, err := c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeSpace, in.ParentID)
			if err != nil {
				return usererror.BadRequest("Parent not found")
			}
			spacePath = paths.Concatinate(parentPath.Value, in.UID)

			// ensure path is within accepted depth!
			err = check.PathDepth(spacePath, true)
			if err != nil {
				return fmt.Errorf("path is invalid: %w", err)
			}
		}

		now := time.Now().UnixMilli()
		space = &types.Space{
			Version:     0,
			ParentID:    in.ParentID,
			UID:         in.UID,
			Path:        spacePath,
			Description: in.Description,
			IsPublic:    in.IsPublic,
			CreatedBy:   session.Principal.ID,
			Created:     now,
			Updated:     now,
		}
		err := c.spaceStore.Create(ctx, space)
		if err != nil {
			return fmt.Errorf("space creation failed: %w", err)
		}

		path := &types.Path{
			Version:    0,
			Value:      space.Path,
			IsPrimary:  true,
			TargetType: enum.PathTargetTypeSpace,
			TargetID:   space.ID,
			CreatedBy:  space.CreatedBy,
			Created:    space.Created,
			Updated:    space.Updated,
		}
		err = c.pathStore.Create(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return space, nil
}

func (c *Controller) checkAuthSpaceCreation(ctx context.Context, session *auth.Session, parentID int64) error {
	if parentID <= 0 {
		// TODO: Restrict top level space creation.
		if session == nil {
			return usererror.ErrUnauthorized
		}

		return nil
	}

	parentSpace, err := c.spaceStore.Find(ctx, parentID)
	if err != nil {
		return fmt.Errorf("failed to get parent space: %w", err)
	}

	// create is a special case - check permission without specific resource
	scope := &types.Scope{SpacePath: parentSpace.Path}
	resource := &types.Resource{
		Type: enum.ResourceTypeSpace,
		Name: "",
	}
	if err = apiauth.Check(ctx, c.authorizer, session, scope, resource, enum.PermissionSpaceCreate); err != nil {
		return err
	}

	return nil
}

func (c *Controller) sanitizeCreateInput(in *CreateInput) error {
	if in.ParentID < 0 {
		return errParentIDNegative
	}

	if err := c.uidCheck(in.UID, in.ParentID == 0); err != nil {
		return err
	}

	in.Description = strings.TrimSpace(in.Description)
	if err := check.Description(in.Description); err != nil {
		return err
	}

	return nil
}
