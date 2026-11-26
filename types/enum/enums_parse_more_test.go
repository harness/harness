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

func TestSortAndAttrParsers(t *testing.T) {
	t.Run("gitspace", func(t *testing.T) {
		require.Equal(t, GitspaceSortLastUsed, ParseGitspaceSort("LAST_USED"))
		require.Equal(t, GitspaceSortCreated, ParseGitspaceSort("created"))
		require.Equal(t, GitspaceSortCreated, ParseGitspaceSort("created_at"))
		require.Equal(t, GitspaceSortLastActivated, ParseGitspaceSort("last_activated"))
		require.Equal(t, GitspaceSortLastActivated, ParseGitspaceSort("other"))

		require.Equal(t, GitspaceOwnerSelf, ParseGitspaceOwner("self"))
		require.Equal(t, GitspaceOwnerAll, ParseGitspaceOwner("ALL"))
		require.Equal(t, GitspaceOwnerSelf, ParseGitspaceOwner("other"))
	})

	t.Run("membership", func(t *testing.T) {
		require.Equal(t, MembershipUserSortName, ParseMembershipUserSort("name"))
		require.Equal(t, MembershipUserSortCreated, ParseMembershipUserSort("Created_At"))
		require.Equal(t, MembershipUserSortName, ParseMembershipUserSort("other"))
		require.Equal(t, "name", MembershipUserSortName.String())
		require.Equal(t, "created", MembershipUserSortCreated.String())
		require.Equal(t, "undefined", MembershipUserSort("x").String())

		require.Equal(t, MembershipSpaceSortIdentifier, ParseMembershipSpaceSort("name"))
		require.Equal(t, MembershipSpaceSortIdentifier, ParseMembershipSpaceSort("identifier"))
		require.Equal(t, MembershipSpaceSortUID, ParseMembershipSpaceSort("uid"))
		require.Equal(t, MembershipSpaceSortCreated, ParseMembershipSpaceSort("created_at"))
		require.Equal(t, MembershipSpaceSortIdentifier, ParseMembershipSpaceSort("other"))
		require.Equal(t, "uid", MembershipSpaceSortUID.String())
		require.Equal(t, "identifier", MembershipSpaceSortIdentifier.String())
		require.Equal(t, "created", MembershipSpaceSortCreated.String())
		require.Equal(t, "undefined", MembershipSpaceSort("x").String())
	})

	t.Run("repo", func(t *testing.T) {
		for in, want := range map[string]RepoAttr{
			"uid": RepoAttrUID, "identifier": RepoAttrIdentifier, "created": RepoAttrCreated,
			"created_at": RepoAttrCreated, "updated": RepoAttrUpdated, "updated_at": RepoAttrUpdated,
			"deleted": RepoAttrDeleted, "deleted_at": RepoAttrDeleted, "last_git_push": RepoAttrLastGITPush,
			"LAST_GIT_PUSH": RepoAttrLastGITPush, "other": RepoAttrNone,
		} {
			require.Equal(t, want, ParseRepoAttr(in), in)
		}
		for attr, want := range map[RepoAttr]string{
			RepoAttrNone: "", RepoAttrUID: "uid", RepoAttrIdentifier: "identifier", RepoAttrCreated: "created",
			RepoAttrUpdated: "updated", RepoAttrDeleted: "deleted", RepoAttrLastGITPush: "last_git_push",
			RepoAttr(99): "undefined",
		} {
			require.Equal(t, want, attr.String())
		}
	})

	t.Run("space", func(t *testing.T) {
		for in, want := range map[string]SpaceAttr{
			"uid": SpaceAttrUID, "identifier": SpaceAttrIdentifier, "created": SpaceAttrCreated,
			"created_at": SpaceAttrCreated, "updated": SpaceAttrUpdated, "deleted_at": SpaceAttrDeleted,
			"other": SpaceAttrNone,
		} {
			require.Equal(t, want, ParseSpaceAttr(in), in)
		}
		for attr, want := range map[SpaceAttr]string{
			SpaceAttrNone: "", SpaceAttrUID: "uid", SpaceAttrIdentifier: "identifier", SpaceAttrCreated: "created",
			SpaceAttrUpdated: "updated", SpaceAttrDeleted: "deleted", SpaceAttr(99): "undefined",
		} {
			require.Equal(t, want, attr.String())
		}
	})

	t.Run("webhook", func(t *testing.T) {
		for in, want := range map[string]WebhookAttr{
			"id": WebhookAttrID, "uid": WebhookAttrUID, "identifier": WebhookAttrIdentifier,
			"display_name": WebhookAttrDisplayName, "created": WebhookAttrCreated,
			"updated_at": WebhookAttrUpdated, "other": WebhookAttrNone,
		} {
			require.Equal(t, want, ParseWebhookAttr(in), in)
		}
		for attr, want := range map[WebhookAttr]string{
			WebhookAttrNone: "", WebhookAttrID: "id", WebhookAttrUID: "uid", WebhookAttrIdentifier: "identifier",
			WebhookAttrDisplayName: "display_name", WebhookAttrCreated: "created", WebhookAttrUpdated: "updated",
			WebhookAttr(99): "undefined",
		} {
			require.Equal(t, want, attr.String())
		}
	})

	t.Run("rule", func(t *testing.T) {
		for in, want := range map[string]RuleSort{
			"uid": RuleSortUID, "created": RuleSortCreated, "created_at": RuleSortCreated,
			"updated": RuleSortUpdated, "updated_at": RuleSortUpdated, "identifier": RuleSortIdentifier,
			"other": RuleSortIdentifier,
		} {
			require.Equal(t, want, ParseRuleSortAttr(in), in)
		}
	})
}

func TestStringParsers(t *testing.T) {
	t.Run("repo state and type", func(t *testing.T) {
		for state, want := range map[RepoState]string{
			RepoStateActive: "active", RepoStateGitImport: "git-import", RepoStateMigrateGitPush: "migrate-git-push",
			RepoStateMigrateDataImport: "migrate-data-import", RepoStateArchived: "archived",
			RepoStateImportFailed: "import-failed", RepoState(99): "undefined",
		} {
			require.Equal(t, want, state.String())
		}
		require.Empty(t, RepoTypeNormal.String())
		require.Equal(t, "linked", RepoTypeLinked.String())
		require.Equal(t, "undefined", RepoType("x").String())
	})

	t.Run("repo activity", func(t *testing.T) {
		got, err := ParseRepoActivityType("branch-created")
		require.NoError(t, err)
		require.Equal(t, RepoActivityTypeBranchCreated, got)
		_, err = ParseRepoActivityType("nope")
		require.Error(t, err)
	})

	t.Run("resolver", func(t *testing.T) {
		kind, err := ParseResolverKind("plugin")
		require.NoError(t, err)
		require.Equal(t, ResolverKindPlugin, kind)
		require.Equal(t, "plugin", kind.String())
		kind, err = ParseResolverKind("template")
		require.NoError(t, err)
		require.Equal(t, "template", kind.String())
		_, err = ParseResolverKind("x")
		require.Error(t, err)
		require.Equal(t, "undefined", ResolverKind("x").String())

		typ, err := ParseResolverType("step")
		require.NoError(t, err)
		require.Equal(t, ResolverTypeStep, typ)
		require.Equal(t, "step", typ.String())
		typ, err = ParseResolverType("stage")
		require.NoError(t, err)
		require.Equal(t, "stage", typ.String())
		_, err = ParseResolverType("x")
		require.Error(t, err)
		require.Equal(t, "undefined", ResolverType("x").String())
	})

	t.Run("scm", func(t *testing.T) {
		require.Equal(t, []string{"GITNESS", "GITHUB", "GITLAB", "UNKNOWN"}, AllSCMTypeStrings())
	})

	t.Run("get all lists", func(t *testing.T) {
		require.Len(t, GetAllSettingsScopes(), 3)
		require.Len(t, GetAllPublicResourceTypes(), 3)
		require.Len(t, GetAllParentResourceTypes(), 2)
		require.Len(t, GetAllConnectorAuthTypes(), 2)
		require.Len(t, AllInfraProviderTypes(), 5)
	})
}

func TestPredicates(t *testing.T) {
	require.True(t, PullReqActivityKindChangeComment.CanBeCodeComment())
	require.True(t, PullReqActivityKindReviewComment.CanBeCodeComment())
	require.False(t, PullReqActivityKindComment.CanBeCodeComment())
	require.False(t, PullReqActivityKindSystem.CanBeCodeComment())

	for state, want := range map[JobState]bool{
		JobStateScheduled: false, JobStateRunning: false,
		JobStateFinished: true, JobStateFailed: true, JobStateCanceled: true,
	} {
		require.Equal(t, want, state.IsCompleted(), state)
	}

	for state, want := range map[GitspaceInstanceStateType][2]bool{
		GitspaceInstanceStateDeleted:        {true, false},
		GitspaceInstanceStateError:          {true, false},
		GitspaceInstanceStateCleaned:        {true, false},
		GitspaceInstanceStateStarting:       {false, true},
		GitspaceInstanceStateStopping:       {false, true},
		GitSpaceInstanceStateCleaning:       {false, true},
		GitSpaceInstanceStateResetting:      {false, true},
		GitspaceInstanceStateRunning:        {false, false},
		GitspaceInstanceStateStopped:        {false, false},
		GitspaceInstanceStatePendingCleanup: {false, false},
	} {
		require.Equal(t, want[0], state.IsFinalStatus(), state)
		require.Equal(t, want[1], state.IsBusyStatus(), state)
	}

	for typ, want := range map[GitspaceCodeRepoType]bool{
		CodeRepoTypeGitlabOnPrem: true, CodeRepoTypeBitbucketServer: true, CodeRepoTypeGithubEnterprise: true,
		CodeRepoTypeGithub: false, CodeRepoTypeGitness: false,
	} {
		require.Equal(t, want, typ.IsOnPrem(), typ)
	}
	var nilType *GitspaceCodeRepoType
	require.False(t, nilType.IsOnPrem())

	require.True(t, ConnectorTypeGithub.IsSCM())
}

func TestTriggerActionEvents(t *testing.T) {
	for action, want := range map[TriggerAction]TriggerEvent{
		TriggerActionPullReqCreated:       TriggerEventPullRequest,
		TriggerActionPullReqBranchUpdated: TriggerEventPullRequest,
		TriggerActionPullReqReopened:      TriggerEventPullRequest,
		TriggerActionPullReqClosed:        TriggerEventPullRequest,
		TriggerActionPullReqMerged:        TriggerEventPullRequest,
		TriggerActionTagCreated:           TriggerEventTag,
		TriggerActionTagUpdated:           TriggerEventTag,
		TriggerActionBranchCreated:        TriggerEventPush,
		TriggerActionBranchUpdated:        TriggerEventPush,
		"":                                TriggerEventManual,
	} {
		require.Equal(t, want, action.GetTriggerEvent(), action)
	}
}

func TestMembershipRolePermissions(t *testing.T) {
	reader := MembershipRoleReader.Permissions()
	executor := MembershipRoleExecutor.Permissions()
	contributor := MembershipRoleContributor.Permissions()
	owner := MembershipRoleSpaceOwner.Permissions()

	require.NotEmpty(t, reader)
	require.Nil(t, MembershipRole("unknown").Permissions())

	// every role includes the reader permissions
	for _, perms := range [][]Permission{executor, contributor, owner} {
		require.Subset(t, perms, reader)
	}
	require.Contains(t, executor, PermissionPipelineExecute)
	require.Contains(t, contributor, PermissionRepoPush)
	require.Contains(t, owner, PermissionRepoDelete)
	require.NotContains(t, reader, PermissionRepoPush)
	require.NotContains(t, contributor, PermissionRepoDelete)
}

func TestUnmarshalJSONEnums(t *testing.T) {
	t.Run("ide", func(t *testing.T) {
		var ide IDEType
		require.NoError(t, json.Unmarshal([]byte(`"vs_code"`), &ide))
		require.Equal(t, IDETypeVSCode, ide)
		require.Equal(t, "vs_code", ide.String())
		require.NoError(t, json.Unmarshal([]byte(`""`), &ide))
		require.Empty(t, ide)
		require.Error(t, json.Unmarshal([]byte(`"emacs"`), &ide))
		require.Error(t, json.Unmarshal([]byte(`1`), &ide))

		var nilIDE *IDEType
		require.Empty(t, nilIDE.String())

		require.True(t, IsJetBrainsIDE(IDETypeGoland))
		require.False(t, IsJetBrainsIDE(IDETypeVSCode))
	})

	t.Run("infra provider type", func(t *testing.T) {
		var p InfraProviderType
		require.NoError(t, json.Unmarshal([]byte(`"docker"`), &p))
		require.Equal(t, InfraProviderTypeDocker, p)
		require.NoError(t, json.Unmarshal([]byte(`""`), &p))
		require.Empty(t, p)
		require.Error(t, json.Unmarshal([]byte(`"lxc"`), &p))
		require.Error(t, json.Unmarshal([]byte(`1`), &p))
	})

	t.Run("code repo type", func(t *testing.T) {
		var p GitspaceCodeRepoType
		require.NoError(t, json.Unmarshal([]byte(`"github"`), &p))
		require.Equal(t, CodeRepoTypeGithub, p)
		require.NoError(t, json.Unmarshal([]byte(`""`), &p))
		require.Empty(t, p)
		require.Error(t, json.Unmarshal([]byte(`"svn"`), &p))
		require.Error(t, json.Unmarshal([]byte(`1`), &p))
	})
}

func TestEventsMessageMapping(t *testing.T) {
	mapping := EventsMessageMapping()
	require.NotEmpty(t, mapping)
	require.Equal(t, "Setting up Gitspace...", mapping[GitspaceEventTypeGitspaceActionStart])
	for event, msg := range mapping {
		require.NotEmpty(t, msg, event)
	}
}

func TestMergeMethodSanitize(t *testing.T) {
	for _, m := range MergeMethods {
		got, ok := m.Sanitize()
		require.True(t, ok)
		require.Equal(t, m, got)
	}
	_, ok := MergeMethod("bogus").Sanitize()
	require.False(t, ok)
}
