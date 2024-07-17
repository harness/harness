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

	// Fetch devcontainer config events.
	GitspaceEventTypeFetchDevcontainerStart     GitspaceEventType = "fetch_devcontainer_start"
	GitspaceEventTypeFetchDevcontainerCompleted GitspaceEventType = "fetch_devcontainer_completed"
	GitspaceEventTypeFetchDevcontainerFailed    GitspaceEventType = "fetch_devcontainer_failed"

	// Infra provisioning events.
	GitspaceEventTypeInfraProvisioningStart     GitspaceEventType = "infra_provisioning_start"
	GitspaceEventTypeInfraProvisioningCompleted GitspaceEventType = "infra_provisioning_completed"
	GitspaceEventTypeInfraProvisioningFailed    GitspaceEventType = "infra_provisioning_failed"

	// Infra stop events.
	GitspaceEventTypeInfraStopStart     GitspaceEventType = "infra_stop_start"
	GitspaceEventTypeInfraStopCompleted GitspaceEventType = "infra_stop_completed"
	GitspaceEventTypeInfraStopFailed    GitspaceEventType = "infra_stop_failed"

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
)
