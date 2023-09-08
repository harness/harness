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
	"github.com/harness/gitness/stream"
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

type Service struct{}

func New(
	ctx context.Context,
	config Config,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqEvReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided trigger service config is invalid: %w", err)
	}

	service := &Service{}

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
					stream.WithMaxRetries(config.MaxRetries),
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
