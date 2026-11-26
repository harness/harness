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
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

type stringEnum[T ~string] interface {
	~string
	Sanitize() (T, bool)
	Enum() []any
}

func checkStringEnum[T stringEnum[T]](t *testing.T, all func() ([]T, T)) {
	t.Helper()

	values, def := all()
	require.NotEmpty(t, values)

	asStrings := make([]string, len(values))
	for i, v := range values {
		asStrings[i] = string(v)
	}
	require.True(t, sort.StringsAreSorted(asStrings), "values must be sorted for binary search: %v", asStrings)

	var zero T
	require.Len(t, zero.Enum(), len(values))

	for _, v := range values {
		got, ok := v.Sanitize()
		require.True(t, ok, "value %q must be valid", v)
		require.Equal(t, v, got)
	}

	// an unknown value is rejected and replaced by the default
	unknown := T("definitely-not-a-valid-enum-value")
	got, ok := unknown.Sanitize()
	require.False(t, ok)
	require.Equal(t, def, got)

	// the empty value is replaced by the default when one exists
	got, ok = zero.Sanitize()
	if def != zero {
		require.True(t, ok)
		require.Equal(t, def, got)
	}
}

func TestStringEnums(t *testing.T) {
	t.Run("AIAgent", func(t *testing.T) { checkStringEnum(t, GetAllAIAgents) })
	t.Run("AITaskState", func(t *testing.T) { checkStringEnum(t, GetAllAITaskState) })
	t.Run("AutoLinkType", func(t *testing.T) { checkStringEnum(t, GetAllAutoLinkTypes) })
	t.Run("CheckStatus", func(t *testing.T) { checkStringEnum(t, GetAllCheckStatuses) })
	t.Run("CheckPayloadKind", func(t *testing.T) { checkStringEnum(t, GetAllCheckPayloadTypes) })
	t.Run("CIStatus", func(t *testing.T) { checkStringEnum(t, GetAllCIStatuses) })
	t.Run("ConnectorStatus", func(t *testing.T) { checkStringEnum(t, GetAllConnectorStatus) })
	t.Run("ConnectorType", func(t *testing.T) { checkStringEnum(t, GetAllConnectorTypes) })
	t.Run("ExecutionSort", func(t *testing.T) { checkStringEnum(t, GetAllExecutionSorts) })
	t.Run("GitOpType", func(t *testing.T) { checkStringEnum(t, GetAllGitOpTypes) })
	t.Run("GitspaceFilterState", func(t *testing.T) { checkStringEnum(t, GetAllGitspaceFilterState) })
	t.Run("JobState", func(t *testing.T) { checkStringEnum(t, GetAllJobStates) })
	t.Run("LabelType", func(t *testing.T) { checkStringEnum(t, GetAllLabelTypes) })
	t.Run("LabelColor", func(t *testing.T) { checkStringEnum(t, GetAllLabelColors) })
	t.Run("MembershipUserSort", func(t *testing.T) { checkStringEnum(t, GetAllMembershipUserSorts) })
	t.Run("MembershipSpaceSort", func(t *testing.T) { checkStringEnum(t, GetAllMembershipSpaceSorts) })
	t.Run("MembershipRole", func(t *testing.T) { checkStringEnum(t, GetAllMembershipRoles) })
	t.Run("MergeQueueEntryState", func(t *testing.T) { checkStringEnum(t, GetAllMergeQueueEntryStates) })
	t.Run("MergeQueueRemovalReason", func(t *testing.T) { checkStringEnum(t, GetAllMergeQueueRemovalReasons) })
	t.Run("PrincipalType", func(t *testing.T) { checkStringEnum(t, GetAllPrincipalTypes) })
	t.Run("PublicKeyScheme", func(t *testing.T) { checkStringEnum(t, GetAllPublicKeySchemes) })
	t.Run("PublicKeyUsage", func(t *testing.T) { checkStringEnum(t, GetAllPublicKeyUsages) })
	t.Run("PublicKeySort", func(t *testing.T) { checkStringEnum(t, GetAllPublicKeySorts) })
	t.Run("RevocationReason", func(t *testing.T) { checkStringEnum(t, GetAllRevocationReasons) })
	t.Run("PullReqState", func(t *testing.T) { checkStringEnum(t, GetAllPullReqStates) })
	t.Run("PullReqSubState", func(t *testing.T) { checkStringEnum(t, GetAllPullReqSubStates) })
	t.Run("PullReqSort", func(t *testing.T) { checkStringEnum(t, GetAllPullReqSorts) })
	t.Run("PullReqActivityType", func(t *testing.T) { checkStringEnum(t, GetAllPullReqActivityTypes) })
	t.Run("PullReqActivityKind", func(t *testing.T) { checkStringEnum(t, GetAllPullReqActivityKinds) })
	t.Run("PullReqCommentStatus", func(t *testing.T) { checkStringEnum(t, GetAllPullReqCommentStatuses) })
	t.Run("PullReqReviewDecision", func(t *testing.T) { checkStringEnum(t, GetAllPullReqReviewDecisions) })
	t.Run("PullReqReviewerType", func(t *testing.T) { checkStringEnum(t, GetAllPullReqReviewerTypes) })
	t.Run("PullReqLabelActivityType", func(t *testing.T) { checkStringEnum(t, GetAllLabelActivityTypes) })
	t.Run("PullReqCommentReactionEmoji", func(t *testing.T) { checkStringEnum(t, GetAllPullReqCommentReactionEmojis) })
	t.Run("RepoType", func(t *testing.T) { checkStringEnum(t, GetAllRepoTypes) })
	t.Run("RepoActivityType", func(t *testing.T) { checkStringEnum(t, GetAllRepoActivityTypes) })
	t.Run("ResourceType", func(t *testing.T) { checkStringEnum(t, GetAllResourceTypes) })
	t.Run("RuleType", func(t *testing.T) { checkStringEnum(t, GetAllRuleTypes) })
	t.Run("RuleState", func(t *testing.T) { checkStringEnum(t, GetAllRuleStates) })
	t.Run("RuleSort", func(t *testing.T) { checkStringEnum(t, GetAllRuleSorts) })
	t.Run("TriggerAction", func(t *testing.T) { checkStringEnum(t, GetAllTriggerActions) })
	t.Run("TriggerEvent", func(t *testing.T) { checkStringEnum(t, GetAllTriggerEvents) })
	t.Run("WebhookTrigger", func(t *testing.T) { checkStringEnum(t, GetAllWebhookTriggers) })
}
