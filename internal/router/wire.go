// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"net/http"

	"github.com/google/wire"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/router/translator"
	"github.com/harness/gitness/internal/store"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(ProvideHTTPHandler)

func ProvideHTTPHandler(
	translator translator.RequestTranslator,
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	tokenStore store.TokenStore,
	saStore store.ServiceAccountStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
) (http.Handler, error) {
	return NewRouter(translator, systemStore, userStore, spaceStore,
		repoStore, tokenStore, saStore, authenticator, authorizer)
}
