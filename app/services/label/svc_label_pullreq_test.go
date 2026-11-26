// Copyright 2026 Harness, Inc.
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

package label

import (
	"context"
	"errors"
	"testing"

	gitnesserrors "github.com/harness/gitness/errors"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	svcRepoID    = int64(1)
	svcParentID  = int64(10)
	svcPullreqID = int64(77)
)

func svcAssign(
	t *testing.T, env *svcEnv, in *types.PullReqLabelAssignInput,
) (*AssignToPullReqOut, error) {
	t.Helper()
	return env.svc.AssignToPullReq(context.Background(), 5, svcPullreqID, svcRepoID, svcParentID, in)
}

func TestSvcAssignToPullReq(t *testing.T) {
	setup := func() (*svcEnv, *types.Label) {
		env := svcNewEnv()
		l := env.labels.add(&types.Label{RepoID: svcInt(svcRepoID), Key: "k", Type: enum.LabelTypeDynamic})
		return env, l
	}

	t.Run("first assignment without value", func(t *testing.T) {
		env, l := setup()
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityAssign, out.ActivityType)
		assert.Nil(t, out.NewLabelValue)
		assert.Nil(t, out.PullReqLabel.ValueID)
		assert.Contains(t, env.assign.assigned, l.ID)
	})
	t.Run("assign with existing value id", func(t *testing.T) {
		env, l := setup()
		v := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "v"})
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &v.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityAssign, out.ActivityType)
		assert.Equal(t, v.ID, out.NewLabelValue.ID)
	})
	t.Run("assign with new dynamic value defines the value", func(t *testing.T) {
		env, l := setup()
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "fresh"})
		require.NoError(t, err)
		assert.Equal(t, "fresh", out.NewLabelValue.Value)
		require.NotNil(t, out.PullReqLabel.ValueID)
		assert.Equal(t, out.NewLabelValue.ID, *out.PullReqLabel.ValueID)
		assert.Equal(t, []int{1}, env.labels.increment)
	})
	t.Run("assign with existing value by name", func(t *testing.T) {
		env, l := setup()
		v := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "known"})
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "known"})
		require.NoError(t, err)
		assert.Equal(t, v.ID, out.NewLabelValue.ID)
		assert.Empty(t, env.labels.increment, "no new value is defined")
	})
	t.Run("static label rejects new values", func(t *testing.T) {
		env, l := setup()
		l.Type = enum.LabelTypeStatic
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "fresh"})
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))
	})
	t.Run("value of another label is rejected", func(t *testing.T) {
		env, l := setup()
		v := env.values.add(&types.LabelValue{LabelID: l.ID + 100, Value: "v"})
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &v.ID})
		require.Error(t, err)
		assert.Equal(t, gitnesserrors.StatusInvalidArgument, gitnesserrors.AsStatus(err))
	})
	t.Run("unknown value id", func(t *testing.T) {
		env, l := setup()
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: svcInt(999)})
		require.Error(t, err)
	})
	t.Run("unknown label", func(t *testing.T) {
		env, _ := setup()
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: 999})
		require.Error(t, err)
	})
	t.Run("label of another repo", func(t *testing.T) {
		env, l := setup()
		l.RepoID = svcInt(999)
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.Error(t, err)
	})
	t.Run("space label outside the space tree", func(t *testing.T) {
		env, l := setup()
		l.RepoID = nil
		l.SpaceID = svcInt(999)
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.Error(t, err)
	})
	t.Run("space label inside the space tree", func(t *testing.T) {
		env, l := setup()
		l.RepoID = nil
		l.SpaceID = svcInt(11)
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityAssign, out.ActivityType)
	})
	t.Run("space hierarchy failure", func(t *testing.T) {
		env, l := setup()
		l.RepoID = nil
		l.SpaceID = svcInt(11)
		env.spaces.ancErr = errors.New("boom")
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.Error(t, err)
	})
	t.Run("reassigning without a value is a no-op", func(t *testing.T) {
		env, l := setup()
		env.assign.assigned = map[int64]*types.PullReqLabel{l.ID: {LabelID: l.ID}}
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityNoop, out.ActivityType)
	})
	t.Run("adding a value to an assigned label is a reassign", func(t *testing.T) {
		env, l := setup()
		env.assign.assigned = map[int64]*types.PullReqLabel{l.ID: {LabelID: l.ID}}
		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "v"})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityReassign, out.ActivityType)
	})
	t.Run("assigning the same value is a no-op", func(t *testing.T) {
		env, l := setup()
		v := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "same"})
		env.assign.assigned = map[int64]*types.PullReqLabel{l.ID: {LabelID: l.ID, ValueID: &v.ID}}

		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &v.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityNoop, out.ActivityType)
		assert.Equal(t, v.ID, out.OldLabelValue.ID)

		out, err = svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "same"})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityNoop, out.ActivityType)
	})
	t.Run("changing the value is a reassign", func(t *testing.T) {
		env, l := setup()
		oldV := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "old"})
		newV := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "new"})
		env.assign.assigned = map[int64]*types.PullReqLabel{l.ID: {LabelID: l.ID, ValueID: &oldV.ID}}

		out, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &newV.ID})
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityReassign, out.ActivityType)
		assert.Equal(t, oldV.ID, out.OldLabelValue.ID)
		assert.Equal(t, newV.ID, out.NewLabelValue.ID)
	})
	t.Run("stale old value id", func(t *testing.T) {
		env, l := setup()
		env.assign.assigned = map[int64]*types.PullReqLabel{l.ID: {LabelID: l.ID, ValueID: svcInt(999)}}
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "x"})
		require.Error(t, err)
	})
	t.Run("store failures", func(t *testing.T) {
		env, l := setup()
		env.assign.findErr = errors.New("boom")
		_, err := svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.Error(t, err)

		env, l = setup()
		env.assign.assignErr = errors.New("boom")
		_, err = svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID})
		require.Error(t, err)

		env, l = setup()
		env.values.listErr = nil
		env.values.defineErr = errors.New("boom")
		_, err = svcAssign(t, env, &types.PullReqLabelAssignInput{LabelID: l.ID, Value: "x"})
		require.Error(t, err)
	})
}

func TestSvcPreparePullReqLabel(t *testing.T) {
	ctx := context.Background()
	env := svcNewEnv()
	l := env.labels.add(&types.Label{RepoID: svcInt(svcRepoID), Key: "k", Type: enum.LabelTypeDynamic})
	v := env.values.add(&types.LabelValue{LabelID: l.ID, Value: "v"})
	other := env.values.add(&types.LabelValue{LabelID: l.ID + 50, Value: "other"})

	prepare := func(in *types.PullReqLabelAssignInput) (WithValue, error) {
		return env.svc.PreparePullReqLabel(ctx, 5, svcRepoID, svcParentID, in)
	}

	got, err := prepare(&types.PullReqLabelAssignInput{LabelID: l.ID})
	require.NoError(t, err)
	assert.Equal(t, l.ID, got.Label.ID)
	assert.Nil(t, got.Value)

	got, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &v.ID})
	require.NoError(t, err)
	assert.Equal(t, v.ID, got.Value.ID)

	got, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID, Value: "brand-new"})
	require.NoError(t, err)
	assert.Equal(t, "brand-new", got.Value.Value)

	_, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: &other.ID})
	require.Error(t, err)
	_, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID, ValueID: svcInt(999)})
	require.Error(t, err)
	_, err = prepare(&types.PullReqLabelAssignInput{LabelID: 999})
	require.Error(t, err)

	l.RepoID = svcInt(999)
	_, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID})
	require.Error(t, err)
	l.RepoID = svcInt(svcRepoID)

	l.Type = enum.LabelTypeStatic
	_, err = prepare(&types.PullReqLabelAssignInput{LabelID: l.ID, Value: "not-allowed"})
	require.Error(t, err)
}

func TestSvcAssignToPullReqOnCreation(t *testing.T) {
	ctx := context.Background()
	env := svcNewEnv()
	l := &types.Label{ID: 3, Key: "k"}
	v := &types.LabelValue{ID: 9, Value: "v"}

	out, err := env.svc.AssignToPullReqOnCreation(ctx, svcPullreqID, 5,
		&WithValue{Label: l, Value: v}, &types.PullReqLabelAssignInput{LabelID: 3})
	require.NoError(t, err)
	assert.Equal(t, enum.LabelActivityAssign, out.ActivityType)
	require.NotNil(t, out.PullReqLabel.ValueID)
	assert.Equal(t, int64(9), *out.PullReqLabel.ValueID)
	assert.Equal(t, svcPullreqID, out.PullReqLabel.PullReqID)

	out, err = env.svc.AssignToPullReqOnCreation(ctx, svcPullreqID, 5,
		&WithValue{Label: l}, &types.PullReqLabelAssignInput{LabelID: 3})
	require.NoError(t, err)
	assert.Nil(t, out.PullReqLabel.ValueID)

	env.assign.assignErr = errors.New("boom")
	_, err = env.svc.AssignToPullReqOnCreation(ctx, svcPullreqID, 5,
		&WithValue{Label: l}, &types.PullReqLabelAssignInput{LabelID: 3})
	require.Error(t, err)
}

func TestSvcUnassignFromPullReq(t *testing.T) {
	ctx := context.Background()
	setup := func() (*svcEnv, *types.Label) {
		env := svcNewEnv()
		return env, env.labels.add(&types.Label{RepoID: svcInt(svcRepoID), Key: "k"})
	}

	t.Run("assigned label with value", func(t *testing.T) {
		env, l := setup()
		env.assign.deleted = true
		env.assign.value = &types.LabelValue{ID: 4, Value: "v"}
		label, value, activity, err := env.svc.UnassignFromPullReq(ctx, svcRepoID, svcParentID, svcPullreqID, l.ID)
		require.NoError(t, err)
		assert.Equal(t, l.ID, label.ID)
		assert.Equal(t, "v", value.Value)
		assert.Equal(t, enum.LabelActivityUnassign, activity)
		assert.Equal(t, []int64{l.ID}, env.assign.unassigned)
	})
	t.Run("label that was not assigned", func(t *testing.T) {
		env, l := setup()
		env.assign.valueErr = gitnessstore.ErrResourceNotFound
		_, _, activity, err := env.svc.UnassignFromPullReq(ctx, svcRepoID, svcParentID, svcPullreqID, l.ID)
		require.NoError(t, err)
		assert.Equal(t, enum.LabelActivityNoop, activity)
	})
	t.Run("failures", func(t *testing.T) {
		env, l := setup()
		_, _, _, err := env.svc.UnassignFromPullReq(ctx, svcRepoID, svcParentID, svcPullreqID, 999)
		require.Error(t, err)

		env.assign.valueErr = errors.New("boom")
		_, _, _, err = env.svc.UnassignFromPullReq(ctx, svcRepoID, svcParentID, svcPullreqID, l.ID)
		require.Error(t, err)

		l.RepoID = svcInt(999)
		_, _, _, err = env.svc.UnassignFromPullReq(ctx, svcRepoID, svcParentID, svcPullreqID, l.ID)
		require.Error(t, err)
	})
}

func TestSvcListPullReqLabels(t *testing.T) {
	ctx := context.Background()
	repo := &types.RepositoryCore{ID: svcRepoID, Path: "root/space/repo"}

	setup := func() *svcEnv {
		env := svcNewEnv()
		repoLabel := env.labels.add(&types.Label{
			RepoID: svcInt(svcRepoID), Key: "b-repo", Scope: 0, Type: enum.LabelTypeStatic,
		})
		spaceLabel := env.labels.add(&types.Label{SpaceID: svcInt(10), Key: "a-space", Scope: 1})
		env.assign.listed = map[int64]*types.LabelAssignment{
			repoLabel.ID: {LabelInfo: types.LabelInfo{ID: repoLabel.ID, Key: "b-repo", Scope: 0}},
			spaceLabel.ID: {LabelInfo: types.LabelInfo{
				ID: spaceLabel.ID, Key: "a-space", Scope: 1, SpaceID: svcInt(10),
			}},
		}
		return env
	}

	t.Run("only assigned labels", func(t *testing.T) {
		env := setup()
		out, total, err := env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID, &types.AssignableLabelFilter{})
		require.NoError(t, err)
		assert.Zero(t, total)
		require.Len(t, out.LabelData, 2)
		assert.Equal(t, "a-space", out.LabelData[0].Key, "assignments are sorted by key")
		assert.Equal(t, "b-repo", out.LabelData[1].Key)
	})
	t.Run("assignable labels", func(t *testing.T) {
		env := setup()
		out, total, err := env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID,
			&types.AssignableLabelFilter{Assignable: true})
		require.NoError(t, err)
		assert.Equal(t, int64(2), total)
		require.Len(t, out.LabelData, 2)
		for _, l := range out.LabelData {
			require.NotNil(t, l.Assigned)
			assert.True(t, *l.Assigned, "both labels are assigned to the pull request")
			assert.Len(t, l.Values, 1)
		}
		require.Len(t, out.ScopeData, 2)
		assert.Equal(t, int64(0), out.ScopeData[0].Scope)
		assert.Same(t, repo, out.ScopeData[0].Repo)
		assert.Equal(t, int64(1), out.ScopeData[1].Scope)
		require.NotNil(t, out.ScopeData[1].Space)
		assert.Equal(t, int64(10), out.ScopeData[1].Space.ID)
	})
	t.Run("unassigned labels are marked", func(t *testing.T) {
		env := setup()
		env.assign.listed = nil
		out, _, err := env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID,
			&types.AssignableLabelFilter{Assignable: true})
		require.NoError(t, err)
		require.Len(t, out.LabelData, 2)
		for _, l := range out.LabelData {
			require.NotNil(t, l.Assigned)
			assert.False(t, *l.Assigned)
		}
	})
	t.Run("failures", func(t *testing.T) {
		env := setup()
		env.spaces.ancErr = errors.New("boom")
		_, _, err := env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID, &types.AssignableLabelFilter{})
		require.Error(t, err)

		env = setup()
		env.spaces.ancestors = []int64{404}
		_, _, err = env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID, &types.AssignableLabelFilter{})
		require.Error(t, err)

		env = setup()
		env.assign.listErr = errors.New("boom")
		_, _, err = env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID, &types.AssignableLabelFilter{})
		require.Error(t, err)

		env = setup()
		env.labels.countErr = errors.New("boom")
		_, _, err = env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID,
			&types.AssignableLabelFilter{Assignable: true})
		require.Error(t, err)

		env = setup()
		env.labels.listErr = errors.New("boom")
		_, _, err = env.svc.ListPullReqLabels(ctx, repo, svcParentID, svcPullreqID,
			&types.AssignableLabelFilter{Assignable: true})
		require.Error(t, err)
	})
}

func TestSvcBackfill(t *testing.T) {
	ctx := context.Background()
	env := svcNewEnv()
	env.assign.byPullreq = map[int64][]*types.LabelPullReqAssignmentInfo{
		1: {{LabelID: 10}},
		2: {{LabelID: 20}, {LabelID: 21}},
	}

	pr := &types.PullReq{ID: 1}
	require.NoError(t, env.svc.Backfill(ctx, pr))
	assert.Len(t, pr.Labels, 1)

	prs := []*types.PullReq{{ID: 1}, {ID: 2}, {ID: 3}}
	require.NoError(t, env.svc.BackfillMany(ctx, prs))
	assert.Len(t, prs[0].Labels, 1)
	assert.Len(t, prs[1].Labels, 2)
	assert.Empty(t, prs[2].Labels)

	env.assign.listErr = errors.New("boom")
	require.Error(t, env.svc.Backfill(ctx, pr))
	require.Error(t, env.svc.BackfillMany(ctx, prs))
}

func TestSvcToLabelPullReqAssignmentInfo(t *testing.T) {
	color := enum.LabelColorRed
	out := &AssignToPullReqOut{
		Label:        &types.Label{ID: 3, Key: "k", Color: enum.LabelColorBlue, Scope: 2, ValueCount: 5},
		PullReqLabel: &types.PullReqLabel{PullReqID: 9},
		NewLabelValue: &types.LabelValue{
			ID: 4, Value: "v", Color: color,
		},
	}
	info := out.ToLabelPullReqAssignmentInfo()
	assert.Equal(t, int64(9), info.PullReqID)
	assert.Equal(t, int64(3), info.LabelID)
	assert.Equal(t, "k", info.LabelKey)
	assert.Equal(t, int64(2), info.LabelScope)
	assert.Equal(t, int64(5), info.ValueCount)
	require.NotNil(t, info.ValueID)
	assert.Equal(t, int64(4), *info.ValueID)
	assert.Equal(t, "v", *info.Value)
	assert.Equal(t, color, *info.ValueColor)

	out.NewLabelValue = nil
	info = out.ToLabelPullReqAssignmentInfo()
	assert.Nil(t, info.ValueID)
	assert.Nil(t, info.Value)
	assert.Nil(t, info.ValueColor)
}
