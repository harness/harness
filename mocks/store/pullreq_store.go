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

type PullReqStore struct{ mock.Mock }

func (m *PullReqStore) Find(_ context.Context, id int64) (*types.PullReq, error) {
	args := m.Called(id)
	if len(args) == 1 {
		if rf, ok := args.Get(0).(func(int64) (*types.PullReq, error)); ok {
			return rf(id)
		}
	}
	var v *types.PullReq
	if args.Get(0) != nil {
		v, _ = args.Get(0).(*types.PullReq)
	}
	if len(args) > 1 {
		return v, args.Error(1)
	}
	return v, nil
}

func (m *PullReqStore) FindByNumberWithLock(_ context.Context, repoID, number int64) (*types.PullReq, error) {
	args := m.Called(repoID, number)
	if v, _ := args.Get(0).(*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqStore) FindByNumber(_ context.Context, repoID, number int64) (*types.PullReq, error) {
	args := m.Called(repoID, number)
	if v, _ := args.Get(0).(*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqStore) Create(_ context.Context, pullreq *types.PullReq) error {
	return m.Called(pullreq).Error(0)
}

func (m *PullReqStore) Update(_ context.Context, pr *types.PullReq) error {
	return m.Called(pr).Error(0)
}

func (m *PullReqStore) UpdateOptLock(
	_ context.Context,
	pr *types.PullReq,
	mutateFn func(pr *types.PullReq) error,
) (*types.PullReq, error) {
	args := m.Called(pr, mutateFn)
	if v, _ := args.Get(0).(*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqStore) UpdateMergeCheckMetadataOptLock(
	_ context.Context,
	pr *types.PullReq,
	mutateFn func(pr *types.PullReq) error,
) (*types.PullReq, error) {
	args := m.Called(pr, mutateFn)
	if v, _ := args.Get(0).(*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqStore) UpdateActivitySeq(_ context.Context, pr *types.PullReq) (*types.PullReq, error) {
	args := m.Called(pr)
	if v, _ := args.Get(0).(*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *PullReqStore) ResetMergeCheckStatus(_ context.Context, targetRepo int64, targetBranch string) error {
	return m.Called(targetRepo, targetBranch).Error(0)
}

func (m *PullReqStore) Delete(_ context.Context, id int64) error {
	return m.Called(id).Error(0)
}

func (m *PullReqStore) Count(_ context.Context, opts *types.PullReqFilter) (int64, error) {
	args := m.Called(opts)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *PullReqStore) List(_ context.Context, opts *types.PullReqFilter) ([]*types.PullReq, error) {
	args := m.Called(opts)
	v, _ := args.Get(0).([]*types.PullReq)
	return v, args.Error(1)
}

func (m *PullReqStore) Stream(
	_ context.Context,
	opts *types.PullReqFilter,
) (<-chan *types.PullReq, <-chan error) {
	args := m.Called(opts)
	items, _ := args.Get(0).(<-chan *types.PullReq)
	errs, _ := args.Get(1).(<-chan error)
	return items, errs
}

func (m *PullReqStore) ListOpenByBranchName(
	_ context.Context,
	repoID int64,
	branchNames []string,
) (map[string][]*types.PullReq, error) {
	args := m.Called(repoID, branchNames)
	if v, _ := args.Get(0).(map[string][]*types.PullReq); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}
