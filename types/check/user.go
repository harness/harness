// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"errors"

	"github.com/bradrydzewski/my-app/types"
)

var (
	// ErrEmailLen  is returned when the email address
	// exceeds the maximum number of characters.
	ErrEmailLen = errors.New("Email address cannot exceed 250 characters")
)

// User returns true if the User if valid.
func User(user *types.User) (bool, error) {
	if len(user.Email) > 250 {
		return false, ErrEmailLen
	}
	return true, nil
}
