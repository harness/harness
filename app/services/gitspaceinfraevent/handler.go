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

	"github.com/rs/zerolog/log"
)

func (s *Service) handleGitspaceInfraEvent(
	ctx context.Context,
	event *events.Event[*gitspaceInfraEvents.GitspaceInfraEventPayload],
) error {
	payload := event.Payload

	config, fetchErr := s.getConfig(
		ctx, payload.Infra.SpacePath, payload.Infra.GitspaceConfigIdentifier)
	if fetchErr != nil {
		return fetchErr
	}

	instance := config.GitspaceInstance
	if payload.Infra.GitspaceInstanceIdentifier != "" {
		gitspaceInstance, err := s.gitspaceSvc.FindInstanceByIdentifier(
			ctx,
			payload.Infra.GitspaceInstanceIdentifier,
			payload.Infra.SpacePath,
		)
		if err != nil {
			return fmt.Errorf("failed to fetch gitspace instance: %w", err)
		}

		instance = gitspaceInstance
		config.GitspaceInstance = instance
	}

	defer func() {
		updateErr := s.gitspaceSvc.UpdateInstance(ctx, instance)
		if updateErr != nil {
			log.Err(updateErr).Msgf("failed to update gitspace instance")
		}
	}()

	var err error

	switch payload.Type {
	case enum.InfraEventProvision:
		updatedInstance, resumeStartErr := s.orchestrator.ResumeStartGitspace(ctx, *config, payload.Infra)
		if resumeStartErr != nil {
			s.emitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceActionStartFailed)
			updatedInstance.ErrorMessage = resumeStartErr.ErrorMessage
			err = fmt.Errorf("failed to resume start gitspace: %w", resumeStartErr.Error)
		}

		instance = &updatedInstance

	case enum.InfraEventStop:
		instanceState, resumeStopErr := s.orchestrator.ResumeStopGitspace(ctx, *config, payload.Infra)
		if resumeStopErr != nil {
			s.emitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeGitspaceActionStopFailed)
			instance.ErrorMessage = resumeStopErr.ErrorMessage
			err = fmt.Errorf("failed to resume stop gitspace: %w", resumeStopErr.Error)
		}

		instance.State = instanceState

	case enum.InfraEventDeprovision:
		instanceState, resumeDeleteErr := s.orchestrator.ResumeDeleteGitspace(ctx, *config, payload.Infra)
		if resumeDeleteErr != nil {
			err = fmt.Errorf("failed to resume delete gitspace: %w", resumeDeleteErr)
		} else if config.IsMarkedForDeletion {
			config.IsDeleted = true
			updateErr := s.gitspaceSvc.UpdateConfig(ctx, config)
			if updateErr != nil {
				err = fmt.Errorf("failed to delete gitspace config with ID: %s %w", config.Identifier, updateErr)
			}
		}

		instance.State = instanceState
	case enum.InfraEventCleanup:
		instanceState, resumeCleanupErr := s.orchestrator.ResumeCleanupInstanceResources(ctx, *config, payload.Infra)
		if resumeCleanupErr != nil {
			s.emitGitspaceConfigEvent(ctx, config, enum.GitspaceEventTypeInfraCleanupFailed)

			err = fmt.Errorf("failed to resume cleanup gitspace: %w", resumeCleanupErr)
		}

		instance.State = instanceState
	default:
		instance.State = enum.GitspaceInstanceStateError
		return fmt.Errorf("unknown event type: %s", event.Payload.Type)
	}

	if err != nil {
		log.Err(err).Msgf("error while handling gitspace infra event")
	}

	return nil
}

func (s *Service) getConfig(
	ctx context.Context,
	spaceRef string,
	identifier string,
) (*types.GitspaceConfig, error) {
	config, err := s.gitspaceSvc.Find(ctx, spaceRef, identifier)
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
