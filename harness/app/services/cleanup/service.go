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

package cleanup

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/app/api/controller/repo"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/job"
)

type Config struct {
	WebhookExecutionsRetentionTime   time.Duration
	DeletedRepositoriesRetentionTime time.Duration
}

func (c *Config) Prepare() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.WebhookExecutionsRetentionTime <= 0 {
		return errors.New("config.WebhookExecutionsRetentionTime has to be provided")
	}

	if c.DeletedRepositoriesRetentionTime <= 0 {
		return errors.New("config.DeletedRepositoriesRetentionTime has to be provided")
	}
	return nil
}

// Service is responsible for cleaning up data in db / git / ...
type Service struct {
	config                Config
	scheduler             *job.Scheduler
	executor              *job.Executor
	webhookExecutionStore store.WebhookExecutionStore
	tokenStore            store.TokenStore
	repoStore             store.RepoStore
	repoCtrl              *repo.Controller
}

func NewService(
	config Config,
	scheduler *job.Scheduler,
	executor *job.Executor,
	webhookExecutionStore store.WebhookExecutionStore,
	tokenStore store.TokenStore,
	repoStore store.RepoStore,
	repoCtrl *repo.Controller,
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided cleanup config is invalid: %w", err)
	}

	return &Service{
		config: config,

		scheduler:             scheduler,
		executor:              executor,
		webhookExecutionStore: webhookExecutionStore,
		tokenStore:            tokenStore,
		repoStore:             repoStore,
		repoCtrl:              repoCtrl,
	}, nil
}

func (s *Service) Register(ctx context.Context) error {
	if err := s.registerJobHandlers(); err != nil {
		return fmt.Errorf("failed to register cleanup job handlers: %w", err)
	}

	if err := s.scheduleRecurringCleanupJobs(ctx); err != nil {
		return fmt.Errorf("failed to schedule cleanup jobs: %w", err)
	}

	return nil
}

// scheduleRecurringCleanupJobs schedules the cleanup jobs.
func (s *Service) scheduleRecurringCleanupJobs(ctx context.Context) error {
	err := s.scheduler.AddRecurring(
		ctx,
		jobTypeWebhookExecutions,
		jobTypeWebhookExecutions,
		jobCronWebhookExecutions,
		jobMaxDurationWebhookExecutions,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule webhook executions job: %w", err)
	}

	err = s.scheduler.AddRecurring(
		ctx,
		jobTypeTokens,
		jobTypeTokens,
		jobCronTokens,
		jobMaxDurationTokens,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule token job: %w", err)
	}

	err = s.scheduler.AddRecurring(
		ctx,
		jobTypeDeletedRepos,
		jobTypeDeletedRepos,
		jobCronDeletedRepos,
		jobMaxDurationDeletedRepos,
	)
	if err != nil {
		return fmt.Errorf("failed to schedule deleted repo cleanup job: %w", err)
	}
	return nil
}

// registerJobHandlers registers handlers for all cleanup jobs.
func (s *Service) registerJobHandlers() error {
	if err := s.executor.Register(
		jobTypeWebhookExecutions,
		newWebhookExecutionsCleanupJob(
			s.config.WebhookExecutionsRetentionTime,
			s.webhookExecutionStore,
		),
	); err != nil {
		return fmt.Errorf("failed to register job handler for webhook executions cleanup: %w", err)
	}

	if err := s.executor.Register(
		jobTypeTokens,
		newTokensCleanupJob(
			s.tokenStore,
		),
	); err != nil {
		return fmt.Errorf("failed to register job handler for token cleanup: %w", err)
	}

	if err := s.executor.Register(
		jobTypeDeletedRepos,
		newDeletedReposCleanupJob(
			s.config.DeletedRepositoriesRetentionTime,
			s.repoStore,
			s.repoCtrl,
		),
	); err != nil {
		return fmt.Errorf("failed to register job handler for deleted repos cleanup: %w", err)
	}
	return nil
}
