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

package label_test

import (
	"context"
	"testing"

	"github.com/harness/gitness/app/services/label"
	"github.com/harness/gitness/app/services/refcache"
	mockstore "github.com/harness/gitness/mocks/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTaskLabels is an in-memory store.TaskLabelStore.
type fakeTaskLabels struct {
	rows map[[2]int64]*types.TaskLabel
}

func (f *fakeTaskLabels) Assign(_ context.Context, l *types.TaskLabel) error {
	f.rows[[2]int64{l.TaskID, l.LabelID}] = l
	return nil
}

func (f *fakeTaskLabels) Unassign(_ context.Context, taskID, labelID int64) (bool, error) {
	k := [2]int64{taskID, labelID}
	_, ok := f.rows[k]
	delete(f.rows, k)
	return ok, nil
}

func (f *fakeTaskLabels) ListAssigned(context.Context, int64) (map[int64]*types.LabelAssignment, error) {
	return nil, nil
}

func (f *fakeTaskLabels) FindByLabelID(_ context.Context, taskID, labelID int64) (*types.TaskLabel, error) {
	if l, ok := f.rows[[2]int64{taskID, labelID}]; ok {
		return l, nil
	}
	return nil, basestore.ErrResourceNotFound
}

func (f *fakeTaskLabels) ListAssignedByTaskIDs(
	context.Context, []int64,
) (map[int64][]*types.LabelTaskAssignmentInfo, error) {
	return nil, nil
}

func TestTaskLabelAssignAndUnassign(t *testing.T) {
	t.Parallel()

	const (
		repoID   = int64(1)
		parentID = int64(10)
		taskID   = int64(77)
		labelID  = int64(123)
		value1ID = int64(1001)
		value2ID = int64(1002)
	)

	repo := repoID
	lbl := &types.Label{ID: labelID, RepoID: &repo, Key: "area", Type: enum.LabelTypeStatic}
	v1 := &types.LabelValue{ID: value1ID, LabelID: labelID, Value: "api"}
	v2 := &types.LabelValue{ID: value2ID, LabelID: labelID, Value: "ui"}

	labelStore := &mockstore.LabelStore{}
	valueStore := &mockstore.LabelValueStore{}
	labelStore.On("FindByID", labelID).Return(lbl, nil)
	valueStore.On("FindByID", value1ID).Return(v1, nil)
	valueStore.On("FindByID", value2ID).Return(v2, nil)

	taskLabels := &fakeTaskLabels{rows: map[[2]int64]*types.TaskLabel{}}
	svc := label.New(nil, nil, labelStore, valueStore, nil, nil, nil, refcache.SpaceFinder{}).
		WithTaskLabelStore(taskLabels)
	ctx := context.Background()

	// first assignment
	out, err := svc.AssignToTask(ctx, 5, taskID, repoID, parentID,
		&types.TaskLabelAssignInput{LabelID: labelID, ValueID: ptr(value1ID)})
	require.NoError(t, err)
	assert.Equal(t, enum.LabelActivityAssign, out.ActivityType)
	info := out.ToLabelTaskAssignmentInfo()
	assert.Equal(t, taskID, info.TaskID)
	assert.Equal(t, "area", info.LabelKey)
	require.NotNil(t, info.Value)
	assert.Equal(t, "api", *info.Value)
	assert.Len(t, taskLabels.rows, 1)

	// same value again is a no-op
	out, err = svc.AssignToTask(ctx, 5, taskID, repoID, parentID,
		&types.TaskLabelAssignInput{LabelID: labelID, ValueID: ptr(value1ID)})
	require.NoError(t, err)
	assert.Equal(t, enum.LabelActivityNoop, out.ActivityType)

	// other value reassigns
	out, err = svc.AssignToTask(ctx, 5, taskID, repoID, parentID,
		&types.TaskLabelAssignInput{LabelID: labelID, ValueID: ptr(value2ID)})
	require.NoError(t, err)
	assert.Equal(t, enum.LabelActivityReassign, out.ActivityType)
	require.NotNil(t, out.OldLabelValue)
	assert.Equal(t, value1ID, out.OldLabelValue.ID)
	assert.Equal(t, value2ID, *taskLabels.rows[[2]int64{taskID, labelID}].ValueID)

	// value of another label is rejected
	valueStore.On("FindByID", int64(5000)).Return(&types.LabelValue{ID: 5000, LabelID: 999}, nil)
	_, err = svc.AssignToTask(ctx, 5, taskID, repoID, parentID,
		&types.TaskLabelAssignInput{LabelID: labelID, ValueID: ptr(int64(5000))})
	require.Error(t, err)

	// unassign
	gotLabel, gotValue, activity, err := svc.UnassignFromTask(ctx, repoID, parentID, taskID, labelID)
	require.NoError(t, err)
	assert.Equal(t, lbl, gotLabel)
	require.NotNil(t, gotValue)
	assert.Equal(t, value2ID, gotValue.ID)
	assert.Equal(t, enum.LabelActivityUnassign, activity)
	assert.Empty(t, taskLabels.rows)

	// unassigning again is a no-op
	_, _, activity, err = svc.UnassignFromTask(ctx, repoID, parentID, taskID, labelID)
	require.NoError(t, err)
	assert.Equal(t, enum.LabelActivityNoop, activity)
}

func TestTaskLabelOutOfScope(t *testing.T) {
	t.Parallel()

	otherRepo := int64(2)
	labelStore := &mockstore.LabelStore{}
	labelStore.On("FindByID", int64(1)).Return(&types.Label{ID: 1, RepoID: &otherRepo}, nil)

	taskLabels := &fakeTaskLabels{rows: map[[2]int64]*types.TaskLabel{}}
	svc := label.New(nil, nil, labelStore, &mockstore.LabelValueStore{}, nil, nil, nil, refcache.SpaceFinder{}).
		WithTaskLabelStore(taskLabels)

	_, err := svc.AssignToTask(context.Background(), 5, 77, 1, 10, &types.TaskLabelAssignInput{LabelID: 1})
	require.Error(t, err)
	assert.Empty(t, taskLabels.rows)
}

func ptr[T any](v T) *T { return &v }
