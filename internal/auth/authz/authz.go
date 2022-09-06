// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authz

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

type Authorizer interface {
	Check(principalType enum.PrincipalType, principalId string, scope *types.Scope, resource *types.Resource, permission enum.Permission) (bool, error)
	CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...*types.PermissionCheck) (bool, error)
}
