// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	ErrNotAuthenticated          = errors.New("not authenticated")
	ErrNotAuthorized             = errors.New("not authorized")
	ErrParentResourceTypeUnknown = errors.New("Unknown parent resource type")
)

// Check checks if a resource specific permission is granted for the current auth session in the scope.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func Check(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	scope *types.Scope, resource *types.Resource, permission enum.Permission,
) error {
	if session == nil {
		return ErrNotAuthenticated
	}

	authorized, err := authorizer.Check(
		ctx,
		session,
		scope,
		resource,
		permission)
	if err != nil {
		return err
	}

	if !authorized {
		return ErrNotAuthorized
	}

	return nil
}

// CheckChild checks if a resource specific permission is granted for the current auth session
// in the scope of a parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckChild(ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	spaceStore store.SpaceStore, repoStore store.RepoStore, parentType enum.ParentResourceType, parentID int64,
	resourceType enum.ResourceType, resourceName string, permission enum.Permission) error {
	scope, err := getScopeForParent(ctx, spaceStore, repoStore, parentType, parentID)
	if err != nil {
		return err
	}

	resource := &types.Resource{
		Type: resourceType,
		Name: resourceName,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}

// getScopeForParent Returns the scope for a given resource parent (space or repo).
func getScopeForParent(ctx context.Context, spaceStore store.SpaceStore, repoStore store.RepoStore,
	parentType enum.ParentResourceType, parentID int64) (*types.Scope, error) {
	// TODO: Can this be done cleaner?
	switch parentType {
	case enum.ParentResourceTypeSpace:
		space, err := spaceStore.Find(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("parent space not found: %w", err)
		}

		return &types.Scope{SpacePath: space.Path}, nil

	case enum.ParentResourceTypeRepo:
		repo, err := repoStore.Find(ctx, parentID)
		if err != nil {
			return nil, fmt.Errorf("parent repo not found: %w", err)
		}

		spacePath, repoName, err := paths.DisectLeaf(repo.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to disect path '%s'", repo.Path)
		}

		return &types.Scope{SpacePath: spacePath, Repo: repoName}, nil

	default:
		log.Ctx(ctx).Debug().Msgf("Unsupported parent type encountered: '%s'", parentType)

		return nil, ErrParentResourceTypeUnknown
	}
}
