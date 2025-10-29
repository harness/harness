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
	"fmt"

	gitspacedeleteevents "github.com/harness/gitness/app/events/gitspacedelete"
	"github.com/harness/gitness/events"

	"github.com/rs/zerolog/log"
)

func (s *Service) handleGitspaceDeleteEvent(
	ctx context.Context,
	event *events.Event[*gitspacedeleteevents.GitspaceDeleteEventPayload],
) error {
	log.Debug().Msgf("handling gitspace delete event with payload: %+v", event.Payload)
	gitspaceConfigIdentifier := event.Payload.GitspaceConfigIdentifier
	spaceID := event.Payload.SpaceID
	gitspaceConfig, err := s.gitspaceSvc.FindWithLatestInstance(ctx, spaceID, gitspaceConfigIdentifier)
	if err != nil {
		return fmt.Errorf("failed to find gitspace config %s for space %d while handling delete event: %w",
			gitspaceConfigIdentifier, spaceID, err)
	}

	err = s.gitspaceSvc.RemoveGitspace(ctx, *gitspaceConfig, true)
	if err != nil {
		// NOTE: No need to retry from the event handler. The background job will take care.
		log.Debug().Err(err).Msgf("unable to delete gitspace: %s", gitspaceConfigIdentifier)
	}

	log.Debug().Msgf("handled gitspace delete event with payload: %+v", event.Payload)
	return nil
}
