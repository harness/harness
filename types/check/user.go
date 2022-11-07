// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types"
)

// User returns true if the User is valid.
type User func(*types.User) error

// UserDefault is the default User validation.
func UserDefault(user *types.User) error {
	// validate UID
	if err := UID(user.UID); err != nil {
		return err
	}

	// Validate Email
	if err := Email(user.Email); err != nil {
		return err
	}

	// validate DisplayName
	if err := DisplayName(user.DisplayName); err != nil {
		return err
	}

	return nil
}
