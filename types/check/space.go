// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"strings"

	"github.com/harness/gitness/types"
)

var (
	illegalRootSpaceNames = []string{"api"}

	ErrRootSpaceNameNotAllowed = fmt.Errorf("The following names are not allowed for a root space: %v", illegalRootSpaceNames)
	ErrInvalidParentSpaceId    = fmt.Errorf("Parent space ID has to be either zero for a root space or greater than zero for a child space.")
)

// Repo checks the provided space and returns an error in it isn't valid.
func Space(space *types.Space) error {
	// validate name
	if err := Name(space.Name); err != nil {
		return err
	}

	// validate display name
	if err := DisplayName(space.DisplayName); err != nil {
		return err
	}

	if space.ParentId < 0 {
		return ErrInvalidParentSpaceId
	}

	// root space specific validations
	if space.ParentId == 0 {
		for _, p := range illegalRootSpaceNames {
			if strings.HasPrefix(space.Name, p) {
				return ErrRootSpaceNameNotAllowed
			}
		}
	}

	return nil
}
