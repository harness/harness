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

type RepoStore struct{ mock.Mock }

func (m *RepoStore) Find(_ context.Context, id int64) (*types.Repository, error) {
	args := m.Called(id)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) FindDeleted(_ context.Context, id int64, deleted *int64) (*types.Repository, error) {
	args := m.Called(id, deleted)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) FindActiveByUID(_ context.Context, parentSpaceID int64, uid string) (*types.Repository, error) {
	args := m.Called(parentSpaceID, uid)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) FindDeletedByUID(
	_ context.Context,
	parentSpaceID int64,
	uid string,
	deletedAt int64,
) (*types.Repository, error) {
	args := m.Called(parentSpaceID, uid, deletedAt)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) Create(_ context.Context, repo *types.Repository) error {
	return m.Called(repo).Error(0)
}

func (m *RepoStore) Update(_ context.Context, repo *types.Repository) error {
	return m.Called(repo).Error(0)
}

func (m *RepoStore) UpdateSize(_ context.Context, id int64, sizeInKiB, sizeLFSInKiB int64) error {
	return m.Called(id, sizeInKiB, sizeLFSInKiB).Error(0)
}

func (m *RepoStore) GetSize(_ context.Context, id int64) (int64, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *RepoStore) GetLFSSize(_ context.Context, id int64) (int64, error) {
	args := m.Called(id)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *RepoStore) UpdateOptLock(
	_ context.Context,
	repo *types.Repository,
	mutateFn func(repository *types.Repository) error,
) (*types.Repository, error) {
	args := m.Called(repo, mutateFn)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) SoftDelete(_ context.Context, repo *types.Repository, deletedAt int64) error {
	return m.Called(repo, deletedAt).Error(0)
}

func (m *RepoStore) Purge(_ context.Context, id int64, deletedAt *int64) error {
	return m.Called(id, deletedAt).Error(0)
}

func (m *RepoStore) Restore(
	_ context.Context,
	repo *types.Repository,
	newIdentifier *string,
	newParentID *int64,
) (*types.Repository, error) {
	args := m.Called(repo, newIdentifier, newParentID)
	if v, _ := args.Get(0).(*types.Repository); v != nil {
		return v, args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *RepoStore) Count(_ context.Context, parentID int64, opts *types.RepoFilter) (int64, error) {
	args := m.Called(parentID, opts)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}

func (m *RepoStore) CountByRootSpaces(_ context.Context) ([]types.RepositoryCount, error) {
	args := m.Called()
	v, _ := args.Get(0).([]types.RepositoryCount)
	return v, args.Error(1)
}

func (m *RepoStore) List(_ context.Context, parentID int64, opts *types.RepoFilter) ([]*types.Repository, error) {
	args := m.Called(parentID, opts)
	v, _ := args.Get(0).([]*types.Repository)
	return v, args.Error(1)
}

func (m *RepoStore) ListAll(_ context.Context, filter *types.RepoFilter) ([]*types.Repository, error) {
	args := m.Called(filter)
	v, _ := args.Get(0).([]*types.Repository)
	return v, args.Error(1)
}

func (m *RepoStore) ListSizeInfos(_ context.Context) ([]*types.RepositorySizeInfo, error) {
	args := m.Called()
	v, _ := args.Get(0).([]*types.RepositorySizeInfo)
	return v, args.Error(1)
}

func (m *RepoStore) UpdateNumForks(_ context.Context, repoID int64, delta int64) error {
	return m.Called(repoID, delta).Error(0)
}

func (m *RepoStore) ClearForkID(_ context.Context, repoUpstreamID int64) error {
	return m.Called(repoUpstreamID).Error(0)
}

func (m *RepoStore) UpdateParent(_ context.Context, currentParentID, newParentID int64) (int64, error) {
	args := m.Called(currentParentID, newParentID)
	if v, ok := args.Get(0).(int64); ok {
		return v, args.Error(1)
	}
	return 0, args.Error(1)
}
