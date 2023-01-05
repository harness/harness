// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package user

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

func ProvideController(userCheck check.User, authorizer authz.Authorizer, principalStore store.PrincipalStore,
	tokenStore store.TokenStore) *Controller {
	return NewController(userCheck, authorizer, principalStore, tokenStore)
}
