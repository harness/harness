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

package verify

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/bootstrap"
	"github.com/harness/gitness/app/services/insights"
	"github.com/harness/gitness/app/services/task"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
	"github.com/harness/gitness/types"
)

const (
	jobType    = "insights-verify"
	jobTimeout = 5 * time.Minute
)

// Service wires the verification job and the reopen observer.
type Service struct {
	Verifier *Verifier
	Observer *ReopenObserver
}

// Handle implements job.Handler.
func (s *Service) Handle(ctx context.Context, _ string, _ job.ProgressReporter) (string, error) {
	res, err := s.Verifier.Run(ctx)
	if err != nil {
		return "", err
	}
	return res.String(), nil
}

// NewService creates the service, registers the recurring job (if enabled) and attaches the
// reopen observer to the Insights service next to any other observer.
func NewService(
	ctx context.Context,
	config *types.Config,
	insightsSvc *insights.Service,
	taskSvc *task.Service,
	linkStore store.TaskLinkStore,
	linkedRefs store.TaskLinkedRefStore,
	taskStore store.TaskStore,
	issueStore store.InsightsIssueStore,
	releaseStore store.InsightsReleaseStore,
	scheduler *job.Scheduler,
	executor *job.Executor,
) (*Service, error) {
	principal := func() int64 { return bootstrap.NewSystemServiceSession().Principal.ID }
	cfg := Config{
		VerifyWindow: config.InsightsVerify.VerifyWindow,
		QuietPeriod:  config.InsightsVerify.QuietPeriod,
	}
	svc := &Service{
		Verifier: NewVerifier(cfg, linkedRefs, issueStore, releaseStore, taskSvc, principal),
		Observer: NewReopenObserver(linkStore, taskStore.Find, taskSvc, principal),
	}
	if !config.InsightsVerify.Enabled {
		return svc, nil
	}

	insightsSvc.AddObserver(svc.Observer)

	if err := executor.Register(jobType, svc); err != nil {
		return nil, fmt.Errorf("failed to register insights verify job: %w", err)
	}
	err := scheduler.AddRecurring(ctx, jobType, jobType, config.InsightsVerify.Cron, jobTimeout)
	if err != nil {
		return nil, fmt.Errorf("failed to schedule insights verify job: %w", err)
	}
	return svc, nil
}
