// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/harness/gitness/types"
)

const (
	minSpaceNameLength = 1
	maxSpaceNameLength = 64
	spaceNameRegex     = "^[a-z][a-z0-9\\-\\_]*$"

	minSpaceDisplayNameLength = 1
	maxSpaceDisplayNameLength = 256
	spaceDisplayNameRegex     = "^[a-zA-Z][a-zA-Z0-9\\-\\_ ]*$"
)

var (
	ErrSpaceNameLength = errors.New(fmt.Sprintf("Space name has to be between %d and %d in length.", minSpaceNameLength, maxSpaceNameLength))
	ErrSpaceNameRegex  = errors.New("Space name has start with a letter and only contain the following [a-z0-9-_].")

	ErrSpaceDisplayNameLength = errors.New(fmt.Sprintf("Space display name has to be between %d and %d in length.", minSpaceDisplayNameLength, maxSpaceDisplayNameLength))
	ErrSpaceDisplayNameRegex  = errors.New("Space display name has start with a letter and only contain the following [a-zA-Z0-9-_ ].")

	illegalRootSpaceNames      = []string{"api"}
	ErrRootSpaceNameNotAllowed = errors.New(fmt.Sprintf("The following names are not allowed for a root space: %v", illegalRootSpaceNames))

	ErrInvalidParentSpaceId = errors.New("Parent space ID has to be either zero for a root space or greater than zero for a child space.")
)

// User returns true if the User if valid.
func Space(space *types.Space) (bool, error) {
	l := len(space.Name)
	if l < minSpaceNameLength || l > maxSpaceNameLength {
		return false, ErrSpaceNameLength
	}

	if ok, _ := regexp.Match(spaceNameRegex, []byte(space.Name)); !ok {
		return false, ErrSpaceNameRegex
	}

	l = len(space.DisplayName)
	if l < minSpaceDisplayNameLength || l > maxSpaceDisplayNameLength {
		return false, ErrSpaceDisplayNameLength
	}

	if ok, _ := regexp.Match(spaceDisplayNameRegex, []byte(space.DisplayName)); !ok {
		return false, ErrSpaceDisplayNameRegex
	}

	if space.ParentId < 0 {
		return false, ErrInvalidParentSpaceId
	}

	// root space specific validations
	if space.ParentId == 0 {
		for _, p := range illegalRootSpaceNames {
			if strings.HasPrefix(space.Name, p) {
				return false, ErrRootSpaceNameNotAllowed
			}
		}
	}

	return true, nil
}
