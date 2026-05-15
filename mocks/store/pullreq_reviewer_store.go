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

type PullReqReviewerStore struct{ mock.Mock }

func (m *PullReqReviewerStore) Find(_ context.Context, prID, principalID int64) (*types.PullReqReviewer, error) {
	args := m.Called(prID, principalID)
	if v, _ := args.Get(0).(*types.PullReqReviewer); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqReviewerStore) Create(_ context.Context, reviewer *types.PullReqReviewer) error {
	return m.Called(reviewer).Error(0)
}

func (m *PullReqReviewerStore) Update(_ context.Context, reviewer *types.PullReqReviewer) error {
	return m.Called(reviewer).Error(0)
}

func (m *PullReqReviewerStore) Delete(_ context.Context, prID, principalID int64) error {
	return m.Called(prID, principalID).Error(0)
}

func (m *PullReqReviewerStore) List(_ context.Context, prID int64) ([]*types.PullReqReviewer, error) {
	args := m.Called(prID)
	v, _ := args.Get(0).([]*types.PullReqReviewer)
	return v, args.Error(1)
}
