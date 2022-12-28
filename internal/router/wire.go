// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package router

import (
	"github.com/harness/gitness/gitrpc"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	"github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

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
	config *types.Config,
	repoStore store.RepoStore,
	authenticator authn.Authenticator,
	authorizer authz.Authorizer,
	client gitrpc.Interface,
) GitHandler {
	return NewGitHandler(config, repoStore, authenticator, authorizer, client)
}

func ProvideAPIHandler(
	config *types.Config,
	authenticator authn.Authenticator,
	repoCtrl *repo.Controller,
	spaceCtrl *space.Controller,
	pullreqCtrl *pullreq.Controller,
	webhookCtrl *webhook.Controller,
	saCtrl *serviceaccount.Controller,
	userCtrl *user.Controller) APIHandler {
	return NewAPIHandler(config, authenticator, repoCtrl, spaceCtrl, pullreqCtrl, webhookCtrl, saCtrl, userCtrl)
}

func ProvideWebHandler(config *types.Config) WebHandler {
	return NewWebHandler(config)
}
