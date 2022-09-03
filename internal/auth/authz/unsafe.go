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

func NewUnsafeAuthorizer() Authorizer {
	return &UnsafeAuthorizer{}
}

type UnsafeAuthorizer struct{}

func (a *UnsafeAuthorizer) Check(principalType enum.PrincipalType, principalId string, resource types.Resource, permission enum.Permission) error {
	fmt.Printf(
		"[Authz] %s '%s' requests %s for %s '%s'\n",
		principalType,
		principalId,
		permission,
		resource.Type,
		resource.Identifier,
	)

	return nil
}
func (a *UnsafeAuthorizer) CheckAll(principalType enum.PrincipalType, principalId string, permissionChecks ...*types.PermissionCheck) error {
	for _, p := range permissionChecks {
		a.Check(principalType, principalId, p.Resource, p.Permission)
	}

	return nil
}
