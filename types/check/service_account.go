// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
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

// ServiceAccountParent verifies the remaining fields of a service account
// that aren't inhereted from principal.
func ServiceAccountParent(parentType enum.ParentResourceType, parentID int64) error {
	if parentType != enum.ParentResourceTypeRepo && parentType != enum.ParentResourceTypeSpace {
		return ErrServiceAccountParentTypeIsInvalid
	}

	// validate service account belongs to sth
	if parentID <= 0 {
		return ErrServiceAccountParentIDInvalid
	}

	return nil
}
