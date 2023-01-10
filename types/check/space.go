// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/types"
)

var (
	illegalRootSpaceNames = []string{"api", "git"}

	ErrRootSpaceNameNotAllowed = &ValidationError{
		fmt.Sprintf("The following names are not allowed for a root space: %v", illegalRootSpaceNames),
	}
	ErrInvalidParentID = &ValidationError{
		"Parent ID has to be either zero for a root space or greater than zero for a child space.",
	}
)

// Space returns true if the Space is valid.
type Space func(*types.Space) error

// SpaceDefault is the default space validation.
func SpaceDefault(space *types.Space) error {
	// validate UID
	if err := UID(space.UID); err != nil {
		return err
	}

	// validate the rest
	return SpaceNoUID(space)
}

// SpaceNoUID validates the space and ignores the UID field.
func SpaceNoUID(space *types.Space) error {
	if space.ParentID < 0 {
		return ErrInvalidParentID
	}

	// root space specific validations
	if space.ParentID == 0 {
		for _, p := range illegalRootSpaceNames {
			if strings.HasPrefix(space.UID, p) {
				return ErrRootSpaceNameNotAllowed
			}
		}
	}

	return nil
}
