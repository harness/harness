// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"

	"github.com/harness/gitness/types"
)

const (
	minEmailLength = 1
	maxEmailLength = 250
)

var (
	// ErrEmailLen  is returned when the email address
	// exceeds the maximum number of characters.
	ErrEmailLen = &ValidationError{
		fmt.Sprintf("Email address has to be within %d and %d characters", minEmailLength, maxEmailLength),
	}
)

// User returns true if the User if valid.
func User(user *types.User) error {
	// validate name
	if err := Name(user.Name); err != nil {
		return err
	}

	// validate email
	l := len(user.Email)
	if l < minEmailLength || l > maxEmailLength {
		return ErrEmailLen
	}

	return nil
}
