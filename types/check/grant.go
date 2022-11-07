// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"github.com/harness/gitness/types/enum"
)

var (
	ErrTokenGrantEmpty = &ValidationError{
		"The token requires at least one grant.",
	}
)

// AccessGrant returns true if the access grant is valid.
func AccessGrant(grant enum.AccessGrant, allowNone bool) error {
	if !allowNone && grant == enum.AccessGrantNone {
		return ErrTokenGrantEmpty
	}

	// TODO: Ensure grant contains valid values?

	return nil
}
