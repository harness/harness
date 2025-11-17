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
	"fmt"
	"time"

	aitaskevents "github.com/harness/gitness/app/events/aitask"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types/enum"

	"github.com/rs/zerolog/log"
)

func (s *Service) handleAITaskEvent(
	ctx context.Context,
	event *events.Event[*aitaskevents.AITaskEventPayload],
) error {
	logr := log.With().Str("event", string(event.Payload.Type)).Logger()
	logr.Debug().Msgf("Received AI task event, identifier: %s", event.Payload.AITask.Identifier)

	payload := event.Payload
	ctxWithTimeOut, cancel := context.WithTimeout(ctx, time.Duration(s.config.TimeoutInMins)*time.Minute)
	defer cancel()

	aiTask, err := s.aiTaskStore.FindByIdentifier(
		ctxWithTimeOut,
		event.Payload.AITask.SpaceID,
		event.Payload.AITask.Identifier,
	)
	if err != nil {
		return fmt.Errorf("failed to get AI task: %w", err)
	}

	if aiTask.State != enum.AITaskStateUninitialized {
		logr.Error().Msgf("ai task is in invalid state, current: %s, expected: %s",
			aiTask.State, enum.AITaskStateUninitialized,
		)
		return fmt.Errorf("ai task is in invalid state, current: %s, expected: %s",
			aiTask.State, enum.AITaskStateUninitialized,
		)
	}

	// Handle the AI task event based on the task state or other logic
	// This is a placeholder for actual business logic
	logr.Info().Msgf("Processing AI task %s event: %s", aiTask.Identifier, payload.Type)

	return nil
}
