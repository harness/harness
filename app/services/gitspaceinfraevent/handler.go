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

	gitspaceEvents "github.com/harness/gitness/app/events/gitspace"
	gitspaceInfraEvents "github.com/harness/gitness/app/events/gitspaceinfra"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

func (s *Service) handleGitspaceInfraEvent(
	ctx context.Context,
	event *events.Event[*gitspaceInfraEvents.GitspaceInfraEventPayload],
) error {
	payload := event.Payload

	config, fetchErr := s.getConfig(
		ctx, payload.Infra.SpaceID, payload.Infra.SpacePath, payload.Infra.GitspaceConfigIdentifier)
	if fetchErr != nil {
		return fetchErr
	}

	var instance = config.GitspaceInstance
	var err error

	switch payload.Type {
	case enum.InfraEventProvision:
		updatedInstance, err := s.orchestrator.ResumeStartGitspace(ctx, *config, payload.Infra)
		if err != nil {
			s.emitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceActionStartFailed)

			return fmt.Errorf("failed to resume start gitspace: %w", err)
		}

		instance = &updatedInstance

	case enum.InfraEventStop:
		instanceState, err := s.orchestrator.ResumeStopGitspace(ctx, *config, payload.Infra)
		if err != nil {
			s.emitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceActionStopFailed)

			return fmt.Errorf("failed to resume stop gitspace: %w", err)
		}

		instance.State = instanceState

	case enum.InfraEventDeprovision:
		instanceState, err := s.orchestrator.ResumeDeleteGitspace(ctx, *config, payload.Infra)
		if err != nil {
			return fmt.Errorf("failed to resume delete gitspace: %w", err)
		}

		instance.State = instanceState

		config.IsDeleted = true
		if err = s.gitspaceSvc.UpdateConfig(ctx, config); err != nil {
			return fmt.Errorf("failed to delete gitspace config with ID: %s %w", config.Identifier, err)
		}

	default:
		return fmt.Errorf("unknown event type: %s", event.Payload.Type)
	}

	err = s.gitspaceSvc.UpdateInstance(ctx, instance)
	if err != nil {
		return fmt.Errorf("failed to update gitspace instance: %w", err)
	}

	return nil
}

func (s *Service) getConfig(
	ctx context.Context,
	spaceID int64,
	spacePath string,
	identifier string,
) (*types.GitspaceConfig, error) {
	config, err := s.gitspaceSvc.Find(ctx, spaceID, spacePath, identifier)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to find gitspace config during infra event handling, identifier %s: %w", identifier, err)
	}
	return config, nil
}

func (s *Service) emitGitspaceConfigEvent(ctx context.Context,
	config *types.GitspaceConfig,
	eventType enum.GitspaceEventType,
) {
	s.eventReporter.EmitGitspaceEvent(ctx, gitspaceEvents.GitspaceEvent, &gitspaceEvents.GitspaceEventPayload{
		QueryKey:   config.Identifier,
		EntityID:   config.ID,
		EntityType: enum.GitspaceEntityTypeGitspaceConfig,
		EventType:  eventType,
		Timestamp:  time.Now().UnixNano(),
	})
}
