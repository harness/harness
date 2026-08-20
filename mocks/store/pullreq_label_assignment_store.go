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

package store

import (
	"context"

	"github.com/harness/gitness/types"

	"github.com/stretchr/testify/mock"
)

type PullReqLabelAssignmentStore struct{ mock.Mock }

func (m *PullReqLabelAssignmentStore) Assign(_ context.Context, label *types.PullReqLabel) error {
	return m.Called(label).Error(0)
}

func (m *PullReqLabelAssignmentStore) Unassign(_ context.Context, pullreqID, labelID int64) (bool, error) {
	args := m.Called(pullreqID, labelID)
	return args.Bool(0), args.Error(1)
}

func (m *PullReqLabelAssignmentStore) ListAssigned(
	_ context.Context,
	pullreqID int64,
) (map[int64]*types.LabelAssignment, error) {
	args := m.Called(pullreqID)
	if v, _ := args.Get(0).(map[int64]*types.LabelAssignment); v != nil {
		err := args.Error(1)
		return v, err
	}
	err := args.Error(1)
	return nil, err
}

func (m *PullReqLabelAssignmentStore) FindByLabelID(
	_ context.Context,
	pullreqID, labelID int64,
) (*types.PullReqLabel, error) {
	args := m.Called(pullreqID, labelID)
	if v, _ := args.Get(0).(*types.PullReqLabel); v != nil {
		err := args.Error(1)
		return v, err
	}
	err := args.Error(1)
	return nil, err
}

func (m *PullReqLabelAssignmentStore) FindValueByLabelID(
	_ context.Context,
	pullreqID, labelID int64,
) (*types.LabelValue, error) {
	args := m.Called(pullreqID, labelID)
	if v, _ := args.Get(0).(*types.LabelValue); v != nil {
		err := args.Error(1)
		return v, err
	}
	err := args.Error(1)
	return nil, err
}

func (m *PullReqLabelAssignmentStore) CountPullreqAssignments(
	_ context.Context,
	labelIDs []int64,
) (map[int64]int64, error) {
	args := m.Called(labelIDs)
	if v, _ := args.Get(0).(map[int64]int64); v != nil {
		err := args.Error(1)
		return v, err
	}
	err := args.Error(1)
	return nil, err
}

func (m *PullReqLabelAssignmentStore) ListAssignedByPullreqIDs(
	_ context.Context,
	pullreqIDs []int64,
) (map[int64][]*types.LabelPullReqAssignmentInfo, error) {
	args := m.Called(pullreqIDs)
	if v, _ := args.Get(0).(map[int64][]*types.LabelPullReqAssignmentInfo); v != nil {
		err := args.Error(1)
		return v, err
	}
	err := args.Error(1)
	return nil, err
}
