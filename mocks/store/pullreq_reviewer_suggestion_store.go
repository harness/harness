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

type PullReqReviewerSuggestionStore struct{ mock.Mock }

func (m *PullReqReviewerSuggestionStore) Find(
	_ context.Context,
	prID,
	principalID int64,
) (*types.PullReqReviewerSuggestion, error) {
	args := m.Called(prID, principalID)
	if v, _ := args.Get(0).(*types.PullReqReviewerSuggestion); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqReviewerSuggestionStore) List(
	_ context.Context,
	prID int64,
	pagination types.Pagination,
) ([]*types.PullReqReviewerSuggestion, error) {
	args := m.Called(prID, pagination)
	v, _ := args.Get(0).([]*types.PullReqReviewerSuggestion)
	return v, args.Error(1)
}

func (m *PullReqReviewerSuggestionStore) Count(_ context.Context, prID int64) (int64, error) {
	args := m.Called(prID)
	count, ok := args.Get(0).(int64)
	if !ok {
		return 0, args.Error(1)
	}
	return count, args.Error(1)
}

func (m *PullReqReviewerSuggestionStore) CreateMany(
	_ context.Context,
	suggestions []*types.PullReqReviewerSuggestion,
) error {
	return m.Called(suggestions).Error(0)
}

func (m *PullReqReviewerSuggestionStore) Delete(_ context.Context, prID, principalID int64) error {
	return m.Called(prID, principalID).Error(0)
}
