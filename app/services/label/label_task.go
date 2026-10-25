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

package label

import (
	"context"
	"fmt"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/errors"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"
)

// AssignToTaskOut is the outcome of assigning a label to a task.
type AssignToTaskOut struct {
	Label         *types.Label
	TaskLabel     *types.TaskLabel
	OldLabelValue *types.LabelValue
	NewLabelValue *types.LabelValue
	ActivityType  enum.PullReqLabelActivityType
}

// ToLabelTaskAssignmentInfo converts the outcome to the label assignment info of the task.
func (out *AssignToTaskOut) ToLabelTaskAssignmentInfo() *types.LabelTaskAssignmentInfo {
	info := &types.LabelTaskAssignmentInfo{
		TaskID:     out.TaskLabel.TaskID,
		LabelID:    out.Label.ID,
		LabelKey:   out.Label.Key,
		LabelColor: out.Label.Color,
		LabelScope: out.Label.Scope,
		ValueCount: out.Label.ValueCount,
	}
	if out.NewLabelValue != nil {
		info.ValueID = &out.NewLabelValue.ID
		info.Value = &out.NewLabelValue.Value
		info.ValueColor = &out.NewLabelValue.Color
	}
	return info
}

// WithTaskLabelStore sets the store of the task label assignments, which is required by the task label methods.
func (s *Service) WithTaskLabelStore(taskLabelStore store.TaskLabelStore) *Service {
	s.taskLabelStore = taskLabelStore
	return s
}

// AssignToTask assigns a label (with an optional value) to a task of the repository.
// Assigning the same label and value again is a no-op, assigning another value reassigns the label.
func (s *Service) AssignToTask(
	ctx context.Context,
	principalID int64,
	taskID int64,
	repoID int64,
	repoParentID int64,
	in *types.TaskLabelAssignInput,
) (*AssignToTaskOut, error) {
	label, err := s.labelStore.FindByID(ctx, in.LabelID)
	if err != nil {
		return nil, fmt.Errorf("failed to find label by id: %w", err)
	}

	if err = s.checkLabelInScope(ctx, repoParentID, repoID, label); err != nil {
		return nil, err
	}

	oldTaskLabel, err := s.taskLabelStore.FindByLabelID(ctx, taskID, label.ID)
	if err != nil && !errors.Is(err, basestore.ErrResourceNotFound) {
		return nil, fmt.Errorf("failed to find task label: %w", err)
	}

	var oldLabelValue *types.LabelValue
	if oldTaskLabel != nil && oldTaskLabel.ValueID != nil {
		oldLabelValue, err = s.labelValueStore.FindByID(ctx, *oldTaskLabel.ValueID)
		if err != nil {
			return nil, fmt.Errorf("failed to find label value by id: %w", err)
		}
	}

	var newLabelValue *types.LabelValue
	switch {
	case in.ValueID != nil:
		newLabelValue, err = s.labelValueStore.FindByID(ctx, *in.ValueID)
		if err != nil {
			return nil, fmt.Errorf("failed to find label value by id: %w", err)
		}
		if label.ID != newLabelValue.LabelID {
			return nil, errors.InvalidArgument("label value is not associated with label")
		}
	case in.Value != "":
		newLabelValue, err = s.getOrDefineValue(ctx, principalID, label, in.Value)
		if err != nil {
			return nil, err
		}
	}

	if oldTaskLabel != nil && sameValueID(oldTaskLabel.ValueID, newLabelValue) {
		return &AssignToTaskOut{
			Label:         label,
			TaskLabel:     oldTaskLabel,
			OldLabelValue: oldLabelValue,
			ActivityType:  enum.LabelActivityNoop,
		}, nil
	}

	now := time.Now().UnixMilli()
	newTaskLabel := &types.TaskLabel{
		TaskID:    taskID,
		LabelID:   label.ID,
		Created:   now,
		Updated:   now,
		CreatedBy: principalID,
		UpdatedBy: principalID,
	}
	if newLabelValue != nil {
		newTaskLabel.ValueID = &newLabelValue.ID
	}

	if err = s.taskLabelStore.Assign(ctx, newTaskLabel); err != nil {
		return nil, fmt.Errorf("failed to assign label to task: %w", err)
	}

	activityType := enum.LabelActivityAssign
	if oldTaskLabel != nil {
		activityType = enum.LabelActivityReassign
	}

	return &AssignToTaskOut{
		Label:         label,
		TaskLabel:     newTaskLabel,
		OldLabelValue: oldLabelValue,
		NewLabelValue: newLabelValue,
		ActivityType:  activityType,
	}, nil
}

func sameValueID(old *int64, value *types.LabelValue) bool {
	if old == nil || value == nil {
		return old == nil && value == nil
	}
	return *old == value.ID
}

// UnassignFromTask removes a label from a task of the repository. If the label wasn't assigned
// the activity type is noop.
func (s *Service) UnassignFromTask(
	ctx context.Context,
	repoID, repoParentID, taskID, labelID int64,
) (*types.Label, *types.LabelValue, enum.PullReqLabelActivityType, error) {
	label, err := s.labelStore.FindByID(ctx, labelID)
	if err != nil {
		return nil, nil, enum.LabelActivityNoop, fmt.Errorf("failed to find label by id: %w", err)
	}

	if err = s.checkLabelInScope(ctx, repoParentID, repoID, label); err != nil {
		return nil, nil, enum.LabelActivityNoop, err
	}

	var value *types.LabelValue
	taskLabel, err := s.taskLabelStore.FindByLabelID(ctx, taskID, labelID)
	if err != nil && !errors.Is(err, basestore.ErrResourceNotFound) {
		return nil, nil, enum.LabelActivityNoop, fmt.Errorf("failed to find task label: %w", err)
	}
	if taskLabel != nil && taskLabel.ValueID != nil {
		value, err = s.labelValueStore.FindByID(ctx, *taskLabel.ValueID)
		if err != nil {
			return nil, nil, enum.LabelActivityNoop, fmt.Errorf("failed to find label value by id: %w", err)
		}
	}

	deleted, err := s.taskLabelStore.Unassign(ctx, taskID, labelID)
	if err != nil {
		return nil, nil, enum.LabelActivityNoop, fmt.Errorf("failed to unassign label from task: %w", err)
	}
	if !deleted {
		return label, value, enum.LabelActivityNoop, nil
	}

	return label, value, enum.LabelActivityUnassign, nil
}

// ListTaskLabels lists the labels of a task grouped by scope. With filter.Assignable it lists all labels
// that can be assigned to tasks of the repository, marking the assigned ones.
func (s *Service) ListTaskLabels(
	ctx context.Context,
	repo *types.RepositoryCore,
	spaceID int64,
	taskID int64,
	filter *types.AssignableLabelFilter,
) (*types.ScopesLabels, int64, error) {
	assignments, err := s.taskLabelStore.ListAssigned(ctx, taskID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list labels assigned to task: %w", err)
	}

	return s.listWithAssignments(ctx, repo, spaceID, assignments, filter)
}

// ListTaskLabelInfos returns the labels assigned to the tasks, keyed by task id.
func (s *Service) ListTaskLabelInfos(
	ctx context.Context,
	taskIDs []int64,
) (map[int64][]*types.LabelTaskAssignmentInfo, error) {
	infos, err := s.taskLabelStore.ListAssignedByTaskIDs(ctx, taskIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to list labels assigned to tasks: %w", err)
	}
	return infos, nil
}
