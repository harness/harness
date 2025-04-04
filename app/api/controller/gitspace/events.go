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

package gitspace

import (
	"context"
	"fmt"
	"time"

	apiauth "github.com/harness/gitness/app/api/auth"
	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

var eventMessageMap map[enum.GitspaceEventType]string

func init() {
	eventMessageMap = enum.EventsMessageMapping()
}

func (c *Controller) Events(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	page int,
	limit int,
) ([]*types.GitspaceEventResponse, int, error) {
	space, err := c.spaceFinder.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize: %w", err)
	}

	pagination := types.Pagination{
		Page: page,
		Size: limit,
	}
	skipEvents := []enum.GitspaceEventType{
		enum.GitspaceEventTypeInfraCleanupStart,
		enum.GitspaceEventTypeInfraCleanupCompleted,
		enum.GitspaceEventTypeInfraCleanupFailed,
	}
	filter := &types.GitspaceEventFilter{
		Pagination: pagination,
		QueryKey:   identifier,
		SkipEvents: skipEvents,
	}
	events, count, err := c.gitspaceEventStore.List(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list gitspace events for identifier %s: %w", identifier, err)
	}

	var result = make([]*types.GitspaceEventResponse, len(events))
	for index, event := range events {
		gitspaceEventResponse := &types.GitspaceEventResponse{
			GitspaceEvent: *event,
			Message:       eventMessageMap[event.Event],
			EventTime:     time.Unix(0, event.Timestamp).Format(time.RFC3339Nano)}
		result[index] = gitspaceEventResponse
	}

	return result, count, nil
}
