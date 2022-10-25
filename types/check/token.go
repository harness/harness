// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package check

import (
	"time"
)

const (
	minTokenLifeTime = 24 * time.Hour       // 1 day
	maxTokenLifeTime = 365 * 24 * time.Hour // 1 year
)

var (
	ErrTokenLifeTimeOutOfBounds = &ValidationError{
		"The life time of a token has to be between 1 day and 365 days.",
	}
)

// TokenLifetime returns true if the lifetime is valid for a token.
func TokenLifetime(lifetime time.Duration) error {
	if lifetime < minTokenLifeTime || lifetime > maxTokenLifeTime {
		return ErrTokenLifeTimeOutOfBounds
	}

	return nil
}
