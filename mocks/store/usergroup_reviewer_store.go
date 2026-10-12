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

type UserGroupReviewerStore struct{ mock.Mock }

func (m *UserGroupReviewerStore) Create(_ context.Context, reviewer *types.UserGroupReviewer) error {
	return m.Called(reviewer).Error(0)
}

func (m *UserGroupReviewerStore) Delete(_ context.Context, prID, principalID int64) error {
	return m.Called(prID, principalID).Error(0)
}

func (m *UserGroupReviewerStore) List(_ context.Context, prID int64) ([]*types.UserGroupReviewer, error) {
	args := m.Called(prID)
	v, _ := args.Get(0).([]*types.UserGroupReviewer)
	return v, args.Error(1)
}

func (m *UserGroupReviewerStore) Count(_ context.Context, prID int64) (int64, error) {
	args := m.Called(prID)
	v, _ := args.Get(0).(int64)
	return v, args.Error(1)
}

func (m *UserGroupReviewerStore) Find(
	_ context.Context,
	prID,
	userGroupReviewerID int64,
) (*types.UserGroupReviewer, error) {
	args := m.Called(prID, userGroupReviewerID)
	if v, _ := args.Get(0).(*types.UserGroupReviewer); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}
