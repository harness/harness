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

package enum

type GitspaceEventType string

func (GitspaceEventType) Enum() []interface{} {
	return toInterfaceSlice(gitspaceEventTypes)
}

var gitspaceEventTypes = []GitspaceEventType{
	GitspaceEventTypeGitspaceActionStart,
	GitspaceEventTypeGitspaceActionStartCompleted,
	GitspaceEventTypeGitspaceActionStartFailed,

	GitspaceEventTypeGitspaceActionStop,
	GitspaceEventTypeGitspaceActionStopCompleted,
	GitspaceEventTypeGitspaceActionStopFailed,

	GitspaceEventTypeFetchDevcontainerStart,
	GitspaceEventTypeFetchDevcontainerCompleted,
	GitspaceEventTypeFetchDevcontainerFailed,

	GitspaceEventTypeInfraProvisioningStart,
	GitspaceEventTypeInfraProvisioningCompleted,
	GitspaceEventTypeInfraProvisioningFailed,

	GitspaceEventTypeInfraStopStart,
	GitspaceEventTypeInfraStopCompleted,
	GitspaceEventTypeInfraStopFailed,

	GitspaceEventTypeInfraDeprovisioningStart,
	GitspaceEventTypeInfraDeprovisioningCompleted,
	GitspaceEventTypeInfraDeprovisioningFailed,

	GitspaceEventTypeAgentConnectStart,
	GitspaceEventTypeAgentConnectCompleted,
	GitspaceEventTypeAgentConnectFailed,

	GitspaceEventTypeAgentGitspaceCreationStart,
	GitspaceEventTypeAgentGitspaceCreationCompleted,
	GitspaceEventTypeAgentGitspaceCreationFailed,

	GitspaceEventTypeAgentGitspaceStopStart,
	GitspaceEventTypeAgentGitspaceStopCompleted,
	GitspaceEventTypeAgentGitspaceStopFailed,

	GitspaceEventTypeAgentGitspaceDeletionStart,
	GitspaceEventTypeAgentGitspaceDeletionCompleted,
	GitspaceEventTypeAgentGitspaceDeletionFailed,

	GitspaceEventTypeAgentGitspaceStateReportRunning,
	GitspaceEventTypeAgentGitspaceStateReportError,
	GitspaceEventTypeAgentGitspaceStateReportStopped,
	GitspaceEventTypeAgentGitspaceStateReportUnknown,

	GitspaceEventTypeGitspaceAutoStop,

	GitspaceEventTypeGitspaceActionReset,
	GitspaceEventTypeGitspaceActionResetCompleted,
	GitspaceEventTypeGitspaceActionResetFailed,
}

const (
	// Start action events.
	GitspaceEventTypeGitspaceActionStart          GitspaceEventType = "gitspace_action_start"
	GitspaceEventTypeGitspaceActionStartCompleted GitspaceEventType = "gitspace_action_start_completed"
	GitspaceEventTypeGitspaceActionStartFailed    GitspaceEventType = "gitspace_action_start_failed"

	// Stop action events.
	GitspaceEventTypeGitspaceActionStop          GitspaceEventType = "gitspace_action_stop"
	GitspaceEventTypeGitspaceActionStopCompleted GitspaceEventType = "gitspace_action_stop_completed"
	GitspaceEventTypeGitspaceActionStopFailed    GitspaceEventType = "gitspace_action_stop_failed"

	// Reset action events.
	GitspaceEventTypeGitspaceActionReset                            = "gitspace_action_reset"
	GitspaceEventTypeGitspaceActionResetCompleted GitspaceEventType = "gitspace_action_reset_completed"
	GitspaceEventTypeGitspaceActionResetFailed    GitspaceEventType = "gitspace_action_reset_failed"

	// Fetch devcontainer config events.
	GitspaceEventTypeFetchDevcontainerStart     GitspaceEventType = "fetch_devcontainer_start"
	GitspaceEventTypeFetchDevcontainerCompleted GitspaceEventType = "fetch_devcontainer_completed"
	GitspaceEventTypeFetchDevcontainerFailed    GitspaceEventType = "fetch_devcontainer_failed"

	// Fetch artifact registry secret.
	GitspaceEventTypeFetchConnectorsDetailsStart     GitspaceEventType = "fetch_connectors_details_start"
	GitspaceEventTypeFetchConnectorsDetailsCompleted GitspaceEventType = "fetch_connectors_details_completed" //nolint
	GitspaceEventTypeFetchConnectorsDetailsFailed    GitspaceEventType = "fetch_connectors_details_failed"

	// Infra provisioning events.
	GitspaceEventTypeInfraProvisioningStart     GitspaceEventType = "infra_provisioning_start"
	GitspaceEventTypeInfraProvisioningCompleted GitspaceEventType = "infra_provisioning_completed"
	GitspaceEventTypeInfraProvisioningFailed    GitspaceEventType = "infra_provisioning_failed"

	// Gateway update events.
	GitspaceEventTypeInfraGatewayRouteStart     GitspaceEventType = "infra_gateway_route_start"
	GitspaceEventTypeInfraGatewayRouteCompleted GitspaceEventType = "infra_gateway_route_completed"
	GitspaceEventTypeInfraGatewayRouteFailed    GitspaceEventType = "infra_gateway_route_failed"

	// Infra stop events.
	GitspaceEventTypeInfraStopStart     GitspaceEventType = "infra_stop_start"
	GitspaceEventTypeInfraStopCompleted GitspaceEventType = "infra_stop_completed"
	GitspaceEventTypeInfraStopFailed    GitspaceEventType = "infra_stop_failed"

	// Infra cleanup events.
	GitspaceEventTypeInfraCleanupStart     GitspaceEventType = "infra_cleanup_start"
	GitspaceEventTypeInfraCleanupCompleted GitspaceEventType = "infra_cleanup_completed"
	GitspaceEventTypeInfraCleanupFailed    GitspaceEventType = "infra_cleanup_failed"

	// Infra deprovisioning events.
	GitspaceEventTypeInfraDeprovisioningStart     GitspaceEventType = "infra_deprovisioning_start"
	GitspaceEventTypeInfraDeprovisioningCompleted GitspaceEventType = "infra_deprovisioning_completed"
	GitspaceEventTypeInfraDeprovisioningFailed    GitspaceEventType = "infra_deprovisioning_failed"

	// Agent connection events.
	GitspaceEventTypeAgentConnectStart     GitspaceEventType = "agent_connect_start"
	GitspaceEventTypeAgentConnectCompleted GitspaceEventType = "agent_connect_completed"
	GitspaceEventTypeAgentConnectFailed    GitspaceEventType = "agent_connect_failed"

	// Gitspace creation events.
	GitspaceEventTypeAgentGitspaceCreationStart     GitspaceEventType = "agent_gitspace_creation_start"
	GitspaceEventTypeAgentGitspaceCreationCompleted GitspaceEventType = "agent_gitspace_creation_completed"
	GitspaceEventTypeAgentGitspaceCreationFailed    GitspaceEventType = "agent_gitspace_creation_failed"

	// Gitspace stop events.
	GitspaceEventTypeAgentGitspaceStopStart     GitspaceEventType = "agent_gitspace_stop_start"
	GitspaceEventTypeAgentGitspaceStopCompleted GitspaceEventType = "agent_gitspace_stop_completed"
	GitspaceEventTypeAgentGitspaceStopFailed    GitspaceEventType = "agent_gitspace_stop_failed"

	// Gitspace deletion events.
	GitspaceEventTypeAgentGitspaceDeletionStart     GitspaceEventType = "agent_gitspace_deletion_start"
	GitspaceEventTypeAgentGitspaceDeletionCompleted GitspaceEventType = "agent_gitspace_deletion_completed"
	GitspaceEventTypeAgentGitspaceDeletionFailed    GitspaceEventType = "agent_gitspace_deletion_failed"

	// Gitspace state events.
	GitspaceEventTypeAgentGitspaceStateReportRunning GitspaceEventType = "agent_gitspace_state_report_running"
	GitspaceEventTypeAgentGitspaceStateReportError   GitspaceEventType = "agent_gitspace_state_report_error"
	GitspaceEventTypeAgentGitspaceStateReportStopped GitspaceEventType = "agent_gitspace_state_report_stopped"
	GitspaceEventTypeAgentGitspaceStateReportUnknown GitspaceEventType = "agent_gitspace_state_report_unknown"

	// AutoStop action event.
	GitspaceEventTypeGitspaceAutoStop GitspaceEventType = "gitspace_action_auto_stop"

	// Cleanup job events.
	GitspaceEventTypeGitspaceCleanupJob GitspaceEventType = "gitspace_action_cleanup_job"

	// Infra reset events.
	GitspaceEventTypeInfraResetStart  GitspaceEventType = "infra_reset_start"
	GitspaceEventTypeInfraResetFailed GitspaceEventType = "infra_reset_failed"

	GitspaceEventTypeDelegateTaskSubmitted GitspaceEventType = "delegate_task_submitted"

	GitspaceEventTypeInfraVMCreationStart     GitspaceEventType = "infra_vm_creation_start"
	GitspaceEventTypeInfraVMCreationCompleted GitspaceEventType = "infra_vm_creation_completed"
	GitspaceEventTypeInfraVMCreationFailed    GitspaceEventType = "infra_vm_creation_failed"

	GitspaceEventTypeInfraPublishGatewayCompleted GitspaceEventType = "infra_publish_gateway_completed"
	GitspaceEventTypeInfraPublishGatewayFailed    GitspaceEventType = "infra_publish_gateway_failed"
)

func EventsMessageMapping() map[GitspaceEventType]string {
	var gitspaceConfigsMap = map[GitspaceEventType]string{
		GitspaceEventTypeGitspaceActionStart:          "Starting gitspace...",
		GitspaceEventTypeGitspaceActionStartCompleted: "Started gitspace",
		GitspaceEventTypeGitspaceActionStartFailed:    "Starting gitspace failed",

		GitspaceEventTypeGitspaceActionStop:          "Stopping gitspace...",
		GitspaceEventTypeGitspaceActionStopCompleted: "Stopped gitspace",
		GitspaceEventTypeGitspaceActionStopFailed:    "Stopping gitspace failed",

		GitspaceEventTypeGitspaceActionReset:          "Resetting gitspace...",
		GitspaceEventTypeGitspaceActionResetCompleted: "Resetted gitspace",
		GitspaceEventTypeGitspaceActionResetFailed:    "Failed to reset gitspace",

		GitspaceEventTypeFetchDevcontainerStart:     "Fetching devcontainer config...",
		GitspaceEventTypeFetchDevcontainerCompleted: "Fetched devcontainer config",
		GitspaceEventTypeFetchDevcontainerFailed:    "Fetching devcontainer config failed",

		GitspaceEventTypeFetchConnectorsDetailsStart:     "Fetching platform connectors details...",
		GitspaceEventTypeFetchConnectorsDetailsCompleted: "Fetched platform connectors details",
		GitspaceEventTypeFetchConnectorsDetailsFailed:    "Fetching platform connectors details failed",

		GitspaceEventTypeInfraProvisioningStart:     "Provisioning infrastructure...",
		GitspaceEventTypeInfraProvisioningCompleted: "Provisioning infrastructure completed",
		GitspaceEventTypeInfraProvisioningFailed:    "Provisioning infrastructure failed",

		GitspaceEventTypeInfraGatewayRouteStart:     "Updating gateway routing...",
		GitspaceEventTypeInfraGatewayRouteCompleted: "Updating gateway routing completed",
		GitspaceEventTypeInfraGatewayRouteFailed:    "Updating gateway routing failed",

		GitspaceEventTypeInfraStopStart:     "Stopping infrastructure...",
		GitspaceEventTypeInfraStopCompleted: "Stopping infrastructure completed",
		GitspaceEventTypeInfraStopFailed:    "Stopping infrastructure failed",

		GitspaceEventTypeInfraDeprovisioningStart:     "Deprovisioning infrastructure...",
		GitspaceEventTypeInfraDeprovisioningCompleted: "Deprovisioning infrastructure completed",
		GitspaceEventTypeInfraDeprovisioningFailed:    "Deprovisioning infrastructure failed",

		GitspaceEventTypeAgentConnectStart:     "Connecting to the gitspace agent...",
		GitspaceEventTypeAgentConnectCompleted: "Connected to the gitspace agent",
		GitspaceEventTypeAgentConnectFailed:    "Failed connecting to the gitspace agent",

		GitspaceEventTypeAgentGitspaceCreationStart:     "Setting up the gitspace...",
		GitspaceEventTypeAgentGitspaceCreationCompleted: "Successfully setup the gitspace",
		GitspaceEventTypeAgentGitspaceCreationFailed:    "Failed to setup the gitspace",

		GitspaceEventTypeAgentGitspaceStopStart:     "Stopping the gitspace...",
		GitspaceEventTypeAgentGitspaceStopCompleted: "Successfully stopped the gitspace",
		GitspaceEventTypeAgentGitspaceStopFailed:    "Failed to stop the gitspace",

		GitspaceEventTypeAgentGitspaceDeletionStart:      "Removing the gitspace...",
		GitspaceEventTypeAgentGitspaceDeletionCompleted:  "Successfully removed the gitspace",
		GitspaceEventTypeAgentGitspaceDeletionFailed:     "Failed to remove the gitspace",
		GitspaceEventTypeAgentGitspaceStateReportRunning: "Gitspace is running",
		GitspaceEventTypeAgentGitspaceStateReportStopped: "Gitspace is stopped",
		GitspaceEventTypeAgentGitspaceStateReportUnknown: "Gitspace is in unknown state",
		GitspaceEventTypeAgentGitspaceStateReportError:   "Gitspace has an error",

		GitspaceEventTypeGitspaceAutoStop: "Triggering auto-stopping due to inactivity...",

		GitspaceEventTypeGitspaceCleanupJob: "Triggering cleanup job...",

		GitspaceEventTypeInfraCleanupStart:     "Cleaning up infrastructure...",
		GitspaceEventTypeInfraCleanupCompleted: "Successfully cleaned up infrastructure",
		GitspaceEventTypeInfraCleanupFailed:    "Failed to cleaned up infrastructure",

		GitspaceEventTypeInfraResetStart:  "Resetting the gitspace infrastructure...",
		GitspaceEventTypeInfraResetFailed: "Failed to reset the gitspace infrastructure",

		GitspaceEventTypeDelegateTaskSubmitted: "Delegate task submitted",

		GitspaceEventTypeInfraVMCreationStart:     "Creating Virtual Machine...",
		GitspaceEventTypeInfraVMCreationCompleted: "Successfully created Virtual Machine",
		GitspaceEventTypeInfraVMCreationFailed:    "Failed to created Virtual Machine",

		GitspaceEventTypeInfraPublishGatewayCompleted: "Published machine port mapping to Gateway",
		GitspaceEventTypeInfraPublishGatewayFailed:    "Failed to publish machine port mapping to Gateway",
	}

	return gitspaceConfigsMap
}
