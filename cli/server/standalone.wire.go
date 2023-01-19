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
	"github.com/harness/gitness/internal/api/controller/githook"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/user"
	controllerwebhook "github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/bootstrap"
	"github.com/harness/gitness/internal/cron"
	eventsgit "github.com/harness/gitness/internal/events/git"
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
	"github.com/harness/gitness/internal/services/branchmonitor"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/cache"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

func initSystem(ctx context.Context, config *types.Config) (*system, error) {
	wire.Build(
		newSystem,
		PackageConfigsWireSet,
		ProvideRedis,
		bootstrap.WireSet,
		database.WireSet,
		cache.WireSet,
		router.WireSet,
		branchmonitor.WireSet,
		services.WireSet,
		server.WireSet,
		cron.WireSet,
		url.WireSet,
		space.WireSet,
		repo.WireSet,
		pullreq.WireSet,
		controllerwebhook.WireSet,
		serviceaccount.WireSet,
		user.WireSet,
		authn.WireSet,
		authz.WireSet,
		eventsgit.WireSet,
		gitrpcserver.WireSet,
		gitrpc.WireSet,
		store.WireSet,
		check.WireSet,
		events.WireSet,
		webhook.WireSet,
		githook.WireSet,
	)
	return &system{}, nil
}
