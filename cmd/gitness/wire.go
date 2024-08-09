// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

//go:build wireinject
// +build wireinject

package main

import (
	"context"

	checkcontroller "github.com/harness/gitness/app/api/controller/check"
	"github.com/harness/gitness/app/api/controller/connector"
	"github.com/harness/gitness/app/api/controller/execution"
	githookCtrl "github.com/harness/gitness/app/api/controller/githook"
	gitspaceCtrl "github.com/harness/gitness/app/api/controller/gitspace"
	infraproviderCtrl "github.com/harness/gitness/app/api/controller/infraprovider"
	controllerkeywordsearch "github.com/harness/gitness/app/api/controller/keywordsearch"
	"github.com/harness/gitness/app/api/controller/limiter"
	controllerlogs "github.com/harness/gitness/app/api/controller/logs"
	"github.com/harness/gitness/app/api/controller/migrate"
	"github.com/harness/gitness/app/api/controller/pipeline"
	"github.com/harness/gitness/app/api/controller/plugin"
	"github.com/harness/gitness/app/api/controller/principal"
	"github.com/harness/gitness/app/api/controller/pullreq"
	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/api/controller/reposettings"
	"github.com/harness/gitness/app/api/controller/secret"
	"github.com/harness/gitness/app/api/controller/service"
	"github.com/harness/gitness/app/api/controller/serviceaccount"
	"github.com/harness/gitness/app/api/controller/space"
	"github.com/harness/gitness/app/api/controller/system"
	"github.com/harness/gitness/app/api/controller/template"
	controllertrigger "github.com/harness/gitness/app/api/controller/trigger"
	"github.com/harness/gitness/app/api/controller/upload"
	"github.com/harness/gitness/app/api/controller/user"
	controllerwebhook "github.com/harness/gitness/app/api/controller/webhook"
	"github.com/harness/gitness/app/api/openapi"
	"github.com/harness/gitness/app/auth/authn"
	"github.com/harness/gitness/app/auth/authz"
	"github.com/harness/gitness/app/bootstrap"
	gitevents "github.com/harness/gitness/app/events/git"
	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	gitspaceinfraevents "github.com/harness/gitness/app/events/gitspaceinfra"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	repoevents "github.com/harness/gitness/app/events/repo"
	infrastructure "github.com/harness/gitness/app/gitspace/infrastructure"
	"github.com/harness/gitness/app/gitspace/logutil"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	containerorchestrator "github.com/harness/gitness/app/gitspace/orchestrator/container"
	"github.com/harness/gitness/app/gitspace/orchestrator/ide"
	"github.com/harness/gitness/app/gitspace/scm"
	"github.com/harness/gitness/app/pipeline/canceler"
	"github.com/harness/gitness/app/pipeline/commit"
	"github.com/harness/gitness/app/pipeline/converter"
	"github.com/harness/gitness/app/pipeline/file"
	"github.com/harness/gitness/app/pipeline/manager"
	"github.com/harness/gitness/app/pipeline/resolver"
	"github.com/harness/gitness/app/pipeline/runner"
	"github.com/harness/gitness/app/pipeline/scheduler"
	"github.com/harness/gitness/app/pipeline/triggerer"
	"github.com/harness/gitness/app/router"
	"github.com/harness/gitness/app/server"
	"github.com/harness/gitness/app/services"
	"github.com/harness/gitness/app/services/cleanup"
	"github.com/harness/gitness/app/services/codecomments"
	"github.com/harness/gitness/app/services/codeowners"
	"github.com/harness/gitness/app/services/exporter"
	"github.com/harness/gitness/app/services/gitspaceevent"
	"github.com/harness/gitness/app/services/gitspaceservice"
	"github.com/harness/gitness/app/services/importer"
	"github.com/harness/gitness/app/services/keywordsearch"
	svclabel "github.com/harness/gitness/app/services/label"
	locker "github.com/harness/gitness/app/services/locker"
	"github.com/harness/gitness/app/services/metric"
	migrateservice "github.com/harness/gitness/app/services/migrate"
	"github.com/harness/gitness/app/services/notification"
	"github.com/harness/gitness/app/services/notification/mailer"
	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/services/publicaccess"
	"github.com/harness/gitness/app/services/publickey"
	pullreqservice "github.com/harness/gitness/app/services/pullreq"
	reposervice "github.com/harness/gitness/app/services/repo"
	"github.com/harness/gitness/app/services/settings"
	"github.com/harness/gitness/app/services/trigger"
	"github.com/harness/gitness/app/services/usergroup"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/app/sse"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/logs"
	"github.com/harness/gitness/app/url"
	"github.com/harness/gitness/audit"
	"github.com/harness/gitness/blob"
	cliserver "github.com/harness/gitness/cli/operations/server"
	"github.com/harness/gitness/encrypt"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/git"
	"github.com/harness/gitness/git/api"
	"github.com/harness/gitness/git/storage"
	infraproviderpkg "github.com/harness/gitness/infraprovider"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/livelog"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/pubsub"
	"github.com/harness/gitness/ssh"
	"github.com/harness/gitness/store/database/dbtx"
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
		cliserver.ProvideBlobStoreConfig,
		mailer.WireSet,
		notification.WireSet,
		blob.WireSet,
		dbtx.WireSet,
		cache.WireSet,
		router.WireSet,
		pullreqservice.WireSet,
		services.WireSet,
		services.ProvideGitspaceServices,
		server.WireSet,
		url.WireSet,
		space.WireSet,
		limiter.WireSet,
		publicaccess.WireSet,
		repo.WireSet,
		reposettings.WireSet,
		pullreq.WireSet,
		controllerwebhook.WireSet,
		svclabel.WireSet,
		serviceaccount.WireSet,
		user.WireSet,
		upload.WireSet,
		service.WireSet,
		principal.WireSet,
		system.WireSet,
		authn.WireSet,
		authz.WireSet,
		infrastructure.WireSet,
		infraproviderpkg.WireSet,
		gitspaceevents.WireSet,
		infraproviderCtrl.WireSet,
		gitspaceCtrl.WireSet,
		gitevents.WireSet,
		pullreqevents.WireSet,
		repoevents.WireSet,
		storage.WireSet,
		api.WireSet,
		cliserver.ProvideGitConfig,
		git.WireSet,
		store.WireSet,
		check.WireSet,
		encrypt.WireSet,
		cliserver.ProvideEventsConfig,
		events.WireSet,
		cliserver.ProvideWebhookConfig,
		cliserver.ProvideNotificationConfig,
		webhook.WireSet,
		cliserver.ProvideTriggerConfig,
		trigger.WireSet,
		githookCtrl.ExtenderWireSet,
		githookCtrl.WireSet,
		cliserver.ProvideLockConfig,
		lock.WireSet,
		locker.WireSet,
		cliserver.ProvidePubsubConfig,
		pubsub.WireSet,
		cliserver.ProvideJobsConfig,
		job.WireSet,
		cliserver.ProvideCleanupConfig,
		cleanup.WireSet,
		codecomments.WireSet,
		protection.WireSet,
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
		converter.WireSet,
		runner.WireSet,
		sse.WireSet,
		scheduler.WireSet,
		commit.WireSet,
		controllertrigger.WireSet,
		plugin.WireSet,
		resolver.WireSet,
		importer.WireSet,
		migrateservice.WireSet,
		canceler.WireSet,
		exporter.WireSet,
		metric.WireSet,
		reposervice.WireSet,
		cliserver.ProvideCodeOwnerConfig,
		codeowners.WireSet,
		gitspaceevent.WireSet,
		cliserver.ProvideKeywordSearchConfig,
		keywordsearch.WireSet,
		controllerkeywordsearch.WireSet,
		settings.WireSet,
		usergroup.WireSet,
		openapi.WireSet,
		repo.ProvideRepoCheck,
		audit.WireSet,
		ssh.WireSet,
		publickey.WireSet,
		migrate.WireSet,
		scm.WireSet,
		orchestrator.WireSet,
		containerorchestrator.WireSet,
		cliserver.ProvideIDEVSCodeWebConfig,
		cliserver.ProvideDockerConfig,
		cliserver.ProvideGitspaceEventConfig,
		logutil.WireSet,
		cliserver.ProvideGitspaceOrchestratorConfig,
		ide.WireSet,
		gitspaceinfraevents.WireSet,
		gitspaceservice.WireSet,
		cliserver.ProvideGitspaceInfraProvisionerConfig,
		cliserver.ProvideIDEVSCodeConfig,
	)
	return &cliserver.System{}, nil
}
