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

	GitspaceEventTypeInfraProvisioningStart,
	GitspaceEventTypeInfraProvisioningCompleted,
	GitspaceEventTypeInfraProvisioningFailed,

	GitspaceEventTypeInfraUnprovisioningStart,
	GitspaceEventTypeInfraUnprovisioningCompleted,
	GitspaceEventTypeInfraUnprovisioningFailed,

	GitspaceEventTypeAgentConnectStart,
	GitspaceEventTypeAgentConnectCompleted,
	GitspaceEventTypeAgentConnectFailed,

	GitspaceEventTypeAgentGitspaceCreationStart,
	GitspaceEventTypeAgentGitspaceCreationCompleted,
	GitspaceEventTypeAgentGitspaceCreationFailed,

	GitspaceEventTypeAgentGitspaceStateReportRunning,
	GitspaceEventTypeAgentGitspaceStateReportError,
	GitspaceEventTypeAgentGitspaceStateReportStopped,
	GitspaceEventTypeAgentGitspaceStateReportUnknown,
}

var eventsMessageMap = eventsMessageMapping()

const (
	// Start action events.
	GitspaceEventTypeGitspaceActionStart          GitspaceEventType = "gitspace_action_start"
	GitspaceEventTypeGitspaceActionStartCompleted GitspaceEventType = "gitspace_action_start_completed"
	GitspaceEventTypeGitspaceActionStartFailed    GitspaceEventType = "gitspace_action_start_failed"

	// Stop action events.
	GitspaceEventTypeGitspaceActionStop          GitspaceEventType = "gitspace_action_stop"
	GitspaceEventTypeGitspaceActionStopCompleted GitspaceEventType = "gitspace_action_stop_completed"
	GitspaceEventTypeGitspaceActionStopFailed    GitspaceEventType = "gitspace_action_stop_failed"

	// Infra provisioning events.
	GitspaceEventTypeInfraProvisioningStart     GitspaceEventType = "infra_provisioning_start"
	GitspaceEventTypeInfraProvisioningCompleted GitspaceEventType = "infra_provisioning_completed"
	GitspaceEventTypeInfraProvisioningFailed    GitspaceEventType = "infra_provisioning_failed"

	// Infra unprovisioning events.
	GitspaceEventTypeInfraUnprovisioningStart     GitspaceEventType = "infra_unprovisioning_start"
	GitspaceEventTypeInfraUnprovisioningCompleted GitspaceEventType = "infra_unprovisioning_completed"
	GitspaceEventTypeInfraUnprovisioningFailed    GitspaceEventType = "infra_unprovisioning_failed"

	// Agent connection events.
	GitspaceEventTypeAgentConnectStart     GitspaceEventType = "agent_connect_start"
	GitspaceEventTypeAgentConnectCompleted GitspaceEventType = "agent_connect_completed"
	GitspaceEventTypeAgentConnectFailed    GitspaceEventType = "agent_connect_failed"

	// Gitspace creation events.
	GitspaceEventTypeAgentGitspaceCreationStart     GitspaceEventType = "agent_gitspace_creation_start"
	GitspaceEventTypeAgentGitspaceCreationCompleted GitspaceEventType = "agent_gitspace_creation_completed"
	GitspaceEventTypeAgentGitspaceCreationFailed    GitspaceEventType = "agent_gitspace_creation_failed"

	// Gitspace state events.
	GitspaceEventTypeAgentGitspaceStateReportRunning GitspaceEventType = "agent_gitspace_state_report_running"
	GitspaceEventTypeAgentGitspaceStateReportError   GitspaceEventType = "agent_gitspace_state_report_error"
	GitspaceEventTypeAgentGitspaceStateReportStopped GitspaceEventType = "agent_gitspace_state_report_stopped"
	GitspaceEventTypeAgentGitspaceStateReportUnknown GitspaceEventType = "agent_gitspace_state_report_unknown"
)

func (e GitspaceEventType) GetValue() string {
	return eventsMessageMap[e]
}

// TODO: Move eventsMessageMapping() to controller.

func eventsMessageMapping() map[GitspaceEventType]string {
	var gitspaceConfigsMap = make(map[GitspaceEventType]string)

	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStart] = "Starting Gitspace..."
	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStartCompleted] = "Started Gitspace"
	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStartFailed] = "Starting Gitspace Failed"

	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStop] = "Stopping Gitspace"
	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStopCompleted] = "Stopped Gitspace"
	gitspaceConfigsMap[GitspaceEventTypeGitspaceActionStopFailed] = "Stopping Gitspace Failed"

	gitspaceConfigsMap[GitspaceEventTypeInfraProvisioningStart] = "Provisioning Infrastructure..."
	gitspaceConfigsMap[GitspaceEventTypeInfraProvisioningCompleted] = "Provisioning Infrastructure Completed"
	gitspaceConfigsMap[GitspaceEventTypeInfraProvisioningFailed] = "Provisioning Infrastructure Failed"

	gitspaceConfigsMap[GitspaceEventTypeInfraUnprovisioningStart] = "Unprovisioning Infrastructure..."
	gitspaceConfigsMap[GitspaceEventTypeInfraUnprovisioningCompleted] = "Unprovisioning Infrastructure Completed"
	gitspaceConfigsMap[GitspaceEventTypeInfraUnprovisioningFailed] = "Unprovisioning Infrastructure Failed"

	gitspaceConfigsMap[GitspaceEventTypeAgentConnectStart] = "Connecting to the gitspace agent..."
	gitspaceConfigsMap[GitspaceEventTypeAgentConnectCompleted] = "Connected to the gitspace agent"
	gitspaceConfigsMap[GitspaceEventTypeAgentConnectFailed] = "Failed connecting to the gitspace agent"

	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceCreationStart] = "Setting up the gitspace..."
	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceCreationCompleted] = "Successfully setup the gitspace"
	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceCreationFailed] = "Failed to setup the gitspace"

	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceStateReportRunning] = "Gitspace is running"
	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceStateReportStopped] = "Gitspace is stopped"
	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceStateReportUnknown] = "Gitspace is in unknown state"
	gitspaceConfigsMap[GitspaceEventTypeAgentGitspaceStateReportError] = "Gitspace has an error"

	return gitspaceConfigsMap
}
