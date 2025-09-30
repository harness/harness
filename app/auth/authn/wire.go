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

package authn

import (
	"crypto/rand"
	"fmt"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
	"github.com/rs/zerolog/log"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideAuthenticator,
)

func ProvideAuthenticator(
	config *types.Config,
	principalStore store.PrincipalStore,
	tokenStore store.TokenStore,
) Authenticator {
	if config.Auth.AnonymousUserSecret == "" {
		var secretBytes [32]byte
		if _, err := rand.Read(secretBytes[:]); err != nil {
			panic(fmt.Sprintf("could not generate random bytes for anonymous user secret: %v", err))
		}
		config.Auth.AnonymousUserSecret = string(secretBytes[:])
		log.Warn().Msg("No anonymous secret provided - generated random secret.")
	}
	return NewTokenAuthenticator(principalStore, tokenStore, config.Token.CookieName, config.Auth.AnonymousUserSecret)
}
