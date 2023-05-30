// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject && !harness
// +build wireinject,!harness

package server

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	gitrpccron "github.com/harness/gitness/gitrpc/server/cron"
	checkcontroller "github.com/harness/gitness/internal/api/controller/check"
	"github.com/harness/gitness/internal/api/controller/githook"
	"github.com/harness/gitness/internal/api/controller/principal"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/service"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	controllerwebhook "github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/bootstrap"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
	"github.com/harness/gitness/internal/services/codecomments"
	pullreqservice "github.com/harness/gitness/internal/services/pullreq"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/cache"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

func initSystem(ctx context.Context, config *types.Config) (*system, error) {
	wire.Build(
		newSystem,
		ProvideRedis,
		bootstrap.WireSet,
		database.WireSet,
		cache.WireSet,
		router.WireSet,
		pullreqservice.WireSet,
		services.WireSet,
		server.WireSet,
		url.WireSet,
		space.WireSet,
		repo.WireSet,
		pullreq.WireSet,
		controllerwebhook.WireSet,
		serviceaccount.WireSet,
		user.WireSet,
		service.WireSet,
		principal.WireSet,
		authn.WireSet,
		authz.WireSet,
		gitevents.WireSet,
		pullreqevents.WireSet,
		ProvideGitRPCServerConfig,
		gitrpcserver.WireSet,
		ProvideGitRPCClientConfig,
		gitrpc.WireSet,
		store.WireSet,
		check.WireSet,
		ProvideEventsConfig,
		events.WireSet,
		ProvideWebhookConfig,
		webhook.WireSet,
		githook.WireSet,
		lock.WireSet,
		pubsub.WireSet,
		codecomments.WireSet,
		gitrpccron.WireSet,
		checkcontroller.WireSet,
	)
	return &system{}, nil
}
