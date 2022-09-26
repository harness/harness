// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package harness

import "github.com/harness/gitness/types/enum"

// Metadata is used for all harness embedded auths (apart from ssh).
type Metadata struct {
	ExecutingPrincipalType enum.PrincipalType
	ExecutingPrincipalID   int64
}

// RequiresEnforcement returns true if the metadata contains authz related info.
func (m *Metadata) RequiresEnforcement() bool {
	return true
}
