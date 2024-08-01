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

package gitspaceinfraevent

import (
	"context"
	"fmt"
	"time"

	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	gitspaceinfraevents "github.com/harness/gitness/app/events/gitspaceinfra"
	"github.com/harness/gitness/app/gitspace/orchestrator"
	"github.com/harness/gitness/app/services/gitspace"
	"github.com/harness/gitness/app/services/gitspaceevent"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/stream"
)

const groupGitspaceInfraEvents = "gitness:gitspaceinfra"

type Service struct {
	config        *gitspaceevent.Config
	orchestrator  orchestrator.Orchestrator
	gitspaceSvc   *gitspace.Service
	eventReporter *gitspaceevents.Reporter
}

func NewService(
	ctx context.Context,
	config *gitspaceevent.Config,
	gitspaceInfraEventReaderFactory *events.ReaderFactory[*gitspaceinfraevents.Reader],
	orchestrator orchestrator.Orchestrator,
	gitspaceSvc *gitspace.Service,
	eventReporter *gitspaceevents.Reporter,
) (*Service, error) {
	if err := config.Sanitize(); err != nil {
		return nil, fmt.Errorf("provided gitspace infra event service config is invalid: %w", err)
	}
	service := &Service{
		config:        config,
		orchestrator:  orchestrator,
		gitspaceSvc:   gitspaceSvc,
		eventReporter: eventReporter,
	}

	_, err := gitspaceInfraEventReaderFactory.Launch(ctx, groupGitspaceInfraEvents, config.EventReaderName,
		func(r *gitspaceinfraevents.Reader) error {
			const idleTimeout = 1 * time.Minute
			r.Configure(
				stream.WithConcurrency(config.Concurrency),
				stream.WithHandlerOptions(
					stream.WithIdleTimeout(idleTimeout),
					stream.WithMaxRetries(config.MaxRetries),
				))

			_ = r.RegisterGitspaceInfraEvent(service.handleGitspaceInfraEvent)

			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("failed to launch gitspace infra event reader: %w", err)
	}

	return service, nil
}
