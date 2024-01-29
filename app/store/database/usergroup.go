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

	gitnessAppStore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
)

var _ gitnessAppStore.UserGroupStore = (*UserGroupStore)(nil)

type UserGroupStore struct {
}

// Find Dummy Method: to be implemented later.
func (s *UserGroupStore) FindByIdentifier(_ context.Context, _ int64, _ string) (*types.UserGroup, error) {
	//nolint: nilnil
	return nil, store.ErrResourceNotFound
}
