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

package rules

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/harness/gitness/app/services/protection"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/audit"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

type txRunner struct{ err error }

func (r txRunner) WithTx(ctx context.Context, fn func(ctx context.Context) error, _ ...any) error {
	if r.err != nil {
		return r.err
	}
	return fn(ctx)
}

type listingRuleStore struct {
	store.RuleStore

	rules    []types.Rule
	count    int64
	listErr  error
	countErr error

	gotParents []types.RuleParentInfo
	countCalls int
}

func (f *listingRuleStore) List(
	_ context.Context, parents []types.RuleParentInfo, _ *types.RuleFilter,
) ([]types.Rule, error) {
	f.gotParents = parents
	return f.rules, f.listErr
}

func (f *listingRuleStore) Count(context.Context, []types.RuleParentInfo, *types.RuleFilter) (int64, error) {
	f.countCalls++
	return f.count, f.countErr
}

type ancestorSpaceStore struct {
	store.SpaceStore

	ancestors []int64
	err       error
}

func (f ancestorSpaceStore) GetAncestorIDs(context.Context, int64) ([]int64, error) {
	return f.ancestors, f.err
}

type mapPrincipalCache struct {
	store.PrincipalInfoCache

	infos map[int64]*types.PrincipalInfo
	err   error
}

func (f mapPrincipalCache) Map(context.Context, []int64) (map[int64]*types.PrincipalInfo, error) {
	return f.infos, f.err
}

type mapGroupStore struct {
	store.UserGroupStore

	groups map[int64]*types.UserGroup
	err    error
	calls  int
}

func (f *mapGroupStore) Map(context.Context, []int64) (map[int64]*types.UserGroup, error) {
	f.calls++
	return f.groups, f.err
}

type lookupRepoIDCache struct {
	store.RepoIDCache

	repos map[int64]*types.RepositoryCore
	err   error
}

func (f lookupRepoIDCache) Get(_ context.Context, id int64) (*types.RepositoryCore, error) {
	if f.err != nil {
		return nil, f.err
	}
	r, ok := f.repos[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return r, nil
}

type findRepoStore struct {
	store.RepoStore

	repos map[int64]*types.Repository
	err   error
}

func (f findRepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	if f.err != nil {
		return nil, f.err
	}
	r, ok := f.repos[id]
	if !ok {
		return nil, gitness_store.ErrResourceNotFound
	}
	return r, nil
}

type publishedEvent struct {
	spaceID int64
	typ     enum.SSEType
	data    any
}

type recordingStreamer struct {
	fakeSSEStreamer

	events []publishedEvent
}

func (r *recordingStreamer) Publish(_ context.Context, spaceID int64, typ enum.SSEType, data any) {
	r.events = append(r.events, publishedEvent{spaceID, typ, data})
}

func newCommonTestService(t *testing.T) *Service {
	t.Helper()

	svc, _ := newTestService(t, &capturingAuditService{}, newTestRule(enum.RuleParentRepo))
	svc.tx = txRunner{}
	svc.spaceStore = ancestorSpaceStore{ancestors: []int64{20, 10}}
	return svc
}

func TestRuleTypeToResourceType(t *testing.T) {
	require.Equal(t, audit.ResourceTypeBranchRule, ruleTypeToResourceType(protection.TypeBranch))
	require.Equal(t, audit.ResourceTypeTagRule, ruleTypeToResourceType(protection.TypeTag))
	require.Equal(t, audit.ResourceTypePushRule, ruleTypeToResourceType(protection.TypePush))
	require.Equal(t, audit.ResourceTypeBranchRule, ruleTypeToResourceType("unknown"))
}

func TestGetParentInfo(t *testing.T) {
	boom := errors.New("boom")
	svc := newCommonTestService(t)
	svc.repoStore = findRepoStore{repos: map[int64]*types.Repository{5: {ID: 5, ParentID: 50}}}

	t.Run("repo", func(t *testing.T) {
		got, err := svc.getParentInfoRepo(t.Context(), 5, false)
		require.NoError(t, err)
		require.Equal(t, []types.RuleParentInfo{{ID: 5, Type: enum.RuleParentRepo}}, got)

		got, err = svc.getParentInfoRepo(t.Context(), 5, true)
		require.NoError(t, err)
		require.Equal(t, []types.RuleParentInfo{
			{ID: 5, Type: enum.RuleParentRepo},
			{ID: 20, Type: enum.RuleParentSpace},
			{ID: 10, Type: enum.RuleParentSpace},
		}, got)
	})

	t.Run("repo errors", func(t *testing.T) {
		_, err := svc.getParentInfoRepo(t.Context(), 404, true)
		require.Error(t, err)

		svc.spaceStore = ancestorSpaceStore{err: boom}
		_, err = svc.getParentInfoRepo(t.Context(), 5, true)
		require.ErrorIs(t, err, boom)
	})

	t.Run("space", func(t *testing.T) {
		svc.spaceStore = ancestorSpaceStore{ancestors: []int64{20, 10}}
		got, err := svc.getParentInfoSpace(t.Context(), 20, false)
		require.NoError(t, err)
		require.Equal(t, []types.RuleParentInfo{{ID: 20, Type: enum.RuleParentSpace}}, got)

		got, err = svc.getParentInfoSpace(t.Context(), 20, true)
		require.NoError(t, err)
		require.Equal(t, []types.RuleParentInfo{
			{ID: 20, Type: enum.RuleParentSpace}, {ID: 10, Type: enum.RuleParentSpace},
		}, got)

		svc.spaceStore = ancestorSpaceStore{err: boom}
		_, err = svc.getParentInfoSpace(t.Context(), 20, true)
		require.ErrorIs(t, err, boom)
	})
}

func branchRule(t *testing.T, bypassUsers, bypassGroups []int64, repoTarget protection.RepoTarget) types.Rule {
	t.Helper()

	def, err := json.Marshal(protection.Branch{Bypass: protection.DefBypass{
		UserIDs: bypassUsers, UserGroupIDs: bypassGroups,
	}})
	require.NoError(t, err)

	return types.Rule{
		ID: 1, Identifier: "rule", Type: protection.TypeBranch, State: enum.RuleStateActive,
		Pattern: json.RawMessage(`{}`), Definition: def, RepoTarget: repoTarget.JSON(),
	}
}

func TestService_List(t *testing.T) {
	boom := errors.New("boom")

	setup := func(t *testing.T, rules []types.Rule) (*Service, *listingRuleStore) {
		t.Helper()
		svc := newCommonTestService(t)
		rs := &listingRuleStore{rules: rules, count: 99}
		svc.ruleStore = rs
		svc.repoStore = findRepoStore{repos: map[int64]*types.Repository{5: {ID: 5, ParentID: 50}}}
		svc.principalInfoCache = mapPrincipalCache{infos: map[int64]*types.PrincipalInfo{1: {ID: 1, UID: "jane"}}}
		svc.userGroupStore = &mapGroupStore{groups: map[int64]*types.UserGroup{7: {ID: 7, Identifier: "devs"}}}
		svc.repoIDCache = lookupRepoIDCache{repos: map[int64]*types.RepositoryCore{5: {ID: 5, Identifier: "repo"}}}
		return svc, rs
	}

	t.Run("short first page skips the count query", func(t *testing.T) {
		target := protection.RepoTarget{Include: protection.RepoTargetFilter{IDs: []int64{5}}}
		svc, rs := setup(t, []types.Rule{branchRule(t, []int64{1}, []int64{7}, target)})

		list, count, err := svc.List(t.Context(), 5, enum.RuleParentRepo, true, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
		require.Zero(t, rs.countCalls)
		require.Len(t, rs.gotParents, 3)

		require.Len(t, list, 1)
		require.Equal(t, "jane", list[0].Users[1].UID)
		require.Equal(t, "devs", list[0].UserGroups[7].Identifier)
		require.Equal(t, "repo", list[0].Repositories[5].Identifier)
	})

	t.Run("full page asks for the total", func(t *testing.T) {
		svc, rs := setup(t, []types.Rule{branchRule(t, nil, nil, protection.RepoTarget{})})
		_, count, err := svc.List(t.Context(), 20, enum.RuleParentSpace, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}},
		})
		require.NoError(t, err)
		require.Equal(t, int64(99), count)
		require.Equal(t, 1, rs.countCalls)
		require.Equal(t, []types.RuleParentInfo{{ID: 20, Type: enum.RuleParentSpace}}, rs.gotParents)
	})

	t.Run("unsupported parent type", func(t *testing.T) {
		svc, _ := setup(t, nil)
		_, _, err := svc.List(t.Context(), 1, "bogus", false, &types.RuleFilter{})
		require.Error(t, err)
	})

	t.Run("parent lookup failures", func(t *testing.T) {
		svc, _ := setup(t, nil)
		_, _, err := svc.List(t.Context(), 404, enum.RuleParentRepo, true, &types.RuleFilter{})
		require.Error(t, err)

		svc.spaceStore = ancestorSpaceStore{err: boom}
		_, _, err = svc.List(t.Context(), 20, enum.RuleParentSpace, true, &types.RuleFilter{})
		require.ErrorIs(t, err, boom)
	})

	t.Run("store failures", func(t *testing.T) {
		svc, rs := setup(t, nil)
		rs.listErr = boom
		_, _, err := svc.List(t.Context(), 20, enum.RuleParentSpace, false, &types.RuleFilter{})
		require.ErrorIs(t, err, boom)

		rs.listErr = nil
		rs.countErr = boom
		_, _, err = svc.List(t.Context(), 20, enum.RuleParentSpace, false,
			&types.RuleFilter{ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 1}}})
		require.ErrorIs(t, err, boom)

		svc.tx = txRunner{err: boom}
		_, _, err = svc.List(t.Context(), 20, enum.RuleParentSpace, false, &types.RuleFilter{})
		require.ErrorIs(t, err, boom)
	})

	t.Run("broken rule definitions are reported", func(t *testing.T) {
		bad := branchRule(t, nil, nil, protection.RepoTarget{})
		bad.Type = "unknown"
		svc, _ := setup(t, []types.Rule{bad})
		_, _, err := svc.List(t.Context(), 20, enum.RuleParentSpace, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.Error(t, err)

		brokenTarget := branchRule(t, nil, nil, protection.RepoTarget{})
		brokenTarget.RepoTarget = json.RawMessage(`not json`)
		svc, _ = setup(t, []types.Rule{brokenTarget})
		_, _, err = svc.List(t.Context(), 20, enum.RuleParentSpace, false, &types.RuleFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.Error(t, err)
		require.Contains(t, err.Error(), "backfill")
	})
}

func TestService_RuleUsersAndGroups(t *testing.T) {
	boom := errors.New("boom")
	svc := newCommonTestService(t)
	groups := &mapGroupStore{groups: map[int64]*types.UserGroup{7: {ID: 7, Identifier: "devs", Name: "Devs"}}}
	svc.userGroupStore = groups
	svc.principalInfoCache = mapPrincipalCache{infos: map[int64]*types.PrincipalInfo{1: {ID: 1}}}

	rule := branchRule(t, []int64{1}, []int64{7}, protection.RepoTarget{})
	users, userIDs, userGroups, groupIDs, err := svc.getRuleUserAndUserGroups(t.Context(), &rule)
	require.NoError(t, err)
	require.Contains(t, users, int64(1))
	require.Equal(t, []int64{1}, userIDs)
	require.Equal(t, "Devs", userGroups[7].Name)
	require.Equal(t, []int64{7}, groupIDs)

	t.Run("no groups skips the group lookup", func(t *testing.T) {
		before := groups.calls
		noGroups := branchRule(t, []int64{1}, nil, protection.RepoTarget{})
		_, _, noGroupMap, noGroupIDs, err := svc.getRuleUserAndUserGroups(t.Context(), &noGroups)
		require.NoError(t, err)
		require.Empty(t, noGroupMap)
		require.Empty(t, noGroupIDs)
		require.Equal(t, before, groups.calls)
	})

	t.Run("unknown rule type", func(t *testing.T) {
		bad := rule
		bad.Type = "unknown"
		_, _, _, _, err := svc.getRuleUserAndUserGroups(t.Context(), &bad)
		require.Error(t, err)
		require.Contains(t, err.Error(), "parse rule")
	})

	t.Run("principal cache failure", func(t *testing.T) {
		svc.principalInfoCache = mapPrincipalCache{err: boom}
		_, _, _, _, err := svc.getRuleUserAndUserGroups(t.Context(), &rule)
		require.ErrorIs(t, err, boom)
	})

	t.Run("group store failure", func(t *testing.T) {
		svc.principalInfoCache = mapPrincipalCache{infos: map[int64]*types.PrincipalInfo{}}
		svc.userGroupStore = &mapGroupStore{err: boom}
		_, _, _, _, err := svc.getRuleUserAndUserGroups(t.Context(), &rule)
		require.ErrorIs(t, err, boom)
	})
}

func TestService_BackfillRuleRepositories(t *testing.T) {
	svc := newCommonTestService(t)
	svc.repoIDCache = lookupRepoIDCache{repos: map[int64]*types.RepositoryCore{
		1: {ID: 1, Identifier: "one"}, 2: {ID: 2, Identifier: "two"},
	}}

	target := protection.RepoTarget{
		Include: protection.RepoTargetFilter{IDs: []int64{1, 2, 1}},
		Exclude: protection.RepoTargetFilter{IDs: []int64{2, 404}},
	}
	rule := &types.Rule{RepoTarget: target.JSON()}

	require.NoError(t, svc.backfillRuleRepositories(t.Context(), rule))
	require.Len(t, rule.Repositories, 2, "duplicates and unknown repositories are dropped")
	require.Equal(t, "one", rule.Repositories[1].Identifier)

	t.Run("empty target", func(t *testing.T) {
		empty := &types.Rule{RepoTarget: json.RawMessage(`{}`)}
		require.NoError(t, svc.backfillRuleRepositories(t.Context(), empty))
		require.NotNil(t, empty.Repositories)
		require.Empty(t, empty.Repositories)
	})

	t.Run("invalid json", func(t *testing.T) {
		require.Error(t, svc.backfillRuleRepositories(t.Context(), &types.Rule{RepoTarget: json.RawMessage(`[`)}))
	})

	t.Run("cache failure", func(t *testing.T) {
		svc.repoIDCache = lookupRepoIDCache{err: errors.New("cache down")}
		require.Error(t, svc.backfillRuleRepositories(t.Context(), rule))
	})
}

func TestService_SendSSE(t *testing.T) {
	rule := &types.Rule{ID: 3}

	t.Run("space events go to the space", func(t *testing.T) {
		svc := newCommonTestService(t)
		streamer := &recordingStreamer{}
		svc.sseStreamer = streamer

		svc.sendSSE(t.Context(), 20, enum.RuleParentSpace, enum.SSETypeRuleCreated, rule)
		require.Equal(t, []publishedEvent{{20, enum.SSETypeRuleCreated, rule}}, streamer.events)
	})

	t.Run("repo events go to the parent space", func(t *testing.T) {
		svc := newCommonTestService(t)
		streamer := &recordingStreamer{}
		svc.sseStreamer = streamer
		svc.repoStore = findRepoStore{repos: map[int64]*types.Repository{5: {ID: 5, ParentID: 50}}}

		svc.sendSSE(t.Context(), 5, enum.RuleParentRepo, enum.SSETypeRuleUpdated, rule)
		require.Equal(t, []publishedEvent{{50, enum.SSETypeRuleUpdated, rule}}, streamer.events)
	})

	t.Run("a missing repo publishes nothing", func(t *testing.T) {
		svc := newCommonTestService(t)
		streamer := &recordingStreamer{}
		svc.sseStreamer = streamer
		svc.repoStore = findRepoStore{}

		svc.sendSSE(t.Context(), 5, enum.RuleParentRepo, enum.SSETypeRuleDeleted, rule)
		require.Empty(t, streamer.events)
	})
}
