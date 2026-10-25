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

package agentexpiry

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/lock"
	"github.com/harness/gitness/types"

	"github.com/google/wire"
)

const (
	jobType    = "gitness:agent-attempt-expiry"
	jobCron    = "*/10 * * * *" // every 10 minutes
	jobTimeout = 5 * time.Minute
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
	config *types.Config,
	attempts store.AgentAttemptStore,
	taskSvc *task.Service,
	mutexes lock.MutexManager,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Service, error) {
	svc := New(attempts, taskSvc, mutexes, config.AIAgents.AttemptTimeout)
	if err := executor.Register(jobType, jobHandler{svc: svc}); err != nil {
		return nil, fmt.Errorf("failed to register agent attempt expiry job: %w", err)
	}
	if err := scheduler.AddRecurring(ctx, jobType, jobType, jobCron, jobTimeout); err != nil {
		return nil, fmt.Errorf("failed to schedule agent attempt expiry job: %w", err)
	}
	return svc, nil
}
