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

package gitspaceoperationsevent

import (
	"context"
	"fmt"
	"time"

	gitspaceEvents "github.com/harness/gitness/app/events/gitspace"
	gitspaceOperationsEvents "github.com/harness/gitness/app/events/gitspaceoperations"
	"github.com/harness/gitness/app/gitspace/orchestrator/container/response"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (s *Service) handleGitspaceOperationsEvent(
	ctx context.Context,
	event *events.Event[*gitspaceOperationsEvents.GitspaceOperationsEventPayload],
) error {
	logr := log.With().Str("event", string(event.Payload.Type)).Logger()
	logr.Debug().Msg("Received gitspace operations event")

	payload := event.Payload
	ctxWithTimedOut, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutInMins)*time.Minute)
	defer cancel()
	config, fetchErr := s.getConfig(
		ctxWithTimedOut, payload.Infra.SpacePath, payload.Infra.GitspaceConfigIdentifier)
	if fetchErr != nil {
		return fetchErr
	}

	instance := config.GitspaceInstance
	if payload.Infra.GitspaceInstanceIdentifier != "" {
		gitspaceInstance, err := s.gitspaceSvc.FindInstanceByIdentifier(
			ctxWithTimedOut,
			payload.Infra.GitspaceInstanceIdentifier,
		)
		if err != nil {
			return fmt.Errorf("failed to fetch gitspace instance: %w", err)
		}

		instance = gitspaceInstance
		config.GitspaceInstance = instance
	}

	defer func() {
		updateErr := s.gitspaceSvc.UpdateInstance(ctxWithTimedOut, instance)
		if updateErr != nil {
			log.Err(updateErr).Msgf("failed to update gitspace instance")
		}
	}()

	var err error

	switch payload.Type {
	case enum.GitspaceOperationsEventStart:
		if config.GitspaceInstance.Identifier != payload.Infra.GitspaceInstanceIdentifier {
			return fmt.Errorf("gitspace instance is not latest, stopping provisioning")
		}

		startResponse, ok := payload.Response.(*response.StartResponse)
		if !ok {
			return fmt.Errorf("failed to cast start response")
		}
		updatedInstance, handleResumeStartErr := s.orchestrator.FinishResumeStartGitspace(
			ctxWithTimedOut,
			*config,
			payload.Infra,
			startResponse,
		)
		if handleResumeStartErr != nil {
			s.emitGitspaceConfigEvent(ctxWithTimedOut, config, enum.GitspaceEventTypeGitspaceActionStartFailed)
			updatedInstance.ErrorMessage = handleResumeStartErr.ErrorMessage
			err = fmt.Errorf("failed to finish resume start gitspace: %w", handleResumeStartErr.Error)
		}
		instance = &updatedInstance
	case enum.GitspaceOperationsEventStop:
		finishStopErr := s.orchestrator.FinishStopGitspaceContainer(ctxWithTimedOut, *config, payload.Infra)
		if finishStopErr != nil {
			s.emitGitspaceConfigEvent(ctxWithTimedOut, config, enum.GitspaceEventTypeGitspaceActionStopFailed)
			instance.ErrorMessage = finishStopErr.ErrorMessage
			err = fmt.Errorf("failed to finish trigger start gitspace: %w", finishStopErr.Error)
		}
	case enum.GitspaceOperationsEventDelete:
		deleteResponse, ok := payload.Response.(*response.DeleteResponse)
		if !ok {
			return fmt.Errorf("failed to cast delete response")
		}
		finishStopAndRemoveErr := s.orchestrator.FinishStopAndRemoveGitspaceContainer(
			ctxWithTimedOut,
			*config,
			payload.Infra,
			deleteResponse.CanDeleteUserData,
		)
		if finishStopAndRemoveErr != nil {
			s.emitGitspaceConfigEvent(ctxWithTimedOut, config, enum.GitspaceEventTypeGitspaceActionStopFailed)
			instance.ErrorMessage = finishStopAndRemoveErr.ErrorMessage
			err = fmt.Errorf("failed to finish trigger start gitspace: %w", finishStopAndRemoveErr.Error)
		}

	default:
		return fmt.Errorf("unknown event type: %s", event.Payload.Type)
	}

	if err != nil {
		log.Err(err).Msgf("error while handling gitspace operations event")
	}

	return nil
}

func (s *Service) getConfig(
	ctx context.Context,
	spaceRef string,
	identifier string,
) (*types.GitspaceConfig, error) {
	config, err := s.gitspaceSvc.FindWithLatestInstanceWithSpacePath(ctx, spaceRef, identifier)
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
