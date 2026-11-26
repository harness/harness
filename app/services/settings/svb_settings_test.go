// Copyright 2026 Harness, Inc.
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
	"errors"
	"fmt"
	"testing"

	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

var errSvbBoom = errors.New("boom")

type svbKey struct {
	scope   enum.SettingsScope
	scopeID int64
	key     string
}

type svbSettingsStore struct {
	store.SettingsStore

	data map[svbKey]json.RawMessage
	// failOn makes the named store method fail (Find, FindMany, Upsert, Delete, DeleteMany).
	failOn map[string]bool
}

func svbNewStore() *svbSettingsStore {
	return &svbSettingsStore{data: map[svbKey]json.RawMessage{}, failOn: map[string]bool{}}
}

func (f *svbSettingsStore) Find(
	_ context.Context, scope enum.SettingsScope, scopeID int64, key string,
) (json.RawMessage, error) {
	if f.failOn["Find"] {
		return nil, errSvbBoom
	}
	raw, ok := f.data[svbKey{scope, scopeID, key}]
	if !ok {
		return nil, gitnessstore.ErrResourceNotFound
	}
	return raw, nil
}

func (f *svbSettingsStore) FindMany(
	_ context.Context, scope enum.SettingsScope, scopeID int64, keys ...string,
) (map[string]json.RawMessage, error) {
	if f.failOn["FindMany"] {
		return nil, errSvbBoom
	}
	out := map[string]json.RawMessage{}
	for _, k := range keys {
		if raw, ok := f.data[svbKey{scope, scopeID, k}]; ok {
			out[k] = raw
		}
	}
	return out, nil
}

func (f *svbSettingsStore) Upsert(
	_ context.Context, scope enum.SettingsScope, scopeID int64, key string, value json.RawMessage,
) error {
	if f.failOn["Upsert"] {
		return errSvbBoom
	}
	f.data[svbKey{scope, scopeID, key}] = value
	return nil
}

func (f *svbSettingsStore) Delete(_ context.Context, scope enum.SettingsScope, scopeID int64, key string) error {
	if f.failOn["Delete"] {
		return errSvbBoom
	}
	delete(f.data, svbKey{scope, scopeID, key})
	return nil
}

func (f *svbSettingsStore) DeleteMany(
	_ context.Context, scope enum.SettingsScope, scopeID int64, keys ...string,
) error {
	if f.failOn["DeleteMany"] {
		return errSvbBoom
	}
	for _, k := range keys {
		delete(f.data, svbKey{scope, scopeID, k})
	}
	return nil
}

type svbSpaceIDCache struct {
	store.SpaceIDCache

	spaces map[int64]*types.SpaceCore
}

func (f *svbSpaceIDCache) Get(_ context.Context, id int64) (*types.SpaceCore, error) {
	s, ok := f.spaces[id]
	if !ok {
		return nil, fmt.Errorf("space %d: %w", id, gitnessstore.ErrResourceNotFound)
	}
	return s, nil
}

func svbNewService(st *svbSettingsStore) *Service {
	ids := &svbSpaceIDCache{spaces: map[int64]*types.SpaceCore{
		1: {ID: 1, ParentID: 0},
		2: {ID: 2, ParentID: 1},
		3: {ID: 3, ParentID: 2},
	}}
	return NewService(st, refcache.NewSpaceFinder(ids, nil, nil, cache.Evictor[*types.SpaceCore]{}))
}

func TestSvbSetGetDelete(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	require.NoError(t, svc.Set(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, int64(42)))
	var got int64
	ok, err := svc.Get(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, &got)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, int64(42), got)

	ok, err = svc.Get(ctx, enum.SettingsScopeRepo, 1, KeyGitLFSEnabled, &got)
	require.NoError(t, err)
	require.False(t, ok)

	require.NoError(t, svc.Delete(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit))
	ok, err = svc.Get(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, &got)
	require.NoError(t, err)
	require.False(t, ok)

	// Unmarshalable values and store failures are reported.
	require.ErrorContains(t, svc.Set(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, make(chan int)), "marshal")

	st.failOn["Upsert"] = true
	require.ErrorIs(t, svc.Set(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, 1), errSvbBoom)
	require.ErrorContains(t, svc.SetMany(ctx, enum.SettingsScopeRepo, 1, KeyValue{Key: KeyFileSizeLimit, Value: 1}),
		"failed to set setting for key")
	st.failOn["Upsert"] = false

	st.data[svbKey{enum.SettingsScopeRepo, 1, string(KeyFileSizeLimit)}] = json.RawMessage(`"not-a-number"`)
	_, err = svc.Get(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, &got)
	require.ErrorContains(t, err, "unmarshal")

	st.failOn["Find"] = true
	_, err = svc.Get(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit, &got)
	require.ErrorIs(t, err, errSvbBoom)

	st.failOn["Delete"] = true
	require.ErrorIs(t, svc.Delete(ctx, enum.SettingsScopeRepo, 1, KeyFileSizeLimit), errSvbBoom)
}

func TestSvbDeleteMany(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	require.NoError(t, svc.SpaceSetMany(ctx, 1,
		KeyValue{Key: KeyFileSizeLimit, Value: 1}, KeyValue{Key: KeyGitLFSEnabled, Value: true}))
	require.Len(t, st.data, 2)

	require.NoError(t, svc.SpaceDeleteMany(ctx, 1, KeyFileSizeLimit, KeyGitLFSEnabled))
	require.Empty(t, st.data)

	st.failOn["DeleteMany"] = true
	require.ErrorIs(t, svc.SpaceDeleteMany(ctx, 1, KeyFileSizeLimit), errSvbBoom)
}

func TestSvbMap(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	require.NoError(t, svc.Map(ctx, enum.SettingsScopeRepo, 1), "no handlers is a no-op")

	require.NoError(t, svc.RepoSetMany(ctx, 1, KeyValue{Key: KeyFileSizeLimit, Value: int64(7)}))
	var limit int64
	var lfs bool
	require.NoError(t, svc.RepoMap(ctx, 1, Mapping(KeyFileSizeLimit, &limit), Mapping(KeyGitLFSEnabled, &lfs)))
	require.Equal(t, int64(7), limit)
	require.False(t, lfs, "missing optional settings are left untouched")

	err := svc.RepoMap(ctx, 1, MappingRequired(KeyGitLFSEnabled, &lfs))
	require.ErrorContains(t, err, "required setting")

	st.data[svbKey{enum.SettingsScopeRepo, 1, string(KeyFileSizeLimit)}] = json.RawMessage(`"bad"`)
	err = svc.RepoMap(ctx, 1, Mapping(KeyFileSizeLimit, &limit))
	require.ErrorContains(t, err, "failed to handle value for setting")

	st.failOn["FindMany"] = true
	require.ErrorIs(t, svc.RepoMap(ctx, 1, Mapping(KeyFileSizeLimit, &limit)), errSvbBoom)

	require.Panics(t, func() { Mapping[int64](KeyFileSizeLimit, nil) })
	require.Panics(t, func() { MappingRequired[int64](KeyFileSizeLimit, nil) })

	h := MappingRequired(KeyFileSizeLimit, &limit)
	require.Equal(t, KeyFileSizeLimit, h.Key())
	require.True(t, h.Required())
	require.False(t, Mapping(KeyFileSizeLimit, &limit).Required())
}

func TestSvbRepoAndSystemHelpers(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	// Defaults are returned when the setting doesn't exist.
	v, err := RepoGet(ctx, svc, 1, KeyFileSizeLimit, int64(99))
	require.NoError(t, err)
	require.Equal(t, int64(99), v)

	_, err = RepoGetRequired[int64](ctx, svc, 1, KeyFileSizeLimit)
	require.ErrorContains(t, err, "not found")

	require.NoError(t, svc.RepoSet(ctx, 1, KeyFileSizeLimit, int64(5)))
	v, err = RepoGet(ctx, svc, 1, KeyFileSizeLimit, int64(99))
	require.NoError(t, err)
	require.Equal(t, int64(5), v)
	v, err = RepoGetRequired[int64](ctx, svc, 1, KeyFileSizeLimit)
	require.NoError(t, err)
	require.Equal(t, int64(5), v)

	var raw bool
	ok, err := svc.RepoGet(ctx, 1, KeyGitLFSEnabled, &raw)
	require.NoError(t, err)
	require.False(t, ok)

	sv, err := SystemGet(ctx, svc, KeyInstallID, "dflt")
	require.NoError(t, err)
	require.Equal(t, "dflt", sv)
	_, err = SystemGetRequired[string](ctx, svc, KeyInstallID)
	require.ErrorContains(t, err, "not found")

	require.NoError(t, svc.SystemSet(ctx, KeyInstallID, "abc"))
	sv, err = SystemGet(ctx, svc, KeyInstallID, "dflt")
	require.NoError(t, err)
	require.Equal(t, "abc", sv)
	sv, err = SystemGetRequired[string](ctx, svc, KeyInstallID)
	require.NoError(t, err)
	require.Equal(t, "abc", sv)

	require.NoError(t, svc.SystemSetMany(ctx, KeyValue{Key: KeyFileSizeLimit, Value: int64(3)}))
	var limit int64
	require.NoError(t, svc.SystemMap(ctx, Mapping(KeyFileSizeLimit, &limit)))
	require.Equal(t, int64(3), limit)
	ok, err = svc.SystemGet(ctx, KeyFileSizeLimit, &limit)
	require.NoError(t, err)
	require.True(t, ok)

	st.failOn["Find"] = true
	_, err = RepoGet(ctx, svc, 1, KeyFileSizeLimit, int64(1))
	require.ErrorIs(t, err, errSvbBoom)
	_, err = RepoGetRequired[int64](ctx, svc, 1, KeyFileSizeLimit)
	require.ErrorIs(t, err, errSvbBoom)
	_, err = SystemGet(ctx, svc, KeyInstallID, "")
	require.ErrorIs(t, err, errSvbBoom)
	_, err = SystemGetRequired[string](ctx, svc, KeyInstallID)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbGeneralAndSecuritySettings(t *testing.T) {
	general := GetDefaultGeneralSettings()
	require.Equal(t, DefaultFileSizeLimit, *general.FileSizeLimit)
	require.Equal(t, DefaultGitLFSEnabled, *general.GitLFSEnabled)
	require.Equal(t, DefaultAutoMergeEnabled, *general.AutoMergeEnabled)
	require.Len(t, GetGeneralSettingsMappings(general), 3)
	require.Len(t, GetGeneralSettingsAsKeyValues(general), 3)
	require.Empty(t, GetGeneralSettingsAsKeyValues(&GeneralSettings{}))

	security := GetDefaultSecuritySettings()
	require.Equal(t, DefaultSecretScanningEnabled, *security.SecretScanningEnabled)
	require.Equal(t, DefaultPrincipalCommitterMatch, *security.PrincipalCommitterMatch)
	require.Len(t, GetSecuritySettingsMappings(security), 2)
	require.Len(t, GetSecuritySettingsAsKeyValues(security), 2)
	require.Empty(t, GetSecuritySettingsAsKeyValues(&SecuritySettings{}))
}

func TestSvbMapWithDefaultsAndUpdate(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	require.NoError(t, svc.RepoSet(ctx, 1, KeyFileSizeLimit, int64(10)))

	got, err := RepoMapWithDefaults(ctx, svc, 1, GetDefaultGeneralSettings, GetGeneralSettingsMappings)
	require.NoError(t, err)
	require.Equal(t, int64(10), *got.FileSizeLimit)
	require.Equal(t, DefaultGitLFSEnabled, *got.GitLFSEnabled)

	oldS, newS, err := RepoUpdateWithDefaults(ctx, svc, 1, GetDefaultGeneralSettings, GetGeneralSettingsMappings,
		KeyValue{Key: KeyFileSizeLimit, Value: int64(20)})
	require.NoError(t, err)
	require.Equal(t, int64(10), *oldS.FileSizeLimit)
	require.Equal(t, int64(20), *newS.FileSizeLimit)

	spaceGot, err := SpaceMapWithDefaults(ctx, svc, 2, GetDefaultGeneralSettingsSpace, getGeneralSettingsMappingsSpace)
	require.NoError(t, err)
	require.Equal(t, DefaultBranch, *spaceGot.DefaultBranch)

	oldSp, newSp, err := SpaceUpdateWithDefaults(
		ctx, svc, 2, GetDefaultGeneralSettingsSpace, getGeneralSettingsMappingsSpace,
		KeyValue{Key: DefaultBranchKey, Value: "develop"})
	require.NoError(t, err)
	require.Equal(t, DefaultBranch, *oldSp.DefaultBranch)
	require.Equal(t, "develop", *newSp.DefaultBranch)

	// A failing store surfaces at the matching stage.
	st.failOn["FindMany"] = true
	_, err = RepoMapWithDefaults(ctx, svc, 1, GetDefaultGeneralSettings, GetGeneralSettingsMappings)
	require.ErrorContains(t, err, "failed to map settings")
	_, _, err = RepoUpdateWithDefaults(ctx, svc, 1, GetDefaultGeneralSettings, GetGeneralSettingsMappings)
	require.ErrorContains(t, err, "(old)")
	_, err = SpaceMapWithDefaults(ctx, svc, 2, GetDefaultGeneralSettingsSpace, getGeneralSettingsMappingsSpace)
	require.ErrorContains(t, err, "failed to map settings")
	_, _, err = SpaceUpdateWithDefaults(ctx, svc, 2, GetDefaultGeneralSettingsSpace, getGeneralSettingsMappingsSpace)
	require.ErrorContains(t, err, "(old)")

	st.failOn["FindMany"] = false
	st.failOn["Upsert"] = true
	_, _, err = RepoUpdateWithDefaults(ctx, svc, 1, GetDefaultGeneralSettings, GetGeneralSettingsMappings,
		KeyValue{Key: KeyFileSizeLimit, Value: int64(1)})
	require.ErrorContains(t, err, "failed to set settings")
	_, _, err = SpaceUpdateWithDefaults(ctx, svc, 2, GetDefaultGeneralSettingsSpace, getGeneralSettingsMappingsSpace,
		KeyValue{Key: DefaultBranchKey, Value: "x"})
	require.ErrorContains(t, err, "failed to set settings")
}

func TestSvbSpaceGetInherited(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	// Only the root space (1) configures the value; space 3 inherits it via 2.
	require.NoError(t, svc.Set(ctx, enum.SettingsScopeSpace, 1, DefaultBranchKey, "trunk"))

	var branch string
	ok, err := svc.SpaceGet(ctx, 3, DefaultBranchKey, false, &branch)
	require.NoError(t, err)
	require.False(t, ok)

	ok, err = svc.SpaceGet(ctx, 3, DefaultBranchKey, true, &branch)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, "trunk", branch)

	got, err := svc.SpaceGetDefaultBranch(ctx, 3, true)
	require.NoError(t, err)
	require.Equal(t, "trunk", got)
	got, err = svc.SpaceGetDefaultBranch(ctx, 3, false)
	require.NoError(t, err)
	require.Equal(t, DefaultBranch, got)

	value, source, found, err := svc.SpaceGetDefaultBranchWithSource(ctx, 3)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "trunk", value)
	require.Equal(t, int64(1), source)

	// Nothing configured anywhere falls back to the global default.
	empty := svbNewService(svbNewStore())
	ok, err = empty.SpaceGet(ctx, 3, DefaultBranchKey, true, &branch)
	require.NoError(t, err)
	require.False(t, ok)
	value, source, found, err = empty.SpaceGetDefaultBranchWithSource(ctx, 3)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, DefaultBranch, value)
	require.Zero(t, source)

	// Unknown spaces and store failures are errors.
	_, err = empty.SpaceGet(ctx, 99, DefaultBranchKey, true, &branch)
	require.ErrorContains(t, err, "failed to find space")
	_, _, _, err = empty.SpaceGetDefaultBranchWithSource(ctx, 99)
	require.ErrorContains(t, err, "failed to find space")

	st.failOn["Find"] = true
	_, err = svc.SpaceGet(ctx, 3, DefaultBranchKey, true, &branch)
	require.ErrorIs(t, err, errSvbBoom)
	_, _, _, err = svc.SpaceGetDefaultBranchWithSource(ctx, 3)
	require.ErrorIs(t, err, errSvbBoom)
	_, err = svc.SpaceGetDefaultBranch(ctx, 3, true)
	require.ErrorIs(t, err, errSvbBoom)
}

func TestSvbSpaceUpdateGeneralSettings(t *testing.T) {
	ctx := context.Background()
	st := svbNewStore()
	svc := svbNewService(st)

	branch := "develop"
	oldS, newS, err := SpaceUpdateGeneralSettings(ctx, svc, 2, &GeneralSettingsSpace{DefaultBranch: &branch})
	require.NoError(t, err)
	require.Equal(t, DefaultBranch, *oldS.DefaultBranch)
	require.Equal(t, "develop", *newS.DefaultBranch)

	// A nil value is neither set nor deleted.
	_, newS, err = SpaceUpdateGeneralSettings(ctx, svc, 2, &GeneralSettingsSpace{})
	require.NoError(t, err)
	require.Equal(t, "develop", *newS.DefaultBranch)

	st.failOn["DeleteMany"] = true
	empty := ""
	_, _, err = SpaceUpdateGeneralSettings(ctx, svc, 2, &GeneralSettingsSpace{DefaultBranch: &empty})
	require.Error(t, err)
}
