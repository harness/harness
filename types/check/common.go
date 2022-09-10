// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
	"regexp"
)

const (
	minNameLength = 1
	maxNameLength = 64
	nameRegex     = "^[a-z][a-z0-9\\-\\_]*$"

	minDisplayNameLength = 1
	maxDisplayNameLength = 256
	displayNameRegex     = "^[a-zA-Z][a-zA-Z0-9\\-\\_ ]*$"
)

var (
	ErrNameLength = &CheckError{fmt.Sprintf("Name has to be between %d and %d in length.", minNameLength, maxNameLength)}
	ErrNameRegex  = &CheckError{fmt.Sprintf("Name has start with a letter and only contain the following [a-z0-9-_].")}

	ErrDisplayNameLength = &CheckError{fmt.Sprintf("Display name has to be between %d and %d in length.", minDisplayNameLength, maxDisplayNameLength)}
	ErrDisplayNameRegex  = &CheckError{fmt.Sprintf("Display name has start with a letter and only contain the following [a-zA-Z0-9-_ ].")}
)

// Name checks the provided name and returns an error in it isn't valid.
func Name(name string) error {
	l := len(name)
	if l < minNameLength || l > maxNameLength {
		return ErrNameLength
	}

	if ok, _ := regexp.Match(nameRegex, []byte(name)); !ok {
		return ErrNameRegex
	}

	return nil
}

// DisplayName checks the provided name and returns an error in it isn't valid.
func DisplayName(name string) error {
	l := len(name)
	if l < minDisplayNameLength || l > maxDisplayNameLength {
		return ErrDisplayNameLength
	}

	if ok, _ := regexp.Match(displayNameRegex, []byte(name)); !ok {
		return ErrDisplayNameRegex
	}

	return nil
}
