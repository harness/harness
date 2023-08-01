// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"context"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ Authorizer = (*MembershipAuthorizer)(nil)

type MembershipAuthorizer struct {
	permissionCache PermissionCache
}

func NewMembershipAuthorizer(
	permissionCache PermissionCache,
) *MembershipAuthorizer {
	return &MembershipAuthorizer{
		permissionCache: permissionCache,
	}
}

func (a *MembershipAuthorizer) Check(
	ctx context.Context,
	session *auth.Session,
	scope *types.Scope,
	resource *types.Resource,
	permission enum.Permission,
) (bool, error) {
	log.Ctx(ctx).Debug().Msgf(
		"[MembershipAuthorizer] %s with id '%d' requests %s for %s '%s' in scope %#v with metadata %#v",
		session.Principal.Type,
		session.Principal.ID,
		permission,
		resource.Type,
		resource.Name,
		scope,
		session.Metadata,
	)

	if session.Principal.Admin {
		return true, nil // system admin can call any API
	}

	var spaceRef string

	switch resource.Type {
	case enum.ResourceTypeSpace:
		spaceRef = paths.Concatinate(scope.SpacePath, resource.Name)

	case enum.ResourceTypeRepo:
		spaceRef = scope.SpacePath

	case enum.ResourceTypeServiceAccount:
		spaceRef = scope.SpacePath

	case enum.ResourceTypeUser:
		// a user is allowed to view / edit themselves
		if resource.Name == session.Principal.UID &&
			(permission == enum.PermissionUserView || permission == enum.PermissionUserEdit) {
			return true, nil
		}

		// everything else is reserved for admins only (like operations on users other than yourself, or setting admin)
		return false, nil

	// Service operations aren't exposed to users
	case enum.ResourceTypeService:
		return false, nil

	default:
		return false, nil
	}

	return a.permissionCache.Get(ctx, PermissionCacheKey{
		PrincipalID: session.Principal.ID,
		SpaceRef:    spaceRef,
		Permission:  permission,
	})
}
func (a *MembershipAuthorizer) CheckAll(ctx context.Context, session *auth.Session,
	permissionChecks ...types.PermissionCheck) (bool, error) {
	for _, p := range permissionChecks {
		if _, err := a.Check(ctx, session, &p.Scope, &p.Resource, p.Permission); err != nil {
			return false, err
		}
	}

	return true, nil
}
