// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var (
	ErrServiceAccountParentTypeIsInvalid = &ValidationError{
		"Provided parent type is invalid.",
	}
	ErrServiceAccountParentIDInvalid = &ValidationError{
		"ParentID required - Global service accounts are not supported.",
	}
)

// ServiceAccount returns true if the ServiceAccount is valid.
type ServiceAccount func(*types.ServiceAccount) error

// ServiceAccountDefault is the default ServiceAccount validation.
func ServiceAccountDefault(sa *types.ServiceAccount) error {
	// validate UID
	if err := UID(sa.UID); err != nil {
		return err
	}

	// Validate Email
	if err := Email(sa.Email); err != nil {
		return err
	}

	// validate DisplayName
	if err := DisplayName(sa.DisplayName); err != nil {
		return err
	}

	// validate remaining
	return ServiceAccountNoPrincipal(sa)
}

// ServiceAccountNoPrincipal verifies the remaining fields of a service account
// that aren't inhereted from principal.
func ServiceAccountNoPrincipal(sa *types.ServiceAccount) error {
	// validate parentType
	if sa.ParentType != enum.ParentResourceTypeRepo && sa.ParentType != enum.ParentResourceTypeSpace {
		return ErrServiceAccountParentTypeIsInvalid
	}

	// validate service account belongs to a space
	if sa.ParentID <= 0 {
		return ErrServiceAccountParentIDInvalid
	}

	return nil
}
