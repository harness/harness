// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// PrincipalType defines the supported types of principals.
type PrincipalType string

func (PrincipalType) Enum() []interface{} { return toInterfaceSlice(principalTypes) }

const (
	// PrincipalTypeUser represents a user.
	PrincipalTypeUser PrincipalType = "user"
	// PrincipalTypeServiceAccount represents a service account.
	PrincipalTypeServiceAccount PrincipalType = "serviceaccount"
	// PrincipalTypeService represents a service.
	PrincipalTypeService PrincipalType = "service"
)

var principalTypes = sortEnum([]PrincipalType{
	PrincipalTypeUser,
	PrincipalTypeServiceAccount,
	PrincipalTypeService,
})
