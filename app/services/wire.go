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
	"github.com/harness/gitness/app/services/cleanup"
	"github.com/harness/gitness/app/services/keywordsearch"
	"github.com/harness/gitness/app/services/metric"
	"github.com/harness/gitness/app/services/notification"
	"github.com/harness/gitness/app/services/pullreq"
	"github.com/harness/gitness/app/services/reposize"
	"github.com/harness/gitness/app/services/trigger"
	"github.com/harness/gitness/app/services/webhook"
	"github.com/harness/gitness/job"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideServices,
)

type Services struct {
	Webhook            *webhook.Service
	PullReq            *pullreq.Service
	Trigger            *trigger.Service
	JobScheduler       *job.Scheduler
	MetricCollector    *metric.Collector
	RepoSizeCalculator *reposize.Calculator
	Cleanup            *cleanup.Service
	Notification       *notification.Service
	Keywordsearch      *keywordsearch.Service
}

func ProvideServices(
	webhooksSvc *webhook.Service,
	pullReqSvc *pullreq.Service,
	triggerSvc *trigger.Service,
	jobScheduler *job.Scheduler,
	metricCollector *metric.Collector,
	repoSizeCalculator *reposize.Calculator,
	cleanupSvc *cleanup.Service,
	notificationSvc *notification.Service,
	keywordsearchSvc *keywordsearch.Service,
) Services {
	return Services{
		Webhook:            webhooksSvc,
		PullReq:            pullReqSvc,
		Trigger:            triggerSvc,
		JobScheduler:       jobScheduler,
		MetricCollector:    metricCollector,
		RepoSizeCalculator: repoSizeCalculator,
		Cleanup:            cleanupSvc,
		Notification:       notificationSvc,
		Keywordsearch:      keywordsearchSvc,
	}
}
