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

package tasksla

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/services/notification"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
)

const (
	jobType    = "task-sla"
	jobTimeout = 5 * time.Minute
)

// Service is the recurring SLA notification job.
type Service struct {
	Runner *Runner
}

// Handle implements job.Handler.
func (s *Service) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	res, err := s.Runner.Run(ctx)
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

// NewService creates the service and registers the recurring job (if enabled).
func NewService(
	ctx context.Context,
	config *types.Config,
	tasks store.TaskStore,
	activities store.TaskActivityStore,
	notifier *notification.Service,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Service, error) {
	svc := &Service{Runner: NewRunner(config.TaskSLA.Window, tasks, activities, notifier)}
	if !config.TaskSLA.Enabled {
		return svc, nil
	}
	if err := executor.Register(jobType, svc); err != nil {
		return nil, fmt.Errorf("failed to register task sla job: %w", err)
	}
	if err := scheduler.AddRecurring(ctx, jobType, jobType, config.TaskSLA.Cron, jobTimeout); err != nil {
		return nil, fmt.Errorf("failed to schedule task sla job: %w", err)
	}
	return svc, nil
}
