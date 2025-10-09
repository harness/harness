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

package auth

import (
	"context"
	"fmt"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/paths"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

var (
	ErrUnauthorized              = errors.New("unauthorized")
	ErrForbidden                 = errors.New("forbidden")
	ErrParentResourceTypeUnknown = errors.New("Unknown parent resource type")
	ErrPrincipalTypeUnknown      = errors.New("Unknown principal type")
)

// Check checks if a resource specific permission is granted for the current auth session in the scope.
// Returns nil if the permission is granted, otherwise returns an error.
func Check(
	ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	scope *types.Scope, resource *types.Resource, permission enum.Permission,
) error {
	authenticated, err := authorizer.Check(
		ctx,
		session,
		scope,
		resource,
		permission,
	)
	if err != nil {
		return err
	}

	return CheckSessionAuth(session, authenticated)
}

// CheckAll checks if multiple resources specific permission is granted for the current auth session in the scope.
// Returns nil if the permission is granted, otherwise returns an error.
func CheckAll(
	ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	permissionChecks ...types.PermissionCheck,
) error {
	hasPermission, err := authorizer.CheckAll(
		ctx,
		session,
		permissionChecks...,
	)
	if err != nil {
		return err
	}
	return CheckSessionAuth(session, hasPermission)
}

// CheckSessionAuth returns nil if the user is authenticated.
// Otherwise, ir returns err unauthorized on anonymous or err forbidden on non anonymous session.
func CheckSessionAuth(session *auth.Session, authenticated bool) error {
	if !authenticated {
		if auth.IsAnonymousSession(session) {
			return ErrUnauthorized
		}
		return ErrForbidden
	}

	return nil
}

// IsNoAccess returns true if the error is ErrUnauthorized or ErrForbidden.
func IsNoAccess(err error) bool {
	return errors.Is(err, ErrForbidden) || errors.Is(err, ErrUnauthorized)
}

// CheckChild checks if a resource specific permission is granted for the current auth session
// in the scope of a parent.
// Returns nil if the permission is granted, otherwise returns an error.
// NotAuthenticated, NotAuthorized, or any underlying error.
func CheckChild(
	ctx context.Context, authorizer authz.Authorizer, session *auth.Session,
	spaceStore store.SpaceStore, repoStore store.RepoStore, parentType enum.ParentResourceType, parentID int64,
	resourceType enum.ResourceType, resourceName string, permission enum.Permission,
) error {
	scope, err := getScopeForParent(ctx, spaceStore, repoStore, parentType, parentID)
	if err != nil {
		return err
	}

	resource := &types.Resource{
		Type:       resourceType,
		Identifier: resourceName,
	}

	return Check(ctx, authorizer, session, scope, resource, permission)
}

// getScopeForParent Returns the scope for a given resource parent (space or repo).
func getScopeForParent(
	ctx context.Context, spaceStore store.SpaceStore, repoStore store.RepoStore,
	parentType enum.ParentResourceType, parentID int64,
) (*types.Scope, error) {
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
			return nil, fmt.Errorf("failed to disect path '%s': %w", repo.Path, err)
		}

		return &types.Scope{SpacePath: spacePath, Repo: repoName}, nil

	default:
		log.Ctx(ctx).Debug().Msgf("Unsupported parent type encountered: '%s'", parentType)

		return nil, ErrParentResourceTypeUnknown
	}
}
