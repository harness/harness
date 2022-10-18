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
)

// ServiceAccount returns true if the ServiceAccount if valid.
func ServiceAccount(sa *types.ServiceAccount) error {
	// validate UID
	if err := UID(sa.UID); err != nil {
		return err
	}

	// validate name
	if err := Name(sa.Name); err != nil {
		return err
	}

	// validate parentType
	if sa.ParentType != enum.ParentResourceTypeRepo && sa.ParentType != enum.ParentResourceTypeSpace {
		return ErrServiceAccountParentTypeIsInvalid
	}

	return nil
}
