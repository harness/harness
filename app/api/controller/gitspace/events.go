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
	eventMessageMap = eventsMessageMapping()
}

func (c *Controller) Events(
	ctx context.Context,
	session *auth.Session,
	spaceRef string,
	identifier string,
	page int,
	limit int,
) ([]*types.GitspaceEventResponse, int, error) {
	space, err := c.spaceStore.FindByRef(ctx, spaceRef)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find space: %w", err)
	}

	err = apiauth.CheckGitspace(ctx, c.authorizer, session, space.Path, identifier, enum.PermissionGitspaceView)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to authorize: %w", err)
	}

	filter := &types.GitspaceEventFilter{}
	filter.QueryKey = identifier
	filter.Page = page
	filter.Size = limit
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

func eventsMessageMapping() map[enum.GitspaceEventType]string {
	var gitspaceConfigsMap = make(map[enum.GitspaceEventType]string)

	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStart] = "Starting Gitspace..."
	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStartCompleted] = "Started Gitspace"
	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStartFailed] = "Starting Gitspace Failed"

	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStop] = "Stopping Gitspace"
	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStopCompleted] = "Stopped Gitspace"
	gitspaceConfigsMap[enum.GitspaceEventTypeGitspaceActionStopFailed] = "Stopping Gitspace Failed"

	gitspaceConfigsMap[enum.GitspaceEventTypeFetchDevcontainerStart] = "Fetching devcontainer config..."
	gitspaceConfigsMap[enum.GitspaceEventTypeFetchDevcontainerCompleted] = "Fetched devcontainer config"
	gitspaceConfigsMap[enum.GitspaceEventTypeFetchDevcontainerFailed] = "Fetching devcontainer config failed"

	gitspaceConfigsMap[enum.GitspaceEventTypeInfraProvisioningStart] = "Provisioning Infrastructure..."
	gitspaceConfigsMap[enum.GitspaceEventTypeInfraProvisioningCompleted] = "Provisioning Infrastructure Completed"
	gitspaceConfigsMap[enum.GitspaceEventTypeInfraProvisioningFailed] = "Provisioning Infrastructure Failed"

	gitspaceConfigsMap[enum.GitspaceEventTypeInfraUnprovisioningStart] = "Unprovisioning Infrastructure..."
	gitspaceConfigsMap[enum.GitspaceEventTypeInfraUnprovisioningCompleted] = "Unprovisioning Infrastructure Completed"
	gitspaceConfigsMap[enum.GitspaceEventTypeInfraUnprovisioningFailed] = "Unprovisioning Infrastructure Failed"

	gitspaceConfigsMap[enum.GitspaceEventTypeAgentConnectStart] = "Connecting to the gitspace agent..."
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentConnectCompleted] = "Connected to the gitspace agent"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentConnectFailed] = "Failed connecting to the gitspace agent"

	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceCreationStart] = "Setting up the gitspace..."
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceCreationCompleted] = "Successfully setup the gitspace"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceCreationFailed] = "Failed to setup the gitspace"

	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceDeletionStart] = "Removing the gitspace..."
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceDeletionCompleted] = "Successfully removed the gitspace"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceDeletionFailed] = "Failed to remove the gitspace"

	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceStateReportRunning] = "Gitspace is running"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceStateReportStopped] = "Gitspace is stopped"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceStateReportUnknown] = "Gitspace is in unknown state"
	gitspaceConfigsMap[enum.GitspaceEventTypeAgentGitspaceStateReportError] = "Gitspace has an error"
	return gitspaceConfigsMap
}
