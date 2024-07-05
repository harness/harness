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

package database

import (
	"context"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/types"

	"github.com/jmoiron/sqlx"
)

var _ store.InfraProviderResourceStore = (*infraProviderResourceStore)(nil)

// TODO Stubbed Impl
// NewGitspaceConfigStore returns a new GitspaceConfigStore.
func NewInfraProviderResourceStore(db *sqlx.DB) store.InfraProviderResourceStore {
	return &infraProviderResourceStore{
		db: db,
	}
}

type infraProviderResourceStore struct {
	db *sqlx.DB
}

func (i infraProviderResourceStore) Find(_ context.Context, _ int64) (*types.InfraProviderResource, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderResourceStore) FindByIdentifier(_ context.Context, _ int64,
	_ string) (*types.InfraProviderResource, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderResourceStore) Create(_ context.Context, _ int64, _ *types.InfraProviderResource) error {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderResourceStore) List(_ context.Context, _ int64,
	_ types.ListQueryFilter) ([]*types.InfraProviderResource, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderResourceStore) ListAll(_ context.Context, _ int64,
	_ types.ListQueryFilter) ([]*types.InfraProviderResource, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderResourceStore) DeleteByIdentifier(_ context.Context, _ int64, _ string) error {
	// TODO implement me
	panic("implement me")
}
