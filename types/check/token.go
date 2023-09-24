// Copyright 2023 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	ErrTokenLifeTimeRequired = &ValidationError{
		"The life time of a token is required.",
	}
)

// TokenLifetime returns true if the lifetime is valid for a token.
func TokenLifetime(lifetime *time.Duration, optional bool) error {
	if lifetime == nil && !optional {
		return ErrTokenLifeTimeRequired
	}

	if lifetime == nil {
		return nil
	}

	if *lifetime < minTokenLifeTime || *lifetime > maxTokenLifeTime {
		return ErrTokenLifeTimeOutOfBounds
	}

	return nil
}
