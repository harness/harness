// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"fmt"

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

func (a *UnsafeAuthorizer) Check(principalType enum.PrincipalType, principalId string, scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error) {
	fmt.Printf(
		"[Authz] %s '%s' requests %s for %s '%s' in scope %v\n",
		principalType,
		principalId,
		permission,
		resource.Type,
		resource.Name,
		scope,
	)

	return true, nil
}
func (a *UnsafeAuthorizer) CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...types.PermissionCheck) (bool, error) {
	for _, p := range permissionChecks {
		if _, err := a.Check(principalType, principalId, &p.Scope, &p.Resource, p.Permission); err != nil {
			return false, err
		}
	}

	return true, nil
}
