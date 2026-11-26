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
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAIAgent(t *testing.T) {
	require.Equal(t, "claude-code", AIAgentClaudeCode.String())

	var a AIAgent
	require.NoError(t, json.Unmarshal([]byte(`"claude-code"`), &a))
	require.Equal(t, AIAgentClaudeCode, a)
	require.NoError(t, json.Unmarshal([]byte(`""`), &a))
	require.Empty(t, a)
	require.Error(t, json.Unmarshal([]byte(`"gpt"`), &a))
	require.Error(t, json.Unmarshal([]byte(`123`), &a))
}

func TestAIAgentAuth(t *testing.T) {
	require.Equal(t, "anthropic-api-key", AnthropicAPIKeyAuth.String())
	require.Len(t, AnthropicAPIKeyAuth.Enum(), len(AIAgentAuthTypes))

	var a AIAgentAuth
	require.NoError(t, json.Unmarshal([]byte(`"anthropic-api-key"`), &a))
	require.Equal(t, AnthropicAPIKeyAuth, a)
	require.NoError(t, json.Unmarshal([]byte(`""`), &a))
	require.Empty(t, a)
	require.Error(t, json.Unmarshal([]byte(`"basic"`), &a))
	require.Error(t, json.Unmarshal([]byte(`123`), &a))
}

func TestAITaskState(t *testing.T) {
	for state, want := range map[AITaskState][2]bool{
		AITaskStateUninitialized: {false, false},
		AITaskStateRunning:       {false, true},
		AITaskStateCompleted:     {true, false},
		AITaskStateError:         {true, false},
	} {
		require.Equal(t, want[0], state.IsFinalStatus(), state)
		require.Equal(t, want[1], state.IsActiveStatus(), state)
	}
}

func TestCheckStatus(t *testing.T) {
	completed := map[CheckStatus]bool{
		CheckStatusPending: false, CheckStatusRunning: false, CheckStatusSuccess: true,
		CheckStatusFailure: true, CheckStatusError: true, CheckStatusFailureIgnored: true,
	}
	success := map[CheckStatus]bool{CheckStatusSuccess: true, CheckStatusFailureIgnored: true}
	for status, want := range completed {
		require.Equal(t, want, status.IsCompleted(), status)
		require.Equal(t, success[status], status.IsSuccess(), status)
	}
}

func TestCIStatus(t *testing.T) {
	converted := map[CIStatus]CheckStatus{
		CIStatusPending:       CheckStatusPending,
		CIStatusWaitingOnDeps: CheckStatusPending,
		CIStatusSuccess:       CheckStatusSuccess,
		CIStatusSkipped:       CheckStatusSuccess,
		CIStatusFailure:       CheckStatusFailure,
		CIStatusRunning:       CheckStatusRunning,
		CIStatusKilled:        CheckStatusError,
		CIStatusError:         CheckStatusError,
		CIStatusBlocked:       CheckStatusError,
		CIStatusDeclined:      CheckStatusError,
	}
	for status, want := range converted {
		require.Equal(t, want, status.ConvertToCheckStatus(), status)
	}

	done := map[CIStatus]bool{
		CIStatusSkipped: true, CIStatusDeclined: true, CIStatusSuccess: true, CIStatusFailure: true,
		CIStatusKilled: true, CIStatusError: true,
		CIStatusBlocked: false, CIStatusWaitingOnDeps: false, CIStatusPending: false, CIStatusRunning: false,
	}
	for status, want := range done {
		require.Equal(t, want, status.IsDone(), status)
	}

	for status, want := range map[CIStatus]bool{
		CIStatusFailure: true, CIStatusKilled: true, CIStatusError: true, CIStatusSuccess: false,
	} {
		require.Equal(t, want, status.IsFailed(), status)
	}

	require.Equal(t, CIStatusRunning, ParseCIStatus("RUNNING"))
	require.Equal(t, CIStatusKilled, ParseCIStatus("killed"))
	require.Equal(t, CIStatusPending, ParseCIStatus(""))
	require.Equal(t, CIStatusError, ParseCIStatus("whatever"))
}

func TestConnectorEnums(t *testing.T) {
	for in, want := range map[string]ConnectorAuthType{
		"basic": ConnectorAuthTypeBasic, "bearer": ConnectorAuthTypeBearer,
	} {
		got, err := ParseConnectorAuthType(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
		require.Equal(t, in, got.String())
		sanitized, ok := got.Sanitize()
		require.True(t, ok)
		require.Equal(t, want, sanitized)
	}
	_, err := ParseConnectorAuthType("digest")
	require.Error(t, err)
	require.Equal(t, "undefined", ConnectorAuthType("digest").String())
	require.Len(t, ConnectorAuthType("").Enum(), 2)
	_, ok := ConnectorAuthType("digest").Sanitize()
	require.False(t, ok)

	got, err := ParseConnectorType("github")
	require.NoError(t, err)
	require.Equal(t, ConnectorTypeGithub, got)
	require.Equal(t, "github", got.String())
	require.True(t, got.IsSCM())
	_, err = ParseConnectorType("svn")
	require.Error(t, err)
	require.Equal(t, "undefined", ConnectorType("svn").String())
	require.False(t, ConnectorType("svn").IsSCM())
}

func TestGitEnums(t *testing.T) {
	require.Equal(t, BranchSortOptionName, ParseBranchSortOption("NAME"))
	require.Equal(t, BranchSortOptionDate, ParseBranchSortOption("date"))
	require.Equal(t, BranchSortOptionDefault, ParseBranchSortOption("other"))
	require.Equal(t, "name", BranchSortOptionName.String())
	require.Equal(t, "date", BranchSortOptionDate.String())
	require.Equal(t, "default", BranchSortOptionDefault.String())
	require.Equal(t, "undefined", BranchSortOption(42).String())

	require.Equal(t, TagSortOptionName, ParseTagSortOption("name"))
	require.Equal(t, TagSortOptionDate, ParseTagSortOption("DATE"))
	require.Equal(t, TagSortOptionDefault, ParseTagSortOption(""))
	require.Equal(t, "name", TagSortOptionName.String())
	require.Equal(t, "date", TagSortOptionDate.String())
	require.Equal(t, "default", TagSortOptionDefault.String())
	require.Equal(t, "undefined", TagSortOption(42).String())

	for in, want := range map[string]GitServiceType{
		"receive-pack": GitServiceTypeReceivePack, "Upload-Pack": GitServiceTypeUploadPack,
	} {
		got, err := ParseGitServiceType(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := ParseGitServiceType("other")
	require.Error(t, err)
}

func TestGitLFSEnums(t *testing.T) {
	for in, want := range map[string]GitLFSTransferType{"basic": GitLFSTransferTypeBasic, "SSH": GitLFSTransferTypeSSH} {
		got, err := ParseGitLFSTransferType(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err := ParseGitLFSTransferType("multipart")
	require.Error(t, err)

	for in, want := range map[string]GitLFSOperationType{
		"download": GitLFSOperationTypeDownload, "UPLOAD": GitLFSOperationTypeUpload,
	} {
		got, err := ParseGitLFSOperationType(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err = ParseGitLFSOperationType("delete")
	require.Error(t, err)

	for in, want := range map[string]GitLFSServiceType{
		"git-lfs-transfer": GitLFSServiceTypeTransfer, "GIT-LFS-AUTHENTICATE": GitLFSServiceTypeAuthenticate,
	} {
		got, err := ParseGitLFSServiceType(in)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	_, err = ParseGitLFSServiceType("other")
	require.Error(t, err)
}
