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

package gitspaceevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
)

const groupGitspaceEvents = "gitness:gitspace"

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
}

func (c *Config) Sanitize() error {
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
	config             *Config
	gitspaceEventStore store.GitspaceEventStore
}

func NewService(
	ctx context.Context,
	config *Config,
	gitspaceEventReaderFactory *events.ReaderFactory[*gitspaceevents.Reader],
	gitspaceEventStore store.GitspaceEventStore,
) (*Service, error) {
	if err := config.Sanitize(); err != nil {
		return nil, fmt.Errorf("provided gitspace event service config is invalid: %w", err)
	}
	service := &Service{
		config:             config,
		gitspaceEventStore: gitspaceEventStore,
	}

	_, err := gitspaceEventReaderFactory.Launch(ctx, groupGitspaceEvents, config.EventReaderName,
		func(r *gitspaceevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			// register gitspace config events
			_ = r.RegisterGitspaceEvent(service.handleGitspaceEvent)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch gitspace event reader: %w", err)
	}

	return service, nil
}
