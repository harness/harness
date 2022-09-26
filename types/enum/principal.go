// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package enum

// Represents the type of the entity requesting permission.
type PrincipalType string

const (
	// Represents a user.
	PrincipalTypeUser PrincipalType = "user"
	// Represents a service account.
	PrincipalTypeServiceAccount PrincipalType = "serviceaccount"
	// Represents a service.
	PrincipalTypeService PrincipalType = "service"
)
