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

type PullReqActivityStore struct{ mock.Mock }

func (m *PullReqActivityStore) Find(_ context.Context, id int64) (*types.PullReqActivity, error) {
	args := m.Called(id)
	if v, _ := args.Get(0).(*types.PullReqActivity); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqActivityStore) Create(_ context.Context, act *types.PullReqActivity) error {
	return m.Called(act).Error(0)
}

func (m *PullReqActivityStore) CreateWithPayload(
	_ context.Context,
	pr *types.PullReq,
	principalID int64,
	payload types.PullReqActivityPayload,
	metadata *types.PullReqActivityMetadata,
) (*types.PullReqActivity, error) {
	args := m.Called(pr, principalID, payload, metadata)
	if v, _ := args.Get(0).(*types.PullReqActivity); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqActivityStore) Update(_ context.Context, act *types.PullReqActivity) error {
	return m.Called(act).Error(0)
}

func (m *PullReqActivityStore) UpdateOptLock(
	_ context.Context,
	act *types.PullReqActivity,
	mutateFn func(act *types.PullReqActivity) error,
) (*types.PullReqActivity, error) {
	args := m.Called(act, mutateFn)
	if v, _ := args.Get(0).(*types.PullReqActivity); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqActivityStore) Count(
	_ context.Context,
	prID int64,
	opts *types.PullReqActivityFilter,
) (int64, error) {
	args := m.Called(prID, opts)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *PullReqActivityStore) CountUnresolved(_ context.Context, prID int64) (int, error) {
	args := m.Called(prID)
	if v, ok := args.Get(0).(int); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *PullReqActivityStore) List(
	_ context.Context,
	prID int64,
	opts *types.PullReqActivityFilter,
) ([]*types.PullReqActivity, error) {
	args := m.Called(prID, opts)
	v, _ := args.Get(0).([]*types.PullReqActivity)
	return v, args.Error(1)
}

func (m *PullReqActivityStore) ListAuthorIDs(_ context.Context, prID int64, order int64) ([]int64, error) {
	args := m.Called(prID, order)
	v, _ := args.Get(0).([]int64)
	return v, args.Error(1)
}
