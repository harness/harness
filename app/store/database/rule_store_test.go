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

package database_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	gitnesscache "github.com/harness/gitness/cache"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func newRule(spaceID, repoID *int64, uid string, typ enum.RuleType, state enum.RuleState) *types.Rule {
	return &types.Rule{
		CreatedBy: userID, Created: 1, Updated: 1, SpaceID: spaceID, RepoID: repoID,
		Identifier: uid, Description: "d-" + uid, Type: typ, State: state,
		Pattern:    json.RawMessage(`{"default":true}`),
		RepoTarget: json.RawMessage(`{}`),
		Definition: json.RawMessage(`{"lifecycle":{"delete_forbidden":true}}`),
	}
}

func TestRuleStore(t *testing.T) {
	ctx, f := newScopedFixture(t)

	infoCache := gitnesscache.NewExtended[int64, *types.PrincipalInfo](
		database.NewPrincipalInfoView(f.db), time.Minute)
	rs := database.NewRuleStore(f.db, infoCache)

	repoBranch := newRule(nil, int64Ptr(1), "repo-branch", enum.RuleTypeBranch, enum.RuleStateActive)
	repoPush := newRule(nil, int64Ptr(1), "repo-push", enum.RuleTypePush, enum.RuleStateDisabled)
	spaceTag := newRule(int64Ptr(1), nil, "space-tag", enum.RuleTypeTag, enum.RuleStateMonitor)
	otherSpace := newRule(int64Ptr(2), nil, "other", enum.RuleTypeBranch, enum.RuleStateActive)
	for _, r := range []*types.Rule{repoBranch, repoPush, spaceTag, otherSpace} {
		require.NoError(t, rs.Create(ctx, r))
		require.NotZero(t, r.ID)
	}

	t.Run("duplicate identifier per parent", func(t *testing.T) {
		dup := newRule(nil, int64Ptr(1), "REPO-BRANCH", enum.RuleTypeBranch, enum.RuleStateActive)
		require.ErrorIs(t, rs.Create(ctx, dup), gitnessstore.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := rs.Find(ctx, repoBranch.ID)
		require.NoError(t, err)
		require.Equal(t, "repo-branch", got.Identifier)
		require.Equal(t, enum.RuleTypeBranch, got.Type)
		require.JSONEq(t, `{"default":true}`, string(got.Pattern))
		require.Equal(t, int64(1), *got.RepoID)
		require.Nil(t, got.SpaceID)
		require.Equal(t, "user_1", got.CreatedByInfo.UID, "the creator is resolved through the principal cache")

		byRepo, err := rs.FindByIdentifier(ctx, enum.RuleParentRepo, 1, "REPO-BRANCH")
		require.NoError(t, err)
		require.Equal(t, repoBranch.ID, byRepo.ID)
		bySpace, err := rs.FindByIdentifier(ctx, enum.RuleParentSpace, 1, "space-tag")
		require.NoError(t, err)
		require.Equal(t, spaceTag.ID, bySpace.ID)

		_, err = rs.Find(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = rs.FindByIdentifier(ctx, enum.RuleParentRepo, 1, "space-tag")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = rs.FindByIdentifier(ctx, enum.RuleParent("bogus"), 1, "x")
		require.Error(t, err)
	})

	t.Run("list and count with filters", func(t *testing.T) {
		parents := []types.RuleParentInfo{
			{Type: enum.RuleParentRepo, ID: 1},
			{Type: enum.RuleParentSpace, ID: 1},
		}
		page := types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}

		tests := []struct {
			name   string
			filter *types.RuleFilter
			want   int
		}{
			{"everything", &types.RuleFilter{ListQueryFilter: page}, 3},
			{"by state", &types.RuleFilter{ListQueryFilter: page, States: []enum.RuleState{enum.RuleStateActive}}, 1},
			{"by several states", &types.RuleFilter{
				ListQueryFilter: page, States: []enum.RuleState{enum.RuleStateActive, enum.RuleStateMonitor},
			}, 2},
			{"by type", &types.RuleFilter{ListQueryFilter: page, Types: []enum.RuleType{enum.RuleTypePush}}, 1},
			{"by query", &types.RuleFilter{
				ListQueryFilter: types.ListQueryFilter{Query: "TAG", Pagination: page.Pagination},
			}, 1},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				list, err := rs.List(ctx, parents, tt.filter)
				require.NoError(t, err)
				require.Len(t, list, tt.want)

				count, err := rs.Count(ctx, parents, tt.filter)
				require.NoError(t, err)
				require.Equal(t, int64(tt.want), count)
			})
		}

		for _, sortBy := range []enum.RuleSort{enum.RuleSortIdentifier, enum.RuleSortCreated, enum.RuleSortUpdated} {
			list, err := rs.List(ctx, parents, &types.RuleFilter{ListQueryFilter: page, Sort: sortBy, Order: enum.OrderDesc})
			require.NoError(t, err, sortBy)
			require.Len(t, list, 3, sortBy)
		}

		byIdentifier, err := rs.List(ctx, parents, &types.RuleFilter{
			ListQueryFilter: page, Sort: enum.RuleSortIdentifier, Order: enum.OrderAsc,
		})
		require.NoError(t, err)
		require.Equal(t, "repo-branch", byIdentifier[0].Identifier)

		_, err = rs.List(ctx, []types.RuleParentInfo{{Type: "bogus", ID: 1}}, &types.RuleFilter{})
		require.Error(t, err)
		_, err = rs.Count(ctx, []types.RuleParentInfo{{Type: "bogus", ID: 1}}, &types.RuleFilter{})
		require.Error(t, err)
	})

	t.Run("repo rules include the parent space and skip disabled rules", func(t *testing.T) {
		all, err := rs.ListAllRepoRules(ctx, 1)
		require.NoError(t, err)
		require.Len(t, all, 2, "the disabled repo rule and the rule of another space are excluded")

		identifiers := map[string]string{}
		for _, r := range all {
			identifiers[r.Identifier] = r.SpacePath + "|" + r.RepoPath
		}
		require.Equal(t, "space_1|", identifiers["space-tag"])
		require.Equal(t, "|space_1/repo_1", identifiers["repo-branch"])

		_, err = rs.ListAllRepoRules(ctx, 1, enum.RuleTypeBranch)
		require.NoError(t, err)
	})

	t.Run("only repo rules", func(t *testing.T) {
		repo := &types.RepositoryCore{ID: 1, ParentID: 1, Identifier: "repo_1"}

		only, err := rs.ListOnlyRepoRules(ctx, repo)
		require.NoError(t, err)
		require.Len(t, only, 1)
		require.Equal(t, "repo-branch", only[0].Identifier)

		typed, err := rs.ListOnlyRepoRules(ctx, repo, enum.RuleTypeTag)
		require.NoError(t, err)
		require.Empty(t, typed)
	})

	t.Run("update", func(t *testing.T) {
		stale := *repoBranch

		repoBranch.State = enum.RuleStateDisabled
		repoBranch.Description = "changed"
		require.NoError(t, rs.Update(ctx, repoBranch))
		require.Equal(t, int64(1), repoBranch.Version)

		got, err := rs.Find(ctx, repoBranch.ID)
		require.NoError(t, err)
		require.Equal(t, enum.RuleStateDisabled, got.State)
		require.Equal(t, "changed", got.Description)

		require.ErrorIs(t, rs.Update(ctx, &stale), gitnessstore.ErrVersionConflict)
	})

	t.Run("move to another parent space and delete", func(t *testing.T) {
		moved, err := rs.UpdateParentSpace(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, int64(1), moved)

		require.NoError(t, rs.Delete(ctx, otherSpace.ID))
		_, err = rs.Find(ctx, otherSpace.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}
