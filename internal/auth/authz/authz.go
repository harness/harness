// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

/*
 * An abstraction of an entity responsible for authorizing access to resources.
 */
type Authorizer interface {
	/*
	 * Checks whether the provided principal has the permission to execute the action on the resource within the scope.
	 * Returns
	 *		(true, nil)   - the principal has permission to perform the action
	 *		(false, nil)  - the principal does not have permission to perform the action
	 *		(false, err)  - an error occured while performing the permission check and the action should be denied
	 */
	Check(principalType enum.PrincipalType, principalId string, scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error)

	/*
	 * Checks whether the provided principal the required permission to execute ALL the requested actions on the resource within the scope.
	 * Returns
	 *		(true, nil)   - the principal has permission to perform all the requested actions
	 *		(false, nil)  - the principal does not have permission to perform all the actions (at least one is not allowed)
	 *		(false, err)  - an error occured while performing the permission check and all actions should be denied
	 */
	CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...*types.PermissionCheck) (bool, error)
}
