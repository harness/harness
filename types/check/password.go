// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"fmt"
)

const (
	minPasswordLength = 1
	maxPasswordLength = 128
)

var (
	// ErrPasswordLength is returned when the password
	// is outside of the allowed length.
	ErrPasswordLength = &ValidationError{
		fmt.Sprintf("Password has to be within %d and %d characters", minPasswordLength, maxPasswordLength),
	}
)

// Password returns true if the Password is valid.
// TODO: add proper password checks.
func Password(pw string) error {
	// validate length
	l := len(pw)
	if l < minPasswordLength || l > maxPasswordLength {
		return ErrPasswordLength
	}

	return nil
}
