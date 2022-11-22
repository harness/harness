// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"

	"github.com/google/wire"
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideRouter,
	ProvideGitHandler,
	ProvideAPIHandler,
	ProvideWebHandler,
)

func ProvideRouter(
	api APIHandler,
	git GitHandler,
	web WebHandler,
) *Router {
	return NewRouter(api, git, web)
}

func ProvideGitHandler(
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
	client gitrpc.Interface,
) GitHandler {
	return NewGitHandler(repoStore, authenticator, authorizer, client)
}

func ProvideAPIHandler(
	systemStore store.SystemStore,
	authenticator authn.Authenticator,
	repoCtrl *repo.Controller,
	spaceCtrl *space.Controller,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller) APIHandler {
	return NewAPIHandler(systemStore, authenticator, repoCtrl, spaceCtrl, saCtrl, userCtrl)
}

func ProvideWebHandler(systemStore store.SystemStore) WebHandler {
	return NewWebHandler(systemStore)
}
