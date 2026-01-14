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

package branch

import (
	"context"
	"errors"
	"fmt"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	pullreqevents "github.com/harness/gitness/app/events/pullreq"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"

	"github.com/rs/zerolog/log"
)

const (
	eventsReaderGroupName = "gitness:branch"
	refsBranchPrefix      = "refs/heads/"
)

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

// Prepare validates the configuration.
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
	branchStore  store.BranchStore
	pullReqStore store.PullReqStore
}

func New(
	ctx context.Context,
	config Config,
	branchStore store.BranchStore,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
	pullreqReaderFactory *events.ReaderFactory[*pullreqevents.Reader],
	pullReqStore store.PullReqStore,
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided branch service config is invalid: %w", err)
	}
	log.Ctx(ctx).Info().Msgf("[branch service] event reader name: %s, concurrency: %d, maxRetries: %d",
		config.EventReaderName, config.Concurrency, config.MaxRetries)

	service := &Service{
		branchStore:  branchStore,
		pullReqStore: pullReqStore,
	}

	_, err := gitReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *gitevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterBranchCreated(service.handleEventBranchCreated)
			_ = r.RegisterBranchUpdated(service.handleEventBranchUpdated)
			_ = r.RegisterBranchDeleted(service.handleEventBranchDeleted)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch git events reader: %w", err)
	}

	// Register for pull request events to update branch information
	_, err = pullreqReaderFactory.Launch(ctx, eventsReaderGroupName, config.EventReaderName,
		func(r *pullreqevents.Reader) error {
			const idleTimeout = 10 * time.Second
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterCreated(service.handleEventPullReqCreated)
			_ = r.RegisterClosed(service.handleEventPullReqClosed)
			_ = r.RegisterReopened(service.handleEventPullReqReopened)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch pullreq events reader: %w", err)
	}

	return service, nil
}
