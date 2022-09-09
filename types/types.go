// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

// Package types defines common data structures.
package types

import (
	"time"

	"github.com/harness/gitness/types/enum"
)

type (
	// Params stores query parameters.
	Params struct {
		Page  int        `json:"page"`
		Size  int        `json:"size"`
		Sort  string     `json:"sort"`
		Order enum.Order `json:"direction"`
	}

	// Token stores token  details.
	Token struct {
		Value   string    `json:"access_token"`
		Address string    `json:"uri,omitempty"`
		Expires time.Time `json:"expires_at,omitempty"`
	}
)
