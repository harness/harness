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

package request

import (
	"net/http"
	"testing"

	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestStringPathGetters(t *testing.T) {
	tests := []struct {
		name  string
		param string
		get   func(*http.Request) (string, error)
	}{
		{"connector ref", PathParamConnectorRef, GetConnectorRefFromPath},
		{"gitspace ref", PathParamGitspaceIdentifier, GetGitspaceRefFromPath},
		{"infra provider ref", PathParamInfraProviderConfigIdentifier, GetInfraProviderRefFromPath},
		{"label key", PathParamLabelKey, GetLabelKeyFromPath},
		{"label value", PathParamLabelValue, GetLabelValueFromPath},
		{"pipeline identifier", PathParamPipelineIdentifier, GetPipelineIdentifierFromPath},
		{"trigger identifier", PathParamTriggerIdentifier, GetTriggerIdentifierFromPath},
		{"principal uid", PathParamPrincipalUID, GetPrincipalUIDFromPath},
		{"user uid", PathParamUserUID, GetUserUIDFromPath},
		{"service account uid", PathParamServiceAccountUID, GetServiceAccountUIDFromPath},
		{"public key identifier", PathParamPublicKeyIdentifier, GetPublicKeyIdentifierFromPath},
		{"rule identifier", PathParamRuleIdentifier, GetRuleIdentifierFromPath},
		{"space ref", PathParamSpaceRef, GetSpaceRefFromPath},
		{"template ref", PathParamTemplateRef, GetTemplateRefFromPath},
		{"template type", PathParamTemplateType, GetTemplateTypeFromPath},
		{"token identifier", PathParamTokenIdentifier, GetTokenIdentifierFromPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get(newTestRequest(t, "/", tt.param, "some%2Fvalue"))
			require.NoError(t, err)
			require.Equal(t, "some/value", got)

			_, err = tt.get(newTestRequest(t, "/"))
			requireBadRequest(t, err, "not found")
		})
	}
}

func TestIntPathGetters(t *testing.T) {
	tests := []struct {
		name  string
		param string
		get   func(*http.Request) (int64, error)
	}{
		{"autolink id", PathParamAutolinkID, GetAutolinkIDFromPath},
		{"resource id", PathParamResourceID, GetResourceIDFromPath},
		{"label id", PathParamLabelID, GetLabelIDFromPath},
		{"execution number", PathParamExecutionNumber, GetExecutionNumberFromPath},
		{"stage number", PathParamStageNumber, GetStageNumberFromPath},
		{"step number", PathParamStepNumber, GetStepNumberFromPath},
		{"user id", PathParamUserID, GetUserIDFromPath},
		{"principal id", PathParamPrincipalID, GetPrincipalIDFromPath},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.get(newTestRequest(t, "/", tt.param, "42"))
			require.NoError(t, err)
			require.Equal(t, int64(42), got)

			_, err = tt.get(newTestRequest(t, "/", tt.param, "0"))
			requireBadRequest(t, err, "must be a positive integer")
			_, err = tt.get(newTestRequest(t, "/"))
			requireBadRequest(t, err, "not found")
		})
	}
}

func TestAuthAndLfsQueryGetters(t *testing.T) {
	req := newTestRequest(t, "/?access_token=tok&include_cookie=true&oid=abc&size=12")

	tok, ok := GetAccessTokenFromQuery(req)
	require.True(t, ok)
	require.Equal(t, "tok", tok)
	_, ok = GetAccessTokenFromQuery(newTestRequest(t, "/"))
	require.False(t, ok)

	cookie, err := GetIncludeCookieFromQueryOrDefault(req, false)
	require.NoError(t, err)
	require.True(t, cookie)
	_, err = GetIncludeCookieFromQueryOrDefault(newTestRequest(t, "/?include_cookie=x"), false)
	requireBadRequest(t, err, "must be a boolean")

	withCookie := newTestRequest(t, "/")
	withCookie.AddCookie(&http.Cookie{Name: "token", Value: "from-cookie"})
	val, ok := GetTokenFromCookie(withCookie, "token")
	require.True(t, ok)
	require.Equal(t, "from-cookie", val)

	oid, err := GetObjectIDFromQuery(req)
	require.NoError(t, err)
	require.Equal(t, "abc", oid)
	_, err = GetObjectIDFromQuery(newTestRequest(t, "/"))
	requireBadRequest(t, err, "not found")

	size, err := GetObjectSizeFromQuery(req)
	require.NoError(t, err)
	require.Equal(t, int64(12), size)
	_, err = GetObjectSizeFromQuery(newTestRequest(t, "/?size=0"))
	requireBadRequest(t, err, "must be a positive integer")
}

func TestListFilters(t *testing.T) {
	list := types.ListQueryFilter{Query: "q", Pagination: types.Pagination{Page: 2, Size: 5}}
	const common = "query=q&page=2&limit=5"

	t.Run("autolink", func(t *testing.T) {
		got, err := ParseAutolinkFilter(newTestRequest(t, "/?"+common+"&inherited=true"))
		require.NoError(t, err)
		require.Equal(t, &types.AutoLinkFilter{ListQueryFilter: list, Inherited: true}, got)
		_, err = ParseAutolinkFilter(newTestRequest(t, "/?inherited=x"))
		require.Error(t, err)
	})

	t.Run("label", func(t *testing.T) {
		got, err := ParseLabelFilter(newTestRequest(t, "/?"+common+"&inherited=true&include_pullreq_count=true"))
		require.NoError(t, err)
		require.Equal(t, &types.LabelFilter{ListQueryFilter: list, Inherited: true, IncludePullreqCount: true}, got)
		_, err = ParseLabelFilter(newTestRequest(t, "/?inherited=x"))
		require.Error(t, err)
		_, err = ParseLabelFilter(newTestRequest(t, "/?include_pullreq_count=x"))
		require.Error(t, err)

		assignable, err := ParseAssignableLabelFilter(newTestRequest(t, "/?"+common+"&assignable=true"))
		require.NoError(t, err)
		require.Equal(t, &types.AssignableLabelFilter{ListQueryFilter: list, Assignable: true}, assignable)
		_, err = ParseAssignableLabelFilter(newTestRequest(t, "/?assignable=x"))
		require.Error(t, err)
	})

	t.Run("checks", func(t *testing.T) {
		require.Equal(t, types.CheckListOptions{ListQueryFilter: list}, ParseCheckListOptions(newTestRequest(t, "/?"+common)))

		got, err := ParseCheckRecentOptions(newTestRequest(t, "/?query=q&since=99"))
		require.NoError(t, err)
		require.Equal(t, types.CheckRecentOptions{Query: "q", Since: 99}, got)
		_, err = ParseCheckRecentOptions(newTestRequest(t, "/?since=0"))
		requireBadRequest(t, err, "must be a positive integer")
	})

	t.Run("executions", func(t *testing.T) {
		got, err := ParseListExecutionsFilterFromRequest(newTestRequest(t,
			"/?"+common+"&pipeline_identifier=build&sort=finished&order=desc"))
		require.NoError(t, err)
		require.Equal(t, list, got.ListQueryFilter)
		require.Equal(t, "build", got.PipelineIdentifier)
		require.Equal(t, enum.OrderDesc, got.Order)
		require.Equal(t, enum.ExecutionSort(enum.ExecutionSortFinished), got.Sort)
		require.Equal(t, enum.ExecutionSort(enum.ExecutionSortStarted), ParseSortExecution(newTestRequest(t, "/?sort=bogus")))
	})

	t.Run("pipelines", func(t *testing.T) {
		got, err := ParseListPipelinesFilterFromRequest(newTestRequest(t, "/?"+common+"&latest=true&last_executions=3"))
		require.NoError(t, err)
		require.Equal(t, list, got.ListQueryFilter)
		require.True(t, got.Latest)
		require.Equal(t, int64(3), got.LastExecutions)

		got, err = ParseListPipelinesFilterFromRequest(newTestRequest(t, "/"))
		require.NoError(t, err)
		require.Equal(t, int64(10), got.LastExecutions)
		require.False(t, got.Latest)

		_, err = ParseListPipelinesFilterFromRequest(newTestRequest(t, "/?last_executions=0"))
		requireBadRequest(t, err, "must be a positive integer")

		require.Equal(t, "dev", GetBranchFromQuery(newTestRequest(t, "/?branch=dev")))
		require.Empty(t, GetBranchFromQuery(newTestRequest(t, "/")))
		require.True(t, GetLatestFromPath(newTestRequest(t, "/?latest=true")))
		require.False(t, GetLatestFromPath(newTestRequest(t, "/?latest=garbage")))
	})

	t.Run("rules", func(t *testing.T) {
		got := ParseRuleFilter(newTestRequest(t, "/?"+common+
			"&state=active&state=active&state=bogus&type=branch&type=push&sort=created&order=desc"))
		require.Equal(t, list, got.ListQueryFilter)
		require.Equal(t, []enum.RuleState{enum.RuleStateActive}, got.States)
		require.ElementsMatch(t, []enum.RuleType{enum.RuleTypeBranch, enum.RuleTypePush}, got.Types)
		require.Equal(t, enum.RuleSortCreated, got.Sort)
		require.Equal(t, enum.OrderDesc, got.Order)

		empty := ParseRuleFilter(newTestRequest(t, "/"))
		require.Empty(t, empty.States)
		require.Empty(t, empty.Types)

		bypass, err := ParseBypassRulesFromQuery(newTestRequest(t, "/?bypass_rules=true"))
		require.NoError(t, err)
		require.True(t, bypass)
		_, err = ParseBypassRulesFromQuery(newTestRequest(t, "/?bypass_rules=x"))
		require.Error(t, err)
		dryRun, err := ParseDryRunRulesFromQuery(newTestRequest(t, "/?dry_run_rules=true"))
		require.NoError(t, err)
		require.True(t, dryRun)
		_, err = ParseDryRunRulesFromQuery(newTestRequest(t, "/?dry_run_rules=x"))
		require.Error(t, err)
	})

	t.Run("repo activity", func(t *testing.T) {
		got, err := ParseRepoActivityFilter(newTestRequest(t, "/?after=5&before=10&page=2&limit=5"))
		require.NoError(t, err)
		require.Equal(t, &types.RepoActivityFilter{After: 5, Before: 10, Page: 2, Size: 5}, got)

		got, err = ParseRepoActivityFilter(newTestRequest(t, "/"))
		require.NoError(t, err)
		require.Equal(t, &types.RepoActivityFilter{Page: 1, Size: PerPageDefault}, got)

		_, err = ParseRepoActivityFilter(newTestRequest(t, "/?after=10&before=5"))
		requireBadRequest(t, err, "greater than or equal")
		_, err = ParseRepoActivityFilter(newTestRequest(t, "/?after=x"))
		require.Error(t, err)
		_, err = ParseRepoActivityFilter(newTestRequest(t, "/?before=x"))
		require.Error(t, err)
	})
}

func TestFavoriteResourceType(t *testing.T) {
	resourceTypes, _ := enum.GetAllResourceTypes()
	require.NotEmpty(t, resourceTypes)
	got, err := ParseResourceType(newTestRequest(t, "/?resource_type="+string(resourceTypes[0])))
	require.NoError(t, err)
	require.Equal(t, resourceTypes[0], got)

	_, err = ParseResourceType(newTestRequest(t, "/?resource_type=bogus"))
	requireBadRequest(t, err, "Invalid resource type")
	_, err = ParseResourceType(newTestRequest(t, "/"))
	requireBadRequest(t, err, "Invalid resource type")
}

func TestGitspaceFilters(t *testing.T) {
	require.Equal(t, enum.GitspaceSortCreated, ParseGitspaceSort(newTestRequest(t, "/?sort=created")))
	require.Equal(t, enum.GitspaceSortLastActivated, ParseGitspaceSort(newTestRequest(t, "/")))
	require.Equal(t, enum.GitspaceOwnerAll, ParseGitspaceOwner(newTestRequest(t, "/?gitspace_owner=all")))
	require.Equal(t, enum.GitspaceOwnerSelf, ParseGitspaceOwner(newTestRequest(t, "/")))

	states := ParseGitspaceStates(newTestRequest(t,
		"/?gitspace_states=running&gitspace_states=running&gitspace_states=bogus"))
	require.Equal(t, []enum.GitspaceFilterState{enum.GitspaceFilterStateRunning}, states)
	require.Empty(t, ParseGitspaceStates(newTestRequest(t, "/")))

	scope := ParseScopeFilter(newTestRequest(t, "/?org_identifiers=a&org_identifiers=b&org_identifiers=a"+
		"&project_identifiers=p&project_identifiers=p"))
	require.Equal(t, types.ScopeFilter{Orgs: []string{"a", "b"}, Projects: []string{"p"}}, scope)

	filter := ParseGitspaceFilter(newTestRequest(t, "/?query=q&page=2&limit=5&sort=created&gitspace_owner=all"+
		"&gitspace_states=stopped&order=asc&org_identifiers=a"))
	require.Equal(t, types.ListQueryFilter{Query: "q", Pagination: types.Pagination{Page: 2, Size: 5}}, filter.QueryFilter)
	require.Equal(t, enum.GitspaceSortCreated, filter.Sort)
	require.Equal(t, enum.GitspaceOwnerAll, filter.Owner)
	require.Equal(t, []enum.GitspaceFilterState{enum.GitspaceFilterStateStopped}, filter.GitspaceFilterStates)
	require.Equal(t, enum.OrderAsc, filter.Order)
	require.Equal(t, []string{"a"}, filter.Orgs)
}

func TestPrincipalFilters(t *testing.T) {
	user := ParseUserFilter(newTestRequest(t, "/?order=desc&page=3&sort=created&limit=7"))
	require.Equal(t, enum.OrderDesc, user.Order)
	require.Equal(t, 3, user.Page)
	require.Equal(t, 7, user.Size)
	require.Equal(t, enum.UserAttrCreated, user.Sort)
	require.Equal(t, enum.UserAttrCreated, ParseSortUser(newTestRequest(t, "/?sort=created")))

	pTypes := ParsePrincipalTypes(newTestRequest(t, "/?type=user&type=user&type=bogus&type=service"))
	require.ElementsMatch(t, []enum.PrincipalType{enum.PrincipalTypeUser, enum.PrincipalTypeService}, pTypes)
	require.Empty(t, ParsePrincipalTypes(newTestRequest(t, "/")))

	filter := ParsePrincipalFilter(newTestRequest(t, "/?query=jane&page=2&limit=4&type=user"))
	require.Equal(t, "jane", filter.Query)
	require.Equal(t, 2, filter.Page)
	require.Equal(t, 4, filter.Size)
	require.Equal(t, []enum.PrincipalType{enum.PrincipalTypeUser}, filter.Types)
}

func TestPublicKeyFilters(t *testing.T) {
	schemes := ParsePublicKeyScheme(newTestRequest(t,
		"/?public_key_scheme=ssh&public_key_scheme=ssh&public_key_scheme=bogus"))
	require.Equal(t, []enum.PublicKeyScheme{enum.PublicKeySchemeSSH}, schemes)
	require.Empty(t, ParsePublicKeyScheme(newTestRequest(t, "/")))

	usages := ParsePublicKeyUsage(newTestRequest(t, "/?public_key_usage=sign&public_key_usage=bogus"))
	require.Equal(t, []enum.PublicKeyUsage{enum.PublicKeyUsageSign}, usages)
	require.Empty(t, ParsePublicKeyUsage(newTestRequest(t, "/")))

	got, err := ParseListPublicKeyQueryFilterFromRequest(newTestRequest(t,
		"/?query=q&page=2&limit=4&order=desc&public_key_scheme=pgp&public_key_usage=sign"))
	require.NoError(t, err)
	require.Equal(t, "q", got.Query)
	require.Equal(t, enum.OrderDesc, got.Order)
	require.Equal(t, []enum.PublicKeyScheme{enum.PublicKeySchemePGP}, got.Schemes)
	require.Equal(t, []enum.PublicKeyUsage{enum.PublicKeyUsageSign}, got.Usages)

	_, err = ParseListPublicKeyQueryFilterFromRequest(newTestRequest(t, "/?sort=bogus"))
	requireBadRequest(t, err, "Invalid value for the sort")
}

func TestSpaceFilters(t *testing.T) {
	got, err := ParseSpaceFilter(newTestRequest(t, "/?query=q&order=desc&page=2&limit=3&sort=created&recursive=true"+
		"&deleted_at=100&deleted_before_or_at=200"))
	require.NoError(t, err)
	require.Equal(t, "q", got.Query)
	require.Equal(t, enum.OrderDesc, got.Order)
	require.Equal(t, enum.SpaceAttrCreated, got.Sort)
	require.True(t, got.Recursive)
	require.Equal(t, int64(100), *got.DeletedAt)
	require.Equal(t, int64(200), *got.DeletedBeforeOrAt)

	got, err = ParseSpaceFilter(newTestRequest(t, "/"))
	require.NoError(t, err)
	require.Nil(t, got.DeletedAt)
	require.Nil(t, got.DeletedBeforeOrAt)

	for _, q := range []string{"recursive=x", "deleted_before_or_at=0", "deleted_at=x"} {
		_, err = ParseSpaceFilter(newTestRequest(t, "/?"+q))
		require.Error(t, err, q)
	}

	require.Equal(t, enum.SpaceAttrNone, ParseSortSpace(newTestRequest(t, "/")))

	include, err := GetIncludeSubspacesFromQuery(newTestRequest(t, "/?include_subspaces=true"))
	require.NoError(t, err)
	require.True(t, include)
	_, err = GetIncludeSubspacesFromQuery(newTestRequest(t, "/?include_subspaces=x"))
	require.Error(t, err)
}
