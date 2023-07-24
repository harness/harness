// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package repo

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	apiauth "github.com/harness/gitness/internal/api/auth"
	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// MoveInput is used for moving a repo.
type MoveInput struct {
	UID         *string `json:"uid"`
	ParentRef   *string `json:"parent_ref"`
	KeepAsAlias bool    `json:"keep_as_alias"`
}

func (i *MoveInput) hasChanges(repo *types.Repository) bool {
	if i.UID != nil && *i.UID != repo.UID {
		return true
	}

	if i.ParentRef != nil {
		parentRefAsID, err := strconv.ParseInt(*i.ParentRef, 10, 64)
		// if parsing was successful, user provided actual space id
		if err == nil && parentRefAsID != repo.ParentID {
			return true
		}

		// if parsing was unsucessful, user provided input as path
		if err != nil {
			// repo is an existing entity, assume that path is not empty and thus no error
			repoParentPath, _, _ := paths.DisectLeaf(repo.Path)
			parentRefAsPath := *i.ParentRef
			if parentRefAsPath != repoParentPath {
				return true
			}
		}
	}

	return false
}

// Move moves a repository to a new space and/or uid.
//
//nolint:gocognit // refactor if needed
func (c *Controller) Move(ctx context.Context, session *auth.Session,
	repoRef string, in *MoveInput) (*types.Repository, error) {
	repo, err := c.repoStore.FindByRef(ctx, repoRef)
	if err != nil {
		return nil, err
	}

	permission := enum.PermissionRepoEdit
	var inParentSpaceID *int64
	if in.ParentRef != nil {
		// ensure user has access to new space (parentId not sanitized!)
		inParentSpace, err := c.getSpaceCheckAuthRepoCreation(ctx, session, *in.ParentRef)
		if err != nil {
			return nil, fmt.Errorf("failed to verify repo creation permissions on new parent space: %w", err)
		}

		inParentSpaceID = &inParentSpace.ID

		if inParentSpace.ID != repo.ParentID {
			// TODO: what would be correct permissions on repo? (technically we are deleting it from the old space)
			permission = enum.PermissionRepoDelete
		}
	}

	if err = apiauth.CheckRepo(ctx, c.authorizer, session, repo, permission, false); err != nil {
		return nil, err
	}

	if !in.hasChanges(repo) {
		return repo, nil
	}

	if err = c.sanitizeMoveInput(in); err != nil {
		return nil, fmt.Errorf("failed to sanitize input: %w", err)
	}

	err = dbtx.New(c.db).WithTx(ctx, func(ctx context.Context) error {
		repo, err = c.repoStore.UpdateOptLock(ctx, repo, func(r *types.Repository) error {
			if in.UID != nil {
				r.UID = *in.UID
			}
			if inParentSpaceID != nil {
				r.ParentID = *inParentSpaceID
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to update repo: %w", err)
		}

		// lock path to ensure it doesn't get updated while we move the repo
		var primaryPath *types.Path
		primaryPath, err = c.pathStore.FindPrimaryWithLock(ctx, enum.PathTargetTypeRepo, repo.ID)
		if err != nil {
			return fmt.Errorf("failed to find primary path: %w", err)
		}

		var parentPath *types.Path
		parentPath, err = c.pathStore.FindPrimary(ctx, enum.PathTargetTypeSpace, repo.ParentID)
		if err != nil {
			return fmt.Errorf("failed to find parent space path: %w", err)
		}

		oldPathValue := primaryPath.Value
		primaryPath.Value = paths.Concatinate(parentPath.Value, repo.UID)
		repo.Path = primaryPath.Value

		err = c.pathStore.Update(ctx, primaryPath)
		if err != nil {
			return fmt.Errorf("failed to update primary path: %w", err)
		}

		if in.KeepAsAlias {
			now := time.Now().UnixMilli()
			err = c.pathStore.Create(ctx, &types.Path{
				Version:    0,
				Value:      oldPathValue,
				IsPrimary:  false,
				TargetType: enum.PathTargetTypeRepo,
				TargetID:   repo.ID,
				CreatedBy:  session.Principal.ID,
				Created:    now,
				Updated:    now,
			})
			if err != nil {
				return fmt.Errorf("failed to create alias path: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	repo.GitURL = c.urlProvider.GenerateRepoCloneURL(repo.Path)

	return repo, nil
}

func (c *Controller) sanitizeMoveInput(in *MoveInput) error {
	if in.UID != nil {
		if err := c.uidCheck(*in.UID, false); err != nil {
			return err
		}
	}

	if in.ParentRef != nil {
		parentRefAsID, err := strconv.ParseInt(*in.ParentRef, 10, 64)
		if (err == nil && parentRefAsID <= 0) || (len(strings.TrimSpace(*in.ParentRef)) == 0) {
			return errRepositoryRequiresParent

		}
	}

	return nil
}
