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
	"fmt"
	"time"

	gitspaceevents "github.com/harness/gitness/app/events/gitspace"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/types"
)

func (s *Service) handleGitspaceEvent(
	ctx context.Context,
	event *events.Event[*gitspaceevents.GitspaceEventPayload],
) error {
	gitspaceEvent := &types.GitspaceEvent{
		Event:      event.Payload.EventType,
		EntityID:   event.Payload.EntityID,
		QueryKey:   event.Payload.QueryKey,
		EntityType: event.Payload.EntityType,
		Timestamp:  event.Payload.Timestamp,
		Created:    time.Now().UnixMilli(),
	}

	err := s.gitspaceEventStore.Create(ctx, gitspaceEvent)
	if err != nil {
		return fmt.Errorf("failed to create gitspace event: %w", err)
	}

	return nil
}
