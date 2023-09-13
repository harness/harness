// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject
// +build wireinject

package main

import (
	"context"

	cliserver "github.com/harness/gitness/cli/server"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/gitrpc"
	gitrpcserver "github.com/harness/gitness/gitrpc/server"
	gitrpccron "github.com/harness/gitness/gitrpc/server/cron"
	checkcontroller "github.com/harness/gitness/internal/api/controller/check"
	"github.com/harness/gitness/internal/api/controller/connector"
	"github.com/harness/gitness/internal/api/controller/execution"
	"github.com/harness/gitness/internal/api/controller/githook"
	controllerlogs "github.com/harness/gitness/internal/api/controller/logs"
	"github.com/harness/gitness/internal/api/controller/pipeline"
	"github.com/harness/gitness/internal/api/controller/plugin"
	"github.com/harness/gitness/internal/api/controller/principal"
	"github.com/harness/gitness/internal/api/controller/pullreq"
	"github.com/harness/gitness/internal/api/controller/repo"
	"github.com/harness/gitness/internal/api/controller/secret"
	"github.com/harness/gitness/internal/api/controller/service"
	"github.com/harness/gitness/internal/api/controller/serviceaccount"
	"github.com/harness/gitness/internal/api/controller/space"
	"github.com/harness/gitness/internal/api/controller/system"
	"github.com/harness/gitness/internal/api/controller/template"
	controllertrigger "github.com/harness/gitness/internal/api/controller/trigger"
	"github.com/harness/gitness/internal/api/controller/user"
	controllerwebhook "github.com/harness/gitness/internal/api/controller/webhook"
	"github.com/harness/gitness/internal/auth/authn"
	"github.com/harness/gitness/internal/auth/authz"
	"github.com/harness/gitness/internal/bootstrap"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/pipeline/canceler"
	"github.com/harness/gitness/internal/pipeline/commit"
	"github.com/harness/gitness/internal/pipeline/file"
	"github.com/harness/gitness/internal/pipeline/manager"
	"github.com/harness/gitness/internal/pipeline/runner"
	"github.com/harness/gitness/internal/pipeline/scheduler"
	"github.com/harness/gitness/internal/pipeline/triggerer"
	"github.com/harness/gitness/internal/router"
	"github.com/harness/gitness/internal/server"
	"github.com/harness/gitness/internal/services"
	"github.com/harness/gitness/internal/services/codecomments"
	"github.com/harness/gitness/internal/services/importer"
	"github.com/harness/gitness/internal/services/job"
	pullreqservice "github.com/harness/gitness/internal/services/pullreq"
	"github.com/harness/gitness/internal/services/trigger"
	"github.com/harness/gitness/internal/services/webhook"
	"github.com/harness/gitness/internal/sse"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/internal/store/cache"
	"github.com/harness/gitness/internal/store/database"
	"github.com/harness/gitness/internal/store/logs"
	"github.com/harness/gitness/internal/url"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"

	"github.com/google/wire"
)

func initSystem(ctx context.Context, config *types.Config) (*cliserver.System, error) {
	wire.Build(
		cliserver.NewSystem,
		cliserver.ProvideRedis,
		bootstrap.WireSet,
		cliserver.ProvideDatabaseConfig,
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
		system.WireSet,
		authn.WireSet,
		authz.WireSet,
		gitevents.WireSet,
		pullreqevents.WireSet,
		cliserver.ProvideGitRPCServerConfig,
		gitrpcserver.WireSet,
		cliserver.ProvideGitRPCClientConfig,
		gitrpc.WireSet,
		store.WireSet,
		check.WireSet,
		encrypt.WireSet,
		cliserver.ProvideEventsConfig,
		events.WireSet,
		cliserver.ProvideWebhookConfig,
		webhook.WireSet,
		cliserver.ProvideTriggerConfig,
		trigger.WireSet,
		githook.WireSet,
		cliserver.ProvideLockConfig,
		lock.WireSet,
		pubsub.WireSet,
		codecomments.WireSet,
		job.WireSet,
		gitrpccron.WireSet,
		checkcontroller.WireSet,
		execution.WireSet,
		pipeline.WireSet,
		logs.WireSet,
		livelog.WireSet,
		controllerlogs.WireSet,
		secret.WireSet,
		connector.WireSet,
		template.WireSet,
		manager.WireSet,
		triggerer.WireSet,
		file.WireSet,
		runner.WireSet,
		sse.WireSet,
		scheduler.WireSet,
		commit.WireSet,
		controllertrigger.WireSet,
		plugin.WireSet,
		importer.WireSet,
		canceler.WireSet,
	)
	return &cliserver.System{}, nil
}
