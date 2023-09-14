// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"context"
	"fmt"

	"github.com/harness/gitness/internal/auth"
	"github.com/harness/gitness/internal/paths"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

var _ Authorizer = (*MembershipAuthorizer)(nil)

type MembershipAuthorizer struct {
	permissionCache PermissionCache
	spaceStore      store.SpaceStore
}

func NewMembershipAuthorizer(
	permissionCache PermissionCache,
	spaceStore store.SpaceStore,
) *MembershipAuthorizer {
	return &MembershipAuthorizer{
		permissionCache: permissionCache,
		spaceStore:      spaceStore,
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

	var spacePath string

	switch resource.Type {
	case enum.ResourceTypeSpace:
		spacePath = paths.Concatinate(scope.SpacePath, resource.Name)

	case enum.ResourceTypeRepo:
		spacePath = scope.SpacePath

	case enum.ResourceTypeServiceAccount:
		spacePath = scope.SpacePath

	case enum.ResourceTypePipeline:
		spacePath = scope.SpacePath

	case enum.ResourceTypeSecret:
		spacePath = scope.SpacePath

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

	// ephemeral membership overrides any other space memberships of the principal
	if membershipMetadata, ok := session.Metadata.(*auth.MembershipMetadata); ok {
		return a.checkWithMembershipMetadata(ctx, membershipMetadata, spacePath, permission)
	}

	// ensure we aren't bypassing unknown metadata with impact on authorization
	if session.Metadata.ImpactsAuthorization() {
		return false, fmt.Errorf("session contains unknown metadata that impacts authorization: %T", session.Metadata)
	}

	return a.permissionCache.Get(ctx, PermissionCacheKey{
		PrincipalID: session.Principal.ID,
		SpaceRef:    spacePath,
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

// checkWithMembershipMetadata checks access using the ephemeral membership provided in the metadata.
func (a *MembershipAuthorizer) checkWithMembershipMetadata(
	ctx context.Context,
	membershipMetadata *auth.MembershipMetadata,
	requestedSpacePath string,
	requestedPermission enum.Permission,
) (bool, error) {
	space, err := a.spaceStore.Find(ctx, membershipMetadata.SpaceID)
	if err != nil {
		return false, fmt.Errorf("failed to find space: %w", err)
	}

	if !paths.IsAncesterOf(space.Path, requestedSpacePath) {
		return false, fmt.Errorf(
			"requested permission scope '%s' is outside of ephemeral membership scope '%s'",
			requestedSpacePath,
			space.Path,
		)
	}

	if !roleHasPermission(membershipMetadata.Role, requestedPermission) {
		return false, fmt.Errorf(
			"requested permission '%s' is outside of ephemeral membership role '%s'",
			requestedPermission,
			membershipMetadata.Role,
		)
	}

	// access is granted by ephemeral membership
	return true, nil
}
