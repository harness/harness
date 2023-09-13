// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package trigger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/harness/gitness/events"
	gitevents "github.com/harness/gitness/internal/events/git"
	pullreqevents "github.com/harness/gitness/internal/events/pullreq"
	"github.com/harness/gitness/internal/pipeline/commit"
	"github.com/harness/gitness/internal/pipeline/triggerer"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/stream"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/hashicorp/go-multierror"
)

const (
	eventsReaderGroupName = "gitness:trigger"
)

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

func (c *Config) Prepare() error {
	if c == nil {
		return errors.New("config is required")
	}
	if c.EventReaderName == "" {
		return errors.New("config.EventReaderName is required")
	}
	if c.Concurrency < 1 {
		return errors.New("config.Concurrency has to be a positive number")
	}
	if c.MaxRetries < 0 {
		return errors.New("config.MaxRetries can't be negative")
	}

	return nil
}

type Service struct {
	triggerStore  store.TriggerStore
	pullReqStore  store.PullReqStore
	repoStore     store.RepoStore
	pipelineStore store.PipelineStore
	triggerSvc    triggerer.Triggerer
	commitSvc     commit.CommitService
}

func New(
	ctx context.Context,
	config Config,
	triggerStore store.TriggerStore,
	pullReqStore store.PullReqStore,
	repoStore store.RepoStore,
	pipelineStore store.PipelineStore,
	triggerSvc triggerer.Triggerer,
	commitSvc commit.CommitService,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided trigger service config is invalid: %w", err)
	}

	service := &Service{
		triggerStore:  triggerStore,
		pullReqStore:  pullReqStore,
		repoStore:     repoStore,
		commitSvc:     commitSvc,
		pipelineStore: pipelineStore,
		triggerSvc:    triggerSvc,
	}

	_, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *gitevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterBranchCreated(service.handleEventBranchCreated)
			_ = r.RegisterBranchUpdated(service.handleEventBranchUpdated)

			_ = r.RegisterTagCreated(service.handleEventTagCreated)
			_ = r.RegisterTagUpdated(service.handleEventTagUpdated)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch git events reader: %w", err)
	}

	_, err = pullreqEvReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					// retries not needed for builds which failed to trigger, can be adjusted when needed
					stream.WithMaxRetries(0),
				))

			_ = r.RegisterCreated(service.handleEventPullReqCreated)
			_ = r.RegisterBranchUpdated(service.handleEventPullReqBranchUpdated)
			_ = r.RegisterReopened(service.handleEventPullReqReopened)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch pr events reader: %w", err)
	}

	return service, nil
}

// trigger a build given an action on a repo and a hook.
// It tries to find all enabled triggers, see if the action is the same
// as the trigger action - and if so, find the pipeline for the trigger
// and fire an execution.
func (s *Service) trigger(ctx context.Context, repoID int64,
	action enum.TriggerAction, hook *triggerer.Hook) error {
	// Get all enabled triggers for a repo.
	ret, err := s.triggerStore.ListAllEnabled(ctx, repoID)
	if err != nil {
		return fmt.Errorf("failed to list all enabled triggers: %w", err)
	}
	validTriggers := []*types.Trigger{}
	// Check which triggers are eligible to be fired
	for _, t := range ret {
		for _, a := range t.Actions {
			if a == action {
				validTriggers = append(validTriggers, t)
				break
			}
		}
	}

	var errs error
	for _, t := range validTriggers {
		pipeline, err := s.pipelineStore.Find(ctx, t.PipelineID)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		_, err = s.triggerSvc.Trigger(ctx, pipeline, hook)
		if err != nil {
			errs = multierror.Append(errs, err)
		}
	}
	return errs
}
