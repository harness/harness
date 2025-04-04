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

func eventsMessageMapping() map[enum.GitspaceEventType]string {
	var gitspaceConfigsMap = map[enum.GitspaceEventType]string{
		enum.GitspaceEventTypeGitspaceActionStart:          "Starting gitspace...",
		enum.GitspaceEventTypeGitspaceActionStartCompleted: "Started gitspace",
		enum.GitspaceEventTypeGitspaceActionStartFailed:    "Starting gitspace failed",

		enum.GitspaceEventTypeGitspaceActionStop:          "Stopping gitspace...",
		enum.GitspaceEventTypeGitspaceActionStopCompleted: "Stopped gitspace",
		enum.GitspaceEventTypeGitspaceActionStopFailed:    "Stopping gitspace failed",

		enum.GitspaceEventTypeFetchDevcontainerStart:     "Fetching devcontainer config...",
		enum.GitspaceEventTypeFetchDevcontainerCompleted: "Fetched devcontainer config",
		enum.GitspaceEventTypeFetchDevcontainerFailed:    "Fetching devcontainer config failed",

		enum.GitspaceEventTypeFetchConnectorsDetailsStart:     "Fetching platform connectors details...",
		enum.GitspaceEventTypeFetchConnectorsDetailsCompleted: "Fetched platform connectors details",
		enum.GitspaceEventTypeFetchConnectorsDetailsFailed:    "Fetching platform connectors details failed",

		enum.GitspaceEventTypeInfraProvisioningStart:     "Provisioning infrastructure...",
		enum.GitspaceEventTypeInfraProvisioningCompleted: "Provisioning infrastructure completed",
		enum.GitspaceEventTypeInfraProvisioningFailed:    "Provisioning infrastructure failed",

		enum.GitspaceEventTypeInfraGatewayRouteStart:     "Updating gateway routing...",
		enum.GitspaceEventTypeInfraGatewayRouteCompleted: "Updating gateway routing completed",
		enum.GitspaceEventTypeInfraGatewayRouteFailed:    "Updating gateway routing failed",

		enum.GitspaceEventTypeInfraStopStart:     "Stopping infrastructure...",
		enum.GitspaceEventTypeInfraStopCompleted: "Stopping infrastructure completed",
		enum.GitspaceEventTypeInfraStopFailed:    "Stopping infrastructure failed",

		enum.GitspaceEventTypeInfraDeprovisioningStart:     "Deprovisioning infrastructure...",
		enum.GitspaceEventTypeInfraDeprovisioningCompleted: "Deprovisioning infrastructure completed",
		enum.GitspaceEventTypeInfraDeprovisioningFailed:    "Deprovisioning infrastructure failed",

		enum.GitspaceEventTypeAgentConnectStart:     "Connecting to the gitspace agent...",
		enum.GitspaceEventTypeAgentConnectCompleted: "Connected to the gitspace agent",
		enum.GitspaceEventTypeAgentConnectFailed:    "Failed connecting to the gitspace agent",

		enum.GitspaceEventTypeAgentGitspaceCreationStart:     "Setting up the gitspace...",
		enum.GitspaceEventTypeAgentGitspaceCreationCompleted: "Successfully setup the gitspace",
		enum.GitspaceEventTypeAgentGitspaceCreationFailed:    "Failed to setup the gitspace",

		enum.GitspaceEventTypeAgentGitspaceStopStart:     "Stopping the gitspace...",
		enum.GitspaceEventTypeAgentGitspaceStopCompleted: "Successfully stopped the gitspace",
		enum.GitspaceEventTypeAgentGitspaceStopFailed:    "Failed to stop the gitspace",

		enum.GitspaceEventTypeAgentGitspaceDeletionStart:      "Removing the gitspace...",
		enum.GitspaceEventTypeAgentGitspaceDeletionCompleted:  "Successfully removed the gitspace",
		enum.GitspaceEventTypeAgentGitspaceDeletionFailed:     "Failed to remove the gitspace",
		enum.GitspaceEventTypeAgentGitspaceStateReportRunning: "Gitspace is running",
		enum.GitspaceEventTypeAgentGitspaceStateReportStopped: "Gitspace is stopped",
		enum.GitspaceEventTypeAgentGitspaceStateReportUnknown: "Gitspace is in unknown state",
		enum.GitspaceEventTypeAgentGitspaceStateReportError:   "Gitspace has an error",

		enum.GitspaceEventTypeGitspaceAutoStop: "Triggering auto-stopping due to inactivity...",

		enum.GitspaceEventTypeInfraCleanupStart:     "Cleaning up infrastructure...",
		enum.GitspaceEventTypeInfraCleanupCompleted: "Successfully cleaned up infrastructure",
		enum.GitspaceEventTypeInfraCleanupFailed:    "Failed to cleaned up infrastructure",

		enum.GitspaceEventTypeInfraResetStart:  "Resetting the gitspace infrastructure...",
		enum.GitspaceEventTypeInfraResetFailed: "Failed to reset the gitspace infrastructure",

		enum.GitspaceEventTypeDelegateTaskSubmitted: "Delegate task submitted",

		enum.GitspaceEventTypeInfraVMCreationStart:     "creating VM...",
		enum.GitspaceEventTypeInfraVMCreationCompleted: "Successfully created VM",
		enum.GitspaceEventTypeInfraVMCreationFailed:    "Failed to created VM",

		enum.GitspaceEventTypeInfraPublishGatewayCompleted: "Published machine port mapping to Gateway",
		enum.GitspaceEventTypeInfraPublishGatewayFailed:    "Failed to publish machine port mapping to Gateway",
	}

	return gitspaceConfigsMap
}
