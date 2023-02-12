// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package pullreq

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/types"
)

func rpcIdentityFromPrincipal(p types.Principal) *gitrpc.Identity {
	return &gitrpc.Identity{
		Name:  p.DisplayName,
		Email: p.Email,
	}
}
