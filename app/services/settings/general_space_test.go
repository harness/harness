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

package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	appstore "github.com/harness/gitness/app/store"
	basestore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestSpaceUpdateGeneralSettingsDeletesDefaultBranch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemorySettingsStore()
	service := NewService(store, refcache.SpaceFinder{})

	require.NoError(t, service.Set(ctx, enum.SettingsScopeSpace, 1, DefaultBranchKey, ptrString("develop")))

	old, out, err := SpaceUpdateGeneralSettings(ctx, service, 1, &GeneralSettingsSpace{
		DefaultBranch: ptrString(""),
	})
	require.NoError(t, err)
	require.NotNil(t, old)
	require.NotNil(t, out)
	require.NotNil(t, old.DefaultBranch)
	require.Equal(t, "develop", *old.DefaultBranch)
	require.NotNil(t, out.DefaultBranch)
	require.Equal(t, DefaultBranch, *out.DefaultBranch)

	_, err = store.Find(ctx, enum.SettingsScopeSpace, 1, string(DefaultBranchKey))
	require.ErrorIs(t, err, basestore.ErrResourceNotFound)
}

func TestSpaceUpdateGeneralSettingsRejectsInvalidBranch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemorySettingsStore()
	service := NewService(store, refcache.SpaceFinder{})

	_, _, err := SpaceUpdateGeneralSettings(ctx, service, 1, &GeneralSettingsSpace{
		DefaultBranch: ptrString("bad branch"),
	})
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid default branch name")

	_, findErr := store.Find(ctx, enum.SettingsScopeSpace, 1, string(DefaultBranchKey))
	require.ErrorIs(t, findErr, basestore.ErrResourceNotFound)
}

func TestSpaceGetDefaultBranchRecursiveReturnsConfiguredBranch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := newInMemorySettingsStore()
	service := NewService(store, refcache.SpaceFinder{})

	require.NoError(t, service.Set(ctx, enum.SettingsScopeSpace, 1, DefaultBranchKey, ptrString("release/1.0")))

	branch, err := service.SpaceGetDefaultBranchRecursive(ctx, 1, 0)
	require.NoError(t, err)
	require.Equal(t, "release/1.0", branch)
}

func TestSpaceGetDefaultBranchRecursiveFallsBackToGlobalDefault(t *testing.T) {
	t.Parallel()

	service := NewService(newInMemorySettingsStore(), refcache.SpaceFinder{})

	branch, err := service.SpaceGetDefaultBranchRecursive(context.Background(), 1, 0)
	require.NoError(t, err)
	require.Equal(t, DefaultBranch, branch)
}

func ptrString(value string) *string {
	return &value
}

type inMemorySettingsStore struct {
	values map[string]json.RawMessage
}

func newInMemorySettingsStore() *inMemorySettingsStore {
	return &inMemorySettingsStore{values: map[string]json.RawMessage{}}
}

func (s *inMemorySettingsStore) Find(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) (json.RawMessage, error) {
	value, ok := s.values[s.makeKey(scope, scopeID, key)]
	if !ok {
		return nil, basestore.ErrResourceNotFound
	}

	return value, nil
}

func (s *inMemorySettingsStore) FindMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) (map[string]json.RawMessage, error) {
	out := make(map[string]json.RawMessage, len(keys))
	for _, key := range keys {
		value, ok := s.values[s.makeKey(scope, scopeID, key)]
		if ok {
			out[key] = value
		}
	}

	return out, nil
}

func (s *inMemorySettingsStore) Upsert(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
	value json.RawMessage,
) error {
	s.values[s.makeKey(scope, scopeID, key)] = value
	return nil
}

func (s *inMemorySettingsStore) Delete(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	key string,
) error {
	delete(s.values, s.makeKey(scope, scopeID, key))
	return nil
}

func (s *inMemorySettingsStore) DeleteMany(
	_ context.Context,
	scope enum.SettingsScope,
	scopeID int64,
	keys ...string,
) error {
	for _, key := range keys {
		delete(s.values, s.makeKey(scope, scopeID, key))
	}
	return nil
}

func (s *inMemorySettingsStore) makeKey(scope enum.SettingsScope, scopeID int64, key string) string {
	return fmt.Sprintf("%s:%d:%s", scope, scopeID, key)
}

var _ appstore.SettingsStore = (*inMemorySettingsStore)(nil)
