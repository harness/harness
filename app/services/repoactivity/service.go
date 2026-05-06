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

package repoactivity

import (
	"context"
	"errors"
	"fmt"
	"time"

	gitevents "github.com/harness/gitness/app/events/git"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"

	"github.com/rs/zerolog/log"
)

const (
	eventsReaderGroupName = "gitness:repo-activity"
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
	repoActivityStore store.RepoActivityStore
}

func New(
	ctx context.Context,
	config Config,
	repoActivityStore store.RepoActivityStore,
	gitReaderFactory *events.ReaderFactory[*gitevents.Reader],
) (*Service, error) {
	if err := config.Prepare(); err != nil {
		return nil, fmt.Errorf("provided repo activity service config is invalid: %w", err)
	}
	log.Ctx(ctx).Info().Msgf("[repo activity service] event reader name: %s, concurrency: %d, maxRetries: %d",
		config.EventReaderName, config.Concurrency, config.MaxRetries)

	service := &Service{repoActivityStore: repoActivityStore}

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

	return service, nil
}
