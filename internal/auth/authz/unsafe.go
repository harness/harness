// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"context"

	"github.com/rs/zerolog/log"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var _ Authorizer = (*UnsafeAuthorizer)(nil)

/*
 * An unsafe authorizer that gives permits any action and simply logs the permission request.
 */
type UnsafeAuthorizer struct{}

func NewUnsafeAuthorizer() Authorizer {
	return &UnsafeAuthorizer{}
}

func (a *UnsafeAuthorizer) Check(ctx context.Context, principalType enum.PrincipalType, principalID string,
	scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error) {
	log.Debug().Msgf(
		"[Authz] %s '%s' requests %s for %s '%s' in scope %v\n",
		principalType,
		principalID,
		permission,
		resource.Type,
		resource.Name,
		scope,
	)

	return true, nil
}
func (a *UnsafeAuthorizer) CheckAll(ctx context.Context, principalType enum.PrincipalType, principalID string,
	permissionChecks ...types.PermissionCheck) (bool, error) {
	for _, p := range permissionChecks {
		if _, err := a.Check(ctx, principalType, principalID, &p.Scope, &p.Resource, p.Permission); err != nil {
			return false, err
		}
	}

	return true, nil
}
