// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject && harness
// +build wireinject,harness

package server

import (
	"context"

	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	"github.com/harness/gitness/harness/auth/authn"
	"github.com/harness/gitness/harness/auth/authz"
	"github.com/harness/gitness/harness/bootstrap"
	"github.com/harness/gitness/harness/client"
	"github.com/harness/gitness/harness/router"
	"github.com/harness/gitness/harness/store"
	"github.com/harness/gitness/harness/types/check"
	"github.com/harness/gitness/internal/api/controller/githook"
	"github.com/harness/gitness/internal/api/controller/principal"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/service"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	controllerwebhook "github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/cron"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
	pullreqservice "github.com/harness/gitness/internal/services/pullreq"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/store/cache"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/lock"
	gitnesstypes "github.com/harness/gitness/types"

	"github.com/google/wire"
)

func initSystem(ctx context.Context, config *gitnesstypes.Config) (*system, error) {
	wire.Build(
		newSystem,
		ProvideHarnessConfig,
		ProvideRedis,
		bootstrap.WireSet,
		database.WireSet,
		pullreqservice.WireSet,
		services.WireSet,
		cache.WireSet,
		server.WireSet,
		cron.WireSet,
		url.WireSet,
		space.WireSet,
		repo.WireSet,
		pullreq.WireSet,
		controllerwebhook.WireSet,
		user.WireSet,
		service.WireSet,
		serviceaccount.WireSet,
		principal.WireSet,
		gitevents.WireSet,
		pullreqevents.WireSet,
		ProvideGitRPCServerConfig,
		gitrpcserver.WireSet,
		ProvideGitRPCClientConfig,
		gitrpc.WireSet,
		router.WireSet,
		authn.WireSet,
		authz.WireSet,
		client.WireSet,
		store.WireSet,
		check.WireSet,
		ProvideEventsConfig,
		events.WireSet,
		ProvideWebhookConfig,
		webhook.WireSet,
		githook.WireSet,
		lock.WireSet,
	)
	return &system{}, nil
}
