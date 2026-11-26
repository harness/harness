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

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnumMethodsReturnValues(t *testing.T) {
	t.Run("AIAgent", func(t *testing.T) {
		var v AIAgent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("AIAgentAuth", func(t *testing.T) {
		var v AIAgentAuth
		require.NotEmpty(t, v.Enum())
	})
	t.Run("AITaskEvent", func(t *testing.T) {
		var v AITaskEvent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("AITaskState", func(t *testing.T) {
		var v AITaskState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("CIStatus", func(t *testing.T) {
		var v CIStatus
		require.NotEmpty(t, v.Enum())
	})
	t.Run("CheckPayloadKind", func(t *testing.T) {
		var v CheckPayloadKind
		require.NotEmpty(t, v.Enum())
	})
	t.Run("CheckStatus", func(t *testing.T) {
		var v CheckStatus
		require.NotEmpty(t, v.Enum())
	})
	t.Run("CodeOwnerViolationCode", func(t *testing.T) {
		var v CodeOwnerViolationCode
		require.NotEmpty(t, v.Enum())
	})
	t.Run("CommunicationProtocol", func(t *testing.T) {
		var v CommunicationProtocol
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ConnectorAuthType", func(t *testing.T) {
		var v ConnectorAuthType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ConnectorStatus", func(t *testing.T) {
		var v ConnectorStatus
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ConnectorType", func(t *testing.T) {
		var v ConnectorType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ContentEncodingType", func(t *testing.T) {
		var v ContentEncodingType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ExecutionSort", func(t *testing.T) {
		var v ExecutionSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("FeatureOptionValueType", func(t *testing.T) {
		var v FeatureOptionValueType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("FeatureSourceType", func(t *testing.T) {
		var v FeatureSourceType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitOpType", func(t *testing.T) {
		var v GitOpType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitSignatureResult", func(t *testing.T) {
		var v GitSignatureResult
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceAccessType", func(t *testing.T) {
		var v GitspaceAccessType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceActionType", func(t *testing.T) {
		var v GitspaceActionType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceCodeRepoType", func(t *testing.T) {
		var v GitspaceCodeRepoType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceEntityType", func(t *testing.T) {
		var v GitspaceEntityType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceEventType", func(t *testing.T) {
		var v GitspaceEventType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceFilterState", func(t *testing.T) {
		var v GitspaceFilterState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceInstanceStateType", func(t *testing.T) {
		var v GitspaceInstanceStateType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceOperationsEvent", func(t *testing.T) {
		var v GitspaceOperationsEvent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceOwner", func(t *testing.T) {
		var v GitspaceOwner
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceSettingsType", func(t *testing.T) {
		var v GitspaceSettingsType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceSort", func(t *testing.T) {
		var v GitspaceSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("GitspaceStateType", func(t *testing.T) {
		var v GitspaceStateType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("IDEType", func(t *testing.T) {
		var v IDEType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("InfraEvent", func(t *testing.T) {
		var v InfraEvent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("InfraProviderType", func(t *testing.T) {
		var v InfraProviderType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("InfraProvisioningType", func(t *testing.T) {
		var v InfraProvisioningType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("InfraStatus", func(t *testing.T) {
		var v InfraStatus
		require.NotEmpty(t, v.Enum())
	})
	t.Run("JobState", func(t *testing.T) {
		var v JobState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("LabelColor", func(t *testing.T) {
		var v LabelColor
		require.NotEmpty(t, v.Enum())
	})
	t.Run("LabelType", func(t *testing.T) {
		var v LabelType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MembershipRole", func(t *testing.T) {
		var v MembershipRole
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MembershipSpaceSort", func(t *testing.T) {
		var v MembershipSpaceSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MembershipUserSort", func(t *testing.T) {
		var v MembershipUserSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MergeMethod", func(t *testing.T) {
		var v MergeMethod
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MergeQueueEntryState", func(t *testing.T) {
		var v MergeQueueEntryState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("MergeQueueRemovalReason", func(t *testing.T) {
		var v MergeQueueRemovalReason
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ParentResourceType", func(t *testing.T) {
		var v ParentResourceType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PrincipalType", func(t *testing.T) {
		var v PrincipalType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PublicKeyScheme", func(t *testing.T) {
		var v PublicKeyScheme
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PublicKeySort", func(t *testing.T) {
		var v PublicKeySort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PublicKeyUsage", func(t *testing.T) {
		var v PublicKeyUsage
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PublicResourceType", func(t *testing.T) {
		var v PublicResourceType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqActivityKind", func(t *testing.T) {
		var v PullReqActivityKind
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqActivityType", func(t *testing.T) {
		var v PullReqActivityType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqCommentReactionEmoji", func(t *testing.T) {
		var v PullReqCommentReactionEmoji
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqCommentStatus", func(t *testing.T) {
		var v PullReqCommentStatus
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqLabelActivityType", func(t *testing.T) {
		var v PullReqLabelActivityType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqReviewDecision", func(t *testing.T) {
		var v PullReqReviewDecision
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqReviewerType", func(t *testing.T) {
		var v PullReqReviewerType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqSort", func(t *testing.T) {
		var v PullReqSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqState", func(t *testing.T) {
		var v PullReqState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("PullReqSubState", func(t *testing.T) {
		var v PullReqSubState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RepoActivityType", func(t *testing.T) {
		var v RepoActivityType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RepoType", func(t *testing.T) {
		var v RepoType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ResourceType", func(t *testing.T) {
		var v ResourceType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RevocationReason", func(t *testing.T) {
		var v RevocationReason
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RuleParent", func(t *testing.T) {
		var v RuleParent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RuleSort", func(t *testing.T) {
		var v RuleSort
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RuleState", func(t *testing.T) {
		var v RuleState
		require.NotEmpty(t, v.Enum())
	})
	t.Run("RuleType", func(t *testing.T) {
		var v RuleType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("ScmType", func(t *testing.T) {
		var v ScmType
		require.NotEmpty(t, v.Enum())
	})
	t.Run("SettingsScope", func(t *testing.T) {
		var v SettingsScope
		require.NotEmpty(t, v.Enum())
	})
	t.Run("TriggerAction", func(t *testing.T) {
		var v TriggerAction
		require.NotEmpty(t, v.Enum())
	})
	t.Run("TriggerEvent", func(t *testing.T) {
		var v TriggerEvent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("WebhookExecutionResult", func(t *testing.T) {
		var v WebhookExecutionResult
		require.NotEmpty(t, v.Enum())
	})
	t.Run("WebhookParent", func(t *testing.T) {
		var v WebhookParent
		require.NotEmpty(t, v.Enum())
	})
	t.Run("WebhookTrigger", func(t *testing.T) {
		var v WebhookTrigger
		require.NotEmpty(t, v.Enum())
	})
	t.Run("WebhookType", func(t *testing.T) {
		var v WebhookType
		require.NotEmpty(t, v.Enum())
	})
}
