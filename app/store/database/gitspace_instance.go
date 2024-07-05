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

var _ store.GitspaceInstanceStore = (*gitspaceInstanceStore)(nil)

// TODO Stubbed Impl
// NewGitspaceInstanceStore returns a new GitspaceInstanceStore.
func NewGitspaceInstanceStore(db *sqlx.DB) store.GitspaceInstanceStore {
	return &gitspaceInstanceStore{
		db: db,
	}
}

type gitspaceInstanceStore struct {
	db *sqlx.DB
}

func (g *gitspaceInstanceStore) Find(_ context.Context, _ int64) (*types.GitspaceInstance, error) {
	// TODO implement me
	panic("implement me")
}

func (g *gitspaceInstanceStore) FindLatestByGitspaceConfigID(
	_ context.Context,
	_ int64,
	_ int64) (*types.GitspaceInstance, error) {
	// TODO implement me
	panic("implement me")
}

func (g *gitspaceInstanceStore) Create(_ context.Context, _ *types.GitspaceInstance) error {
	// TODO implement me
	panic("implement me")
}

func (g *gitspaceInstanceStore) Update(_ context.Context, _ *types.GitspaceInstance) (*types.GitspaceInstance, error) {
	// TODO implement me
	panic("implement me")
}

func (g *gitspaceInstanceStore) List(_ context.Context, _ *types.GitspaceFilter) ([]*types.GitspaceInstance, error) {
	// TODO implement me
	panic("implement me")
}

func (g *gitspaceInstanceStore) Delete(_ context.Context, _ int64) error {
	// TODO implement me
	panic("implement me")
}
