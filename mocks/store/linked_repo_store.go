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

type LinkedRepoStore struct{ mock.Mock }

func (m *LinkedRepoStore) Find(_ context.Context, repoID int64) (*types.LinkedRepo, error) {
	args := m.Called(repoID)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(int64) (*types.LinkedRepo, error)); ok {
			return rf(repoID)
		}
	}
	var v *types.LinkedRepo
	if args.Get(0) != nil {
		v, _ = args.Get(0).(*types.LinkedRepo)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *LinkedRepoStore) Create(_ context.Context, v *types.LinkedRepo) error {
	return m.Called(v).Error(0)
}

func (m *LinkedRepoStore) Update(_ context.Context, linked *types.LinkedRepo) error {
	return m.Called(linked).Error(0)
}

func (m *LinkedRepoStore) UpdateOptLock(
	_ context.Context,
	r *types.LinkedRepo,
	mutateFn func(*types.LinkedRepo) error,
) (*types.LinkedRepo, error) {
	args := m.Called(r, mutateFn)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(*types.LinkedRepo, func(*types.LinkedRepo) error) (*types.LinkedRepo, error)); ok {
			return rf(r, mutateFn)
		}
	}
	var v *types.LinkedRepo
	if args.Get(0) != nil {
		v, _ = args.Get(0).(*types.LinkedRepo)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *LinkedRepoStore) List(_ context.Context, limit int) ([]types.LinkedRepo, error) {
	args := m.Called(limit)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(int) ([]types.LinkedRepo, error)); ok {
			return rf(limit)
		}
	}
	var v []types.LinkedRepo
	if args.Get(0) != nil {
		v, _ = args.Get(0).([]types.LinkedRepo)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *LinkedRepoStore) ListByProviderID(
	_ context.Context,
	accountID, provider, providerID string,
	pagination types.Pagination,
) ([]types.LinkedRepo, error) {
	args := m.Called(accountID, provider, providerID, pagination)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(string, string, string, types.Pagination) ([]types.LinkedRepo, error)); ok {
			return rf(accountID, provider, providerID, pagination)
		}
	}
	var v []types.LinkedRepo
	if args.Get(0) != nil {
		v, _ = args.Get(0).([]types.LinkedRepo)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}
