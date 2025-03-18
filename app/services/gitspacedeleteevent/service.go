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

package gitspacedeleteevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	gitspacedeleteevents "github.com/harness/gitness/app/events/gitspacedelete"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
)

const groupGitspaceDeleteEvents = "gitness:gitspace_delete"

type Config struct {
	EventReaderName string
	Concurrency     int
	MaxRetries      int
	TimeoutInMins   int
}

func (c *Config) Sanitize() error {
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
	config      *Config
	gitspaceSvc *gitspace.Service
}

func NewService(
	ctx context.Context,
	config *Config,
	gitspaceDeleteEventReaderFactory *events.ReaderFactory[*gitspacedeleteevents.Reader],
	gitspaceSvc *gitspace.Service,
) (*Service, error) {
	if err := config.Sanitize(); err != nil {
		return nil, fmt.Errorf("provided gitspace event service config is invalid: %w", err)
	}
	service := &Service{
		config:      config,
		gitspaceSvc: gitspaceSvc,
	}

	_, err := gitspaceDeleteEventReaderFactory.Launch(ctx, groupGitspaceDeleteEvents, config.EventReaderName,
		func(r *gitspacedeleteevents.Reader) error {
			var idleTimeout = time.Duration(config.TimeoutInMins) * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterGitspaceDeleteEvent(service.handleGitspaceDeleteEvent)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch gitspace delete event reader: %w", err)
	}

	return service, nil
}
