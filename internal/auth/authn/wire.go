// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package authn

import (
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideAuthenticator,
)

func ProvideAuthenticator(config *types.Config, principalStore store.PrincipalStore, tokenStore store.TokenStore) Authenticator {
	return NewTokenAuthenticator(principalStore, tokenStore, config.Token.CookieName)
}
