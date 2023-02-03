// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package service

import (
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	NewController,
)

func ProvideController(principalUIDCheck check.PrincipalUID, authorizer authz.Authorizer,
	principalStore store.PrincipalStore) *Controller {
	return NewController(principalUIDCheck, authorizer, principalStore)
}
