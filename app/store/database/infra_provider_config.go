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

var _ store.InfraProviderConfigStore = (*infraProviderConfigStore)(nil)

// TODO Stubbed Impl
// NewGitspaceConfigStore returns a new GitspaceConfigStore.
func NewInfraProviderConfigStore(db *sqlx.DB) store.InfraProviderConfigStore {
	return &infraProviderConfigStore{
		db: db,
	}
}

type infraProviderConfigStore struct {
	db *sqlx.DB
}

func (i infraProviderConfigStore) Find(_ context.Context, _ int64) (*types.InfraProviderConfig, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderConfigStore) FindByIdentifier(_ context.Context, _ int64,
	_ string) (*types.InfraProviderConfig, error) {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderConfigStore) Create(_ context.Context, _ *types.InfraProviderConfig) error {
	// TODO implement me
	panic("implement me")
}

func (i infraProviderConfigStore) DeleteByIdentifier(_ context.Context, _ int64, _ string) error {
	// TODO implement me
	panic("implement me")
}
