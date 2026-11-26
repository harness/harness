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
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func prqNewPullReqLabel(prID, labelID int64, valueID *int64, by int64, at int64) *types.PullReqLabel {
	return &types.PullReqLabel{
		PullReqID: prID,
		LabelID:   labelID,
		ValueID:   valueID,
		Created:   at,
		Updated:   at,
		CreatedBy: by,
		UpdatedBy: by,
	}
}

func TestPrqPullReqLabelStore_AssignAndFind(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	labelID := prqDefineLabel(f.ctx, t, f.db, "bug")
	valueID := prqDefineLabelValue(f.ctx, t, f.db, labelID, "critical")
	otherValueID := prqDefineLabelValue(f.ctx, t, f.db, labelID, "minor")
	store := database.NewPullReqLabelStore(f.db)

	assigned := prqNewPullReqLabel(pr.ID, labelID, &valueID, prqUserA, prqBaseTime)
	require.NoError(t, store.Assign(f.ctx, assigned))
	require.Equal(t, prqBaseTime, assigned.Created)
	require.Equal(t, prqUserA, assigned.CreatedBy)

	got, err := store.FindByLabelID(f.ctx, pr.ID, labelID)
	require.NoError(t, err)
	require.NotNil(t, got.ValueID)
	require.Equal(t, valueID, *got.ValueID)
	require.Equal(t, prqBaseTime, got.Created)

	// re-assigning another value updates the value but keeps the original creation data.
	reassigned := prqNewPullReqLabel(pr.ID, labelID, &otherValueID, prqUserB, prqBaseTime+50)
	require.NoError(t, store.Assign(f.ctx, reassigned))
	require.Equal(t, prqBaseTime, reassigned.Created)
	require.Equal(t, prqUserA, reassigned.CreatedBy)

	got, err = store.FindByLabelID(f.ctx, pr.ID, labelID)
	require.NoError(t, err)
	require.Equal(t, otherValueID, *got.ValueID)
	require.Equal(t, prqBaseTime+50, got.Updated)
	require.Equal(t, prqUserB, got.UpdatedBy)
	require.Equal(t, prqUserA, got.CreatedBy)

	// re-assigning without a value clears it.
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr.ID, labelID, nil, prqUserB, prqBaseTime+60)))
	got, err = store.FindByLabelID(f.ctx, pr.ID, labelID)
	require.NoError(t, err)
	require.Nil(t, got.ValueID)

	_, err = store.FindByLabelID(f.ctx, pr.ID, labelID+100)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqLabelStore_AssignErrors(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	labelID := prqDefineLabel(f.ctx, t, f.db, "bug")
	store := database.NewPullReqLabelStore(f.db)

	err := store.Assign(f.ctx, prqNewPullReqLabel(pr.ID, 9999, nil, prqUserA, prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	err = store.Assign(f.ctx, prqNewPullReqLabel(9999, labelID, nil, prqUserA, prqBaseTime))
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}

func TestPrqPullReqLabelStore_Unassign(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	labelID := prqDefineLabel(f.ctx, t, f.db, "bug")
	store := database.NewPullReqLabelStore(f.db)

	removed, err := store.Unassign(f.ctx, pr.ID, labelID)
	require.NoError(t, err)
	require.False(t, removed)

	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr.ID, labelID, nil, prqUserA, prqBaseTime)))

	removed, err = store.Unassign(f.ctx, pr.ID, labelID)
	require.NoError(t, err)
	require.True(t, removed)

	_, err = store.FindByLabelID(f.ctx, pr.ID, labelID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqLabelStore_ListAssigned(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	bugID := prqDefineLabel(f.ctx, t, f.db, "bug")
	areaID := prqDefineLabel(f.ctx, t, f.db, "area")
	highID := prqDefineLabelValue(f.ctx, t, f.db, bugID, "high")
	store := database.NewPullReqLabelStore(f.db)

	empty, err := store.ListAssigned(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Empty(t, empty)

	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr.ID, bugID, &highID, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr.ID, areaID, nil, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(other.ID, areaID, nil, prqUserA, prqBaseTime)))

	assigned, err := store.ListAssigned(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Len(t, assigned, 2)

	require.Equal(t, "bug", assigned[bugID].Key)
	require.Equal(t, enum.LabelTypeStatic, assigned[bugID].Type)
	require.NotNil(t, assigned[bugID].RepoID)
	require.Equal(t, prqRepoMain, *assigned[bugID].RepoID)
	require.NotNil(t, assigned[bugID].AssignedValue)
	require.Equal(t, highID, *assigned[bugID].AssignedValue.ID)
	require.Equal(t, "high", *assigned[bugID].AssignedValue.Value)

	require.Equal(t, "area", assigned[areaID].Key)
	require.Nil(t, assigned[areaID].AssignedValue)
}

func TestPrqPullReqLabelStore_ListAssignedByPullreqIDs(t *testing.T) {
	f := prqNewFixture(t)
	pr1 := f.prqCreatePullReq(t, prqRepoMain, 1)
	pr2 := f.prqCreatePullReq(t, prqRepoMain, 2)
	pr3 := f.prqCreatePullReq(t, prqRepoMain, 3)
	bugID := prqDefineLabel(f.ctx, t, f.db, "bug")
	areaID := prqDefineLabel(f.ctx, t, f.db, "area")
	highID := prqDefineLabelValue(f.ctx, t, f.db, bugID, "high")
	store := database.NewPullReqLabelStore(f.db)

	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr1.ID, bugID, &highID, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr1.ID, areaID, nil, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr2.ID, areaID, nil, prqUserA, prqBaseTime)))

	res, err := store.ListAssignedByPullreqIDs(f.ctx, []int64{pr1.ID, pr2.ID, pr3.ID})
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Len(t, res[pr1.ID], 2)
	require.Len(t, res[pr2.ID], 1)
	require.Empty(t, res[pr3.ID])

	byLabel := map[int64]*types.LabelPullReqAssignmentInfo{}
	for _, info := range res[pr1.ID] {
		byLabel[info.LabelID] = info
	}
	require.Equal(t, "bug", byLabel[bugID].LabelKey)
	require.Equal(t, enum.LabelColorBlue, byLabel[bugID].LabelColor)
	require.NotNil(t, byLabel[bugID].ValueID)
	require.Equal(t, highID, *byLabel[bugID].ValueID)
	require.Equal(t, "high", *byLabel[bugID].Value)
	require.NotNil(t, byLabel[bugID].ValueColor)
	require.Equal(t, enum.LabelColorRed, *byLabel[bugID].ValueColor)
	require.Nil(t, byLabel[areaID].ValueID)
	require.Nil(t, byLabel[areaID].ValueColor)

	none, err := store.ListAssignedByPullreqIDs(f.ctx, []int64{pr3.ID})
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestPrqPullReqLabelStore_FindValue(t *testing.T) {
	f := prqNewFixture(t)
	pr1 := f.prqCreatePullReq(t, prqRepoMain, 1)
	pr2 := f.prqCreatePullReq(t, prqRepoMain, 2)
	bugID := prqDefineLabel(f.ctx, t, f.db, "bug")
	areaID := prqDefineLabel(f.ctx, t, f.db, "area")
	highID := prqDefineLabelValue(f.ctx, t, f.db, bugID, "high")
	store := database.NewPullReqLabelStore(f.db)

	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr1.ID, bugID, &highID, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr2.ID, bugID, nil, prqUserA, prqBaseTime)))
	require.NoError(t, store.Assign(f.ctx, prqNewPullReqLabel(pr1.ID, areaID, nil, prqUserA, prqBaseTime)))

	value, err := store.FindValueByLabelID(f.ctx, pr1.ID, bugID)
	require.NoError(t, err)
	require.Equal(t, highID, value.ID)
	require.Equal(t, "high", value.Value)
	require.Equal(t, bugID, value.LabelID)

	// no value assigned, and no assignment at all, both mean not found.
	_, err = store.FindValueByLabelID(f.ctx, pr2.ID, bugID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	_, err = store.FindValueByLabelID(f.ctx, pr2.ID, areaID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqLabelSuggestionStore_CreateFindDelete(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	bugID := prqDefineLabel(f.ctx, t, f.db, "bug")
	areaID := prqDefineLabel(f.ctx, t, f.db, "area")
	highID := prqDefineLabelValue(f.ctx, t, f.db, bugID, "high")
	lowID := prqDefineLabelValue(f.ctx, t, f.db, bugID, "low")
	store := database.NewPullReqLabelSuggestionStore(f.db)

	// an empty batch is a no-op.
	require.NoError(t, store.CreateMany(f.ctx, nil))

	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqLabelSuggestion{
		{PullReqID: pr.ID, PrincipalID: prqUserB, LabelID: bugID, ValueID: &highID},
		{PullReqID: pr.ID, PrincipalID: prqUserB, LabelID: areaID},
	}))

	count, err := store.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	got, err := store.Find(f.ctx, pr.ID, bugID)
	require.NoError(t, err)
	require.Equal(t, prqUserB, got.PrincipalID)
	require.NotNil(t, got.ValueID)
	require.Equal(t, highID, *got.ValueID)
	require.NotZero(t, got.CreatedAt)

	got, err = store.Find(f.ctx, pr.ID, areaID)
	require.NoError(t, err)
	require.Nil(t, got.ValueID)

	// the same value from somebody else is ignored.
	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqLabelSuggestion{
		{PullReqID: pr.ID, PrincipalID: prqUserC, LabelID: bugID, ValueID: &highID},
	}))
	got, err = store.Find(f.ctx, pr.ID, bugID)
	require.NoError(t, err)
	require.Equal(t, prqUserB, got.PrincipalID)

	// a different value replaces the existing suggestion.
	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqLabelSuggestion{
		{PullReqID: pr.ID, PrincipalID: prqUserC, LabelID: bugID, ValueID: &lowID},
	}))
	got, err = store.Find(f.ctx, pr.ID, bugID)
	require.NoError(t, err)
	require.Equal(t, prqUserC, got.PrincipalID)
	require.Equal(t, lowID, *got.ValueID)

	count, err = store.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	require.NoError(t, store.Delete(f.ctx, pr.ID, bugID))

	_, err = store.Find(f.ctx, pr.ID, bugID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	err = store.Delete(f.ctx, pr.ID, bugID)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqLabelSuggestionStore_List(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	store := database.NewPullReqLabelSuggestionStore(f.db)

	labels := make([]int64, 0, 3)
	for _, key := range []string{"one", "two", "three"} {
		labels = append(labels, prqDefineLabel(f.ctx, t, f.db, key))
	}

	for _, labelID := range labels {
		require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqLabelSuggestion{
			{PullReqID: pr.ID, PrincipalID: prqUserB, LabelID: labelID},
		}))
	}
	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqLabelSuggestion{
		{PullReqID: other.ID, PrincipalID: prqUserB, LabelID: labels[0]},
	}))

	all, err := store.List(f.ctx, pr.ID, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Len(t, all, 3)
	for _, s := range all {
		require.Equal(t, pr.ID, s.PullReqID)
	}

	// pages don't overlap and together cover all suggestions.
	seen := map[int64]bool{}
	for page := 1; page <= 2; page++ {
		res, listErr := store.List(f.ctx, pr.ID, types.ListQueryFilter{Pagination: types.Pagination{Page: page, Size: 2}})
		require.NoError(t, listErr)
		require.Len(t, res, map[int]int{1: 2, 2: 1}[page])
		for _, s := range res {
			require.False(t, seen[s.LabelID])
			seen[s.LabelID] = true
		}
	}
	require.Len(t, seen, 3)

	none, err := store.List(f.ctx, 9999, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Empty(t, none)

	count, err := store.Count(f.ctx, 9999)
	require.NoError(t, err)
	require.Equal(t, int64(0), count)
}

func TestPrqPullReqLabelSuggestionStore_FindNotFound(t *testing.T) {
	f := prqNewFixture(t)
	store := database.NewPullReqLabelSuggestionStore(f.db)

	_, err := store.Find(f.ctx, 1, 1)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
}

func TestPrqPullReqReviewerSuggestionStore_Lifecycle(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	other := f.prqCreatePullReq(t, prqRepoMain, 2)
	store := database.NewPullReqReviewerSuggestionStore(f.db)

	require.NoError(t, store.CreateMany(f.ctx, nil))

	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqReviewerSuggestion{
		{PullReqID: pr.ID, CreatedBy: prqUserA, PrincipalID: prqUserB},
		{PullReqID: pr.ID, CreatedBy: prqUserA, PrincipalID: prqUserC},
		{PullReqID: other.ID, CreatedBy: prqUserA, PrincipalID: prqUserB},
	}))

	// duplicates are silently ignored and don't change the original suggestion.
	require.NoError(t, store.CreateMany(f.ctx, []*types.PullReqReviewerSuggestion{
		{PullReqID: pr.ID, CreatedBy: prqUserC, PrincipalID: prqUserB},
	}))

	count, err := store.Count(f.ctx, pr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	got, err := store.Find(f.ctx, pr.ID, prqUserB)
	require.NoError(t, err)
	require.Equal(t, pr.ID, got.PullReqID)
	require.Equal(t, prqUserA, got.CreatedBy)
	require.Equal(t, prqUserB, got.PrincipalID)
	require.NotZero(t, got.Created)

	list, err := store.List(f.ctx, pr.ID, types.Pagination{})
	require.NoError(t, err)
	require.Len(t, list, 2)

	page, err := store.List(f.ctx, pr.ID, types.Pagination{Page: 2, Size: 1})
	require.NoError(t, err)
	require.Len(t, page, 1)

	page, err = store.List(f.ctx, pr.ID, types.Pagination{Page: 3, Size: 1})
	require.NoError(t, err)
	require.Empty(t, page)

	require.NoError(t, store.Delete(f.ctx, pr.ID, prqUserB))

	_, err = store.Find(f.ctx, pr.ID, prqUserB)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	err = store.Delete(f.ctx, pr.ID, prqUserB)
	require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

	// the other pull request's suggestion is untouched.
	_, err = store.Find(f.ctx, other.ID, prqUserB)
	require.NoError(t, err)
}

func TestPrqPullReqReviewerSuggestionStore_Errors(t *testing.T) {
	f := prqNewFixture(t)
	pr := f.prqCreatePullReq(t, prqRepoMain, 1)
	store := database.NewPullReqReviewerSuggestionStore(f.db)

	err := store.CreateMany(f.ctx, []*types.PullReqReviewerSuggestion{
		{PullReqID: pr.ID, CreatedBy: prqUserA, PrincipalID: 9999},
	})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)

	err = store.CreateMany(f.ctx, []*types.PullReqReviewerSuggestion{
		{PullReqID: 9999, CreatedBy: prqUserA, PrincipalID: prqUserB},
	})
	require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
}
