// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"github.com/google/wire"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/guard"
	"github.com/harness/gitness/internal/router/translator"
	"github.com/harness/gitness/internal/store"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideRouter,
	ProvideGitHandler,
	ProvideAPIHandler,
	ProvideWebHandler,
)

func ProvideRouter(
	translator translator.RequestTranslator,
	api APIHandler,
	git GitHandler,
	web WebHandler,
) *Router {
	return NewRouter(translator, api, git, web)
}

func ProvideGitHandler(
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	guard *guard.Guard) GitHandler {
	return NewGitHandler(repoStore, authenticator, guard)
}

func ProvideAPIHandler(
	systemStore store.SystemStore,
	userStore store.UserStore,
	spaceStore store.SpaceStore,
	repoStore store.RepoStore,
	tokenStore store.TokenStore,
	saStore store.ServiceAccountStore,
	authenticator authn.Authenticator,
	guard *guard.Guard) APIHandler {
	return NewAPIHandler(systemStore, userStore, spaceStore, repoStore, tokenStore,
		saStore, authenticator, guard)
}

func ProvideWebHandler(systemStore store.SystemStore) WebHandler {
	return NewWebHandler(systemStore)
}
