// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package services

import (
	"github.com/harness/gitness/internal/services/job"
	"github.com/harness/gitness/internal/services/pullreq"
	"github.com/harness/gitness/internal/services/webhook"

	"github.com/google/wire"
)

var WireSet = wire.NewSet(
	ProvideServices,
)

type Services struct {
	Webhook      *webhook.Service
	PullReq      *pullreq.Service
	JobExecutor  *job.Executor
	JobScheduler *job.Scheduler
}

func ProvideServices(
	webhooksSrv *webhook.Service,
	pullReqSrv *pullreq.Service,
	jobExecutor *job.Executor,
	jobScheduler *job.Scheduler,
) Services {
	return Services{
		Webhook:      webhooksSrv,
		PullReq:      pullReqSrv,
		JobExecutor:  jobExecutor,
		JobScheduler: jobScheduler,
	}
}
