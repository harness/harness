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

package services

import (
	"github.com/harness/gitness/app/services/branch"
	"github.com/harness/gitness/app/services/cleanup"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/gitspacedeleteevent"
	"github.com/harness/gitness/app/services/gitspaceevent"
	"github.com/harness/gitness/app/services/gitspaceinfraevent"
	"github.com/harness/gitness/app/services/gitspaceoperationsevent"
	"github.com/harness/gitness/app/services/infraprovider"
	"github.com/harness/gitness/app/services/instrument"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/metric"
	"github.com/harness/gitness/app/services/notification"
	"github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/repo"
	"github.com/harness/gitness/app/services/trigger"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/job"
	registrywebhooks "github.com/harness/gitness/registry/services/webhook"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideServices,
)

type Services struct {
	Webhook                 *webhook.Service
	PullReq                 *pullreq.Service
	Trigger                 *trigger.Service
	JobScheduler            *job.Scheduler
	MetricCollector         *metric.CollectorJob
	RepoSizeCalculator      *repo.SizeCalculator
	Repo                    *repo.Service
	Cleanup                 *cleanup.Service
	Notification            *notification.Service
	Keywordsearch           *keywordsearch.Service
	GitspaceService         *GitspaceServices
	Instrumentation         instrument.Service
	instrumentConsumer      instrument.Consumer
	instrumentRepoCounter   *instrument.RepositoryCount
	registryWebhooksService *registrywebhooks.Service
	Branch                  *branch.Service
}

type GitspaceServices struct {
	GitspaceEvent              *gitspaceevent.Service
	infraProvider              *infraprovider.Service
	gitspace                   *gitspace.Service
	gitspaceInfraEventSvc      *gitspaceinfraevent.Service
	gitspaceOperationsEventSvc *gitspaceoperationsevent.Service
	gitspaceDeleteEventSvc     *gitspacedeleteevent.Service
}

func ProvideGitspaceServices(
	gitspaceEventSvc *gitspaceevent.Service,
	gitspaceDeleteEventSvc *gitspacedeleteevent.Service,
	infraProviderSvc *infraprovider.Service,
	gitspaceSvc *gitspace.Service,
	gitspaceInfraEventSvc *gitspaceinfraevent.Service,
	gitspaceOperationsEventSvc *gitspaceoperationsevent.Service,
) *GitspaceServices {
	return &GitspaceServices{
		GitspaceEvent:              gitspaceEventSvc,
		infraProvider:              infraProviderSvc,
		gitspace:                   gitspaceSvc,
		gitspaceInfraEventSvc:      gitspaceInfraEventSvc,
		gitspaceOperationsEventSvc: gitspaceOperationsEventSvc,
		gitspaceDeleteEventSvc:     gitspaceDeleteEventSvc,
	}
}

func ProvideServices(
	webhooksSvc *webhook.Service,
	pullReqSvc *pullreq.Service,
	triggerSvc *trigger.Service,
	jobScheduler *job.Scheduler,
	metricCollector *metric.CollectorJob,
	repoSizeCalculator *repo.SizeCalculator,
	repo *repo.Service,
	cleanupSvc *cleanup.Service,
	notificationSvc *notification.Service,
	keywordsearchSvc *keywordsearch.Service,
	gitspaceSvc *GitspaceServices,
	instrumentation instrument.Service,
	instrumentConsumer instrument.Consumer,
	instrumentRepoCounter *instrument.RepositoryCount,
	registryWebhooksService *registrywebhooks.Service,
	branchSvc *branch.Service,
) Services {
	return Services{
		Webhook:                 webhooksSvc,
		PullReq:                 pullReqSvc,
		Trigger:                 triggerSvc,
		JobScheduler:            jobScheduler,
		MetricCollector:         metricCollector,
		RepoSizeCalculator:      repoSizeCalculator,
		Repo:                    repo,
		Cleanup:                 cleanupSvc,
		Notification:            notificationSvc,
		Keywordsearch:           keywordsearchSvc,
		GitspaceService:         gitspaceSvc,
		Instrumentation:         instrumentation,
		instrumentConsumer:      instrumentConsumer,
		instrumentRepoCounter:   instrumentRepoCounter,
		registryWebhooksService: registryWebhooksService,
		Branch:                  branchSvc,
	}
}
