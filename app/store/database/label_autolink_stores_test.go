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
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

func TestLabelAndLabelValueStores(t *testing.T) {
	ctx, f := newScopedFixture(t)
	labels := database.NewLabelStore(f.db)
	values := database.NewLabelValueStore(f.db)

	newLabel := func(spaceID, repoID *int64, scope int64, key string) *types.Label {
		return &types.Label{
			SpaceID: spaceID, RepoID: repoID, Scope: scope, Key: key, Description: "d-" + key,
			Type: enum.LabelTypeDynamic, Color: enum.LabelColorBlue, Created: 1, Updated: 1,
			CreatedBy: userID, UpdatedBy: userID,
		}
	}

	spaceBug := newLabel(int64Ptr(1), nil, 1, "bug")
	spaceDocs := newLabel(int64Ptr(1), nil, 1, "docs")
	repoBug := newLabel(nil, int64Ptr(1), 0, "bug")
	otherSpace := newLabel(int64Ptr(2), nil, 1, "other")
	for _, l := range []*types.Label{spaceBug, spaceDocs, repoBug, otherSpace} {
		require.NoError(t, labels.Define(ctx, l))
		require.NotZero(t, l.ID)
	}

	t.Run("define rejects duplicates in the same scope", func(t *testing.T) {
		err := labels.Define(ctx, newLabel(int64Ptr(1), nil, 1, "BUG"))
		require.ErrorIs(t, err, gitnessstore.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := labels.Find(ctx, int64Ptr(1), nil, "BUG")
		require.NoError(t, err)
		require.Equal(t, spaceBug.ID, got.ID)
		require.Equal(t, enum.LabelColorBlue, got.Color)

		inRepo, err := labels.Find(ctx, nil, int64Ptr(1), "bug")
		require.NoError(t, err)
		require.Equal(t, repoBug.ID, inRepo.ID)

		byID, err := labels.FindByID(ctx, spaceDocs.ID)
		require.NoError(t, err)
		require.Equal(t, "docs", byID.Key)

		_, err = labels.Find(ctx, int64Ptr(2), nil, "bug")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = labels.FindByID(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("find by ids and infos", func(t *testing.T) {
		byIDs, err := labels.FindByIDs(ctx, []int64{spaceBug.ID, repoBug.ID, 9999})
		require.NoError(t, err)
		require.Len(t, byIDs, 2)
		require.Equal(t, "bug", byIDs[repoBug.ID].Key)

		infos, err := labels.FindInfosByIDs(ctx, []int64{spaceBug.ID, spaceDocs.ID})
		require.NoError(t, err)
		require.Len(t, infos, 2)
		require.Equal(t, "docs", infos[spaceDocs.ID].Key)
		require.Equal(t, int64(1), infos[spaceDocs.ID].Scope)
	})

	t.Run("update", func(t *testing.T) {
		spaceDocs.Description = "documentation"
		spaceDocs.Color = enum.LabelColorGreen
		spaceDocs.UpdatedBy = userID
		require.NoError(t, labels.Update(ctx, spaceDocs))

		got, err := labels.FindByID(ctx, spaceDocs.ID)
		require.NoError(t, err)
		require.Equal(t, "documentation", got.Description)
		require.Equal(t, enum.LabelColorGreen, got.Color)
		require.Equal(t, userID, got.UpdatedBy)
	})

	t.Run("list and count per parent", func(t *testing.T) {
		filter := &types.LabelFilter{ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}}

		inSpace, err := labels.List(ctx, int64Ptr(1), nil, filter)
		require.NoError(t, err)
		require.Len(t, inSpace, 2)
		require.Equal(t, "bug", inSpace[0].Key, "labels are ordered by key")

		inRepo, err := labels.List(ctx, nil, int64Ptr(1), filter)
		require.NoError(t, err)
		require.Len(t, inRepo, 1)

		searched, err := labels.List(ctx, int64Ptr(1), nil, &types.LabelFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "DOC", Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Len(t, searched, 1)

		spaceCount, err := labels.CountInSpace(ctx, 1, filter)
		require.NoError(t, err)
		require.Equal(t, int64(2), spaceCount)
		repoCount, err := labels.CountInRepo(ctx, 1, filter)
		require.NoError(t, err)
		require.Equal(t, int64(1), repoCount)
	})

	t.Run("list and count in scopes", func(t *testing.T) {
		filter := &types.LabelFilter{ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}}

		all, err := labels.ListInScopes(ctx, 1, []int64{1}, filter)
		require.NoError(t, err)
		require.Len(t, all, 3)
		// The repository label and the space label share a key, the closer scope sorts first.
		require.Equal(t, "bug", all[0].Key)
		require.Equal(t, int64(0), all[0].Scope)

		count, err := labels.CountInScopes(ctx, 1, []int64{1}, filter)
		require.NoError(t, err)
		require.Equal(t, int64(3), count)

		matching, err := labels.ListInScopes(ctx, 1, []int64{1, 2}, &types.LabelFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "oth", Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Len(t, matching, 1)

		infos, err := labels.ListInfosInScopes(ctx, 1, []int64{1}, &types.AssignableLabelFilter{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Len(t, infos, 3)

		searchedInfos, err := labels.ListInfosInScopes(ctx, 1, []int64{1}, &types.AssignableLabelFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "doc", Pagination: types.Pagination{Page: 1, Size: 10}},
		})
		require.NoError(t, err)
		require.Len(t, searchedInfos, 1)
	})

	t.Run("label values", func(t *testing.T) {
		urgent := &types.LabelValue{
			LabelID: spaceBug.ID, Value: "urgent", Color: enum.LabelColorRed, Created: 1, Updated: 1,
			CreatedBy: userID, UpdatedBy: userID,
		}
		minor := &types.LabelValue{
			LabelID: spaceBug.ID, Value: "minor", Color: enum.LabelColorGreen, Created: 1, Updated: 1,
			CreatedBy: userID, UpdatedBy: userID,
		}
		for _, v := range []*types.LabelValue{urgent, minor} {
			require.NoError(t, values.Define(ctx, v))
			require.NotZero(t, v.ID)
		}
		count, err := labels.IncrementValueCount(ctx, spaceBug.ID, 2)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		require.ErrorIs(t, values.Define(ctx, &types.LabelValue{LabelID: spaceBug.ID, Value: "URGENT"}),
			gitnessstore.ErrDuplicate)

		byValue, err := values.FindByLabelID(ctx, spaceBug.ID, "URGENT")
		require.NoError(t, err)
		require.Equal(t, urgent.ID, byValue.ID)
		byID, err := values.FindByID(ctx, minor.ID)
		require.NoError(t, err)
		require.Equal(t, "minor", byID.Value)
		_, err = values.FindByID(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = values.FindByLabelID(ctx, spaceBug.ID, "missing")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		minor.Value = "trivial"
		minor.Color = enum.LabelColorYellow
		require.NoError(t, values.Update(ctx, minor))
		got, err := values.FindByID(ctx, minor.ID)
		require.NoError(t, err)
		require.Equal(t, "trivial", got.Value)
		require.Equal(t, enum.LabelColorYellow, got.Color)

		list, err := values.List(ctx, spaceBug.ID, types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}})
		require.NoError(t, err)
		require.Len(t, list, 2)
		list, err = values.List(ctx, spaceBug.ID, types.ListQueryFilter{
			Query: "urg", Pagination: types.Pagination{Page: 1, Size: 10},
		})
		require.NoError(t, err)
		require.Len(t, list, 1)

		total, err := values.Count(ctx, spaceBug.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), total)
		total, err = values.Count(ctx, spaceBug.ID, types.ListQueryFilter{Query: "urg"})
		require.NoError(t, err)
		require.Equal(t, int64(1), total)

		infos, err := values.ListInfosByLabelIDs(ctx, []int64{spaceBug.ID, spaceDocs.ID})
		require.NoError(t, err)
		require.Len(t, infos, 1, "only labels that have values show up")
		require.Len(t, infos[spaceBug.ID], 2)
		require.Equal(t, "trivial", *infos[spaceBug.ID][0].Value, "values are ordered by value")

		require.NoError(t, values.Delete(ctx, spaceBug.ID, "URGENT"))
		require.NoError(t, values.DeleteMany(ctx, spaceBug.ID, []string{"trivial", "missing"}))
		total, err = values.Count(ctx, spaceBug.ID, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Zero(t, total)
	})

	t.Run("move labels to another parent space and delete", func(t *testing.T) {
		moved, err := labels.UpdateParentSpace(ctx, 1, 2)
		require.NoError(t, err)
		require.Equal(t, int64(2), moved)

		_, err = labels.Find(ctx, int64Ptr(1), nil, "docs")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		require.NoError(t, labels.Delete(ctx, int64Ptr(2), nil, "DOCS"))
		_, err = labels.FindByID(ctx, spaceDocs.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}

func TestAutoLinkStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	as := database.NewAutoLinkStore(f.db)

	newLink := func(spaceID, repoID *int64, pattern string) *types.AutoLink {
		return &types.AutoLink{
			SpaceID: spaceID, RepoID: repoID, Type: enum.AutoLinkTypeRegex, Pattern: pattern,
			TargetURL: "https://tracker/" + pattern, Created: 1, Updated: 1, CreatedBy: userID, UpdatedBy: userID,
		}
	}

	inSpace := newLink(int64Ptr(1), nil, "JIRA-(\\d+)")
	inRepo := newLink(nil, int64Ptr(1), "GH-(\\d+)")
	otherSpace := newLink(int64Ptr(2), nil, "OTHER-(\\d+)")
	for _, l := range []*types.AutoLink{inSpace, inRepo, otherSpace} {
		require.NoError(t, as.Create(ctx, l))
		require.NotZero(t, l.ID)
	}

	got, err := as.Find(ctx, inSpace.ID)
	require.NoError(t, err)
	require.Equal(t, "JIRA-(\\d+)", got.Pattern)
	require.Equal(t, enum.AutoLinkTypeRegex, got.Type)
	require.Equal(t, int64(1), *got.SpaceID)
	require.Nil(t, got.RepoID)

	_, err = as.Find(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	inSpace.TargetURL = "https://jira/browse/"
	inSpace.UpdatedBy = userID
	require.NoError(t, as.Update(ctx, inSpace))
	updated, err := as.Find(ctx, inSpace.ID)
	require.NoError(t, err)
	require.Equal(t, "https://jira/browse/", updated.TargetURL)
	require.Equal(t, userID, updated.UpdatedBy)

	page := types.Pagination{Page: 1, Size: 10}

	list, err := as.List(ctx, int64Ptr(1), nil, &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: page},
	})
	require.NoError(t, err)
	require.Len(t, list, 1)
	count, err := as.Count(ctx, int64Ptr(1), nil, &types.AutoLinkFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	searched, err := as.List(ctx, nil, int64Ptr(1), &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "gh", Pagination: page},
	})
	require.NoError(t, err)
	require.Len(t, searched, 1)
	count, err = as.Count(ctx, nil, int64Ptr(1), &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "nope"},
	})
	require.NoError(t, err)
	require.Zero(t, count)

	inScopes, err := as.ListInScopes(ctx, 1, []int64{1, 2}, &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Pagination: page},
	})
	require.NoError(t, err)
	require.Len(t, inScopes, 3)
	filtered, err := as.ListInScopes(ctx, 1, []int64{1}, &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "other", Pagination: page},
	})
	require.NoError(t, err)
	require.Empty(t, filtered)

	count, err = as.CountInScopes(ctx, 1, []int64{1, 2}, &types.AutoLinkFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(3), count)
	count, err = as.CountInScopes(ctx, 1, []int64{1}, &types.AutoLinkFilter{
		ListQueryFilter: types.ListQueryFilter{Query: "jira"},
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	require.NoError(t, as.Delete(ctx, inRepo.ID))
	_, err = as.Find(ctx, inRepo.ID)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
}
