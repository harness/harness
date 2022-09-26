// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package auth

import (
	"github.com/harness/gitness/types"
)

// Session contains information of the authenticated principal and auth related metadata.
type Session struct {
	// Principal is the authenticated principal.
	Principal types.Principal

	// Metadata contains auth related information (access grants, tokenId, sshKeyId, ...)
	Metadata Metadata
}
