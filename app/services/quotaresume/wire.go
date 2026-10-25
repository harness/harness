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

package quotaresume

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/aigateway"
	"github.com/harness/gitness/app/services/insights/triage"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types/enum"

	"github.com/google/wire"
)

const (
	jobType    = "gitness:quota-resume"
	jobCron    = "*/15 * * * *" // every 15 minutes, includes 00:00 UTC when the daily quota resets
	jobTimeout = 10 * time.Minute
)

// WireSet provides a wire set for this package.
var WireSet = wire.NewSet(
	ProvideService,
)

type jobHandler struct{ svc *Service }

// Handle implements job.Handler.
func (h jobHandler) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	res, err := h.svc.Run(ctx)
	return res.String(), err
}

// ProvideService creates the service and schedules its recurring job.
func ProvideService(
	ctx context.Context,
	tasks store.TaskStore,
	links store.TaskLinkStore,
	taskSvc *task.Service,
	gateway *aigateway.Gateway,
	observer *triage.Observer,
	mutexes lock.MutexManager,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Service, error) {
	svc := New(tasks, taskSvc, gateway.Planner(), mutexes, map[enum.TaskOrigin]Resumer{
		enum.TaskOriginInsights: InsightsResumer(links, observer),
		enum.TaskOriginShield:   ShieldResumer(),
	})
	if err := executor.Register(jobType, jobHandler{svc: svc}); err != nil {
		return nil, fmt.Errorf("failed to register quota resume job: %w", err)
	}
	if err := scheduler.AddRecurring(ctx, jobType, jobType, jobCron, jobTimeout); err != nil {
		return nil, fmt.Errorf("failed to schedule quota resume job: %w", err)
	}
	return svc, nil
}
