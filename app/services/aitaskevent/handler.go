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

package aitaskevent

import (
	"context"
	"errors"
	"fmt"
	"time"

	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var ErrNilResource = errors.New("nil resource")

func (s *Service) handleAITaskEvent(
	ctx context.Context,
	event *events.Event[*aitaskevents.AITaskEventPayload],
) error {
	logr := log.With().Str("event", string(event.Payload.Type)).Logger()
	logr.Debug().Msgf("Received AI task event, identifier: %s", event.Payload.AITaskIdentifier)

	payload := event.Payload
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutInMins)*time.Minute)
	defer cancel()

	aiTask, err := s.fetchWithRetry(
		ctxWithTimeOut,
		event.Payload.AITaskIdentifier,
		event.Payload.AITaskSpaceID,
	)
	if err != nil {
		logr.Error().Err(err).Msgf("failed to find AI task: %s", aiTask.Identifier)
		return fmt.Errorf("failed to get AI task: %w", err)
	}
	if aiTask == nil {
		logr.Error().Msg("failed to find AI task: ai task is nil")
		return fmt.Errorf("failed to find ai task: %w", ErrNilResource)
	}

	// mark ai task as running
	aiTask.State = enum.AITaskStateRunning
	err = s.aiTaskStore.Update(ctx, aiTask)
	if err != nil {
		return fmt.Errorf("failed to update aiTask state: %w", err)
	}

	gitspaceConfig, err := s.gitspaceSvc.FindWithLatestInstanceByID(ctx, aiTask.GitspaceConfigID, false)
	if err != nil {
		return fmt.Errorf("failed to get gitspace config: %w", err)
	}
	if gitspaceConfig == nil {
		return fmt.Errorf("failed to find gitspace config: %w", ErrNilResource)
	}

	// Handle the AI task event based on the task state or other logic
	logr.Info().Msgf("Processing AI task %s event: %s", aiTask.Identifier, payload.Type)

	var handleEventErr error
	switch payload.Type {
	case enum.AITaskEventStart:
		handleEventErr = s.handleStartEvent(ctx, *aiTask, *gitspaceConfig, logr)
	case enum.AITaskEventStop:
		handleEventErr = s.handleStopEvent(ctx, payload)
	default:
		handleEventErr = fmt.Errorf("invalid AI task event type: %s", payload.Type)
	}

	aiTask.State = enum.AITaskStateRunning
	if handleEventErr != nil {
		logr.Error().Err(handleEventErr).Msgf("failed to handle AI task event: %s, aiTask ID: %s",
			payload.Type, aiTask.Identifier)

		aiTask.State = enum.AITaskStateError
		errStr := handleEventErr.Error()
		aiTask.ErrorMessage = &errStr
	}

	err = s.aiTaskStore.Update(ctx, aiTask)
	if err != nil {
		return fmt.Errorf("failed to update aiTask state: %w", err)
	}

	return nil
}

// fetchWithRetry trys to fetch ai task from db with retry only for case where AI task is not found.
func (s *Service) fetchWithRetry(ctx context.Context, aiTaskID string, spaceID int64) (*types.AITask, error) {
	for i := 0; i < 3; i++ {
		aiTask, err := s.aiTaskStore.FindByIdentifier(ctx, spaceID, aiTaskID)
		if err != nil && !errors.Is(err, store.ErrResourceNotFound) {
			// if error is not resource not found, return error
			return nil, err
		}
		if err == nil {
			return aiTask, nil
		}

		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("failed to find ai task: %w", ErrNilResource)
}

func (s *Service) handleStartEvent(
	ctx context.Context,
	aiTask types.AITask,
	gitspaceConfig types.GitspaceConfig,
	logr zerolog.Logger,
) error {
	switch aiTask.State {
	case enum.AITaskStateUninitialized, enum.AITaskStateRunning, enum.AITaskStateError:
		logr.Debug().Msgf("ai task: %s is starting from %s state", aiTask.Identifier,
			aiTask.State)
		// continue
	case enum.AITaskStateCompleted:
		logr.Debug().Msgf("ai task: %s already completed", aiTask.Identifier)
		return nil
	default:
		logr.Debug().Msgf("ai task: %s in invalid state %s", aiTask.Identifier, aiTask.State)
		return fmt.Errorf("ai task: %s in invalid state %s", aiTask.Identifier, aiTask.State)
	}

	// validate before triggering ai task
	if gitspaceConfig.State != enum.GitspaceStateRunning {
		return fmt.Errorf("gitspace is not running, current: %s, expected: %s", gitspaceConfig.State,
			enum.GitspaceStateRunning)
	}

	return s.orchestrator.TriggerAITask(ctx, aiTask, gitspaceConfig)
}

// handleStopEvent is NOOP as we currently do not support stopping of ai task.
func (s *Service) handleStopEvent(ctx context.Context, eventPayload *aitaskevents.AITaskEventPayload) error {
	return nil
}
