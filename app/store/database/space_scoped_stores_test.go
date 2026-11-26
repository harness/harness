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

package database_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var errBoom = errors.New("boom")

// scopedFixture holds a database with one user, two root spaces (1 and 2) and one repository (1) in space 1.
type scopedFixture struct {
	db *sqlx.DB
}

func newScopedFixture(t *testing.T) (context.Context, scopedFixture) {
	t.Helper()

	db, teardown := setupDB(t)
	t.Cleanup(teardown)

	principalStore, spaceStore, spacePathStore, repoStore := setupStores(t, db)
	ctx := context.Background()

	createUser(ctx, t, principalStore)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 2, 0)
	createRepo(ctx, t, repoStore, 1, 1, 0)

	return ctx, scopedFixture{db: db}
}

func TestSecretStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ss := database.NewSecretStore(f.db)

	newSecret := func(spaceID int64, uid string) *types.Secret {
		return &types.Secret{
			SpaceID: spaceID, Identifier: uid, Description: "d-" + uid, Data: "ciphertext",
			CreatedBy: userID, Created: 1, Updated: 1,
		}
	}

	api, db, other := newSecret(1, "api-key"), newSecret(1, "db-password"), newSecret(2, "api-key")
	for _, s := range []*types.Secret{api, db, other} {
		require.NoError(t, ss.Create(ctx, s))
		require.NotZero(t, s.ID)
	}

	t.Run("create duplicate in the same space", func(t *testing.T) {
		require.ErrorIs(t, ss.Create(ctx, newSecret(1, "api-key")), gitnessstore.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		byID, err := ss.Find(ctx, api.ID)
		require.NoError(t, err)
		require.Equal(t, "ciphertext", byID.Data)

		byIdentifier, err := ss.FindByIdentifier(ctx, 2, "api-key")
		require.NoError(t, err)
		require.Equal(t, other.ID, byIdentifier.ID)

		_, err = ss.Find(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ss.FindByIdentifier(ctx, 2, "db-password")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("list and count", func(t *testing.T) {
		all, err := ss.ListAll(ctx, 1)
		require.NoError(t, err)
		require.Len(t, all, 2)

		page, err := ss.List(ctx, 1, types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 1}})
		require.NoError(t, err)
		require.Len(t, page, 1)

		matched, err := ss.List(ctx, 1, types.ListQueryFilter{Query: "db", Pagination: types.Pagination{Page: 1, Size: 10}})
		require.NoError(t, err)
		require.Len(t, matched, 1)
		require.Equal(t, "db-password", matched[0].Identifier)

		count, err := ss.Count(ctx, 1, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		count, err = ss.Count(ctx, 1, types.ListQueryFilter{Query: "api"})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("update bumps the version and detects conflicts", func(t *testing.T) {
		stale := *api

		api.Description = "rotated"
		require.NoError(t, ss.Update(ctx, api))
		require.Equal(t, int64(1), api.Version)

		got, err := ss.Find(ctx, api.ID)
		require.NoError(t, err)
		require.Equal(t, "rotated", got.Description)

		stale.Description = "stale write"
		require.ErrorIs(t, ss.Update(ctx, &stale), gitnessstore.ErrVersionConflict)
	})

	t.Run("update with optimistic lock retries on conflict", func(t *testing.T) {
		stale := *db // Version 0 while the stored version moves on below.
		db.Description = "first"
		require.NoError(t, ss.Update(ctx, db))

		calls := 0
		updated, err := ss.UpdateOptLock(ctx, &stale, func(s *types.Secret) error {
			calls++
			s.Description = "second"
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls, "the first attempt conflicts, the second runs on the reloaded secret")
		require.Equal(t, "second", updated.Description)
		require.Equal(t, int64(2), updated.Version)

		_, err = ss.UpdateOptLock(ctx, updated, func(*types.Secret) error { return errBoom })
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, ss.Delete(ctx, api.ID))
		_, err := ss.Find(ctx, api.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		require.NoError(t, ss.DeleteByIdentifier(ctx, 1, "db-password"))
		count, err := ss.Count(ctx, 1, types.ListQueryFilter{})
		require.NoError(t, err)
		require.Zero(t, count)
	})
}

func TestTemplateStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ts := database.NewTemplateStore(f.db)

	newTemplate := func(uid string, typ enum.ResolverType) *types.Template {
		return &types.Template{
			SpaceID: 1, Identifier: uid, Type: typ, Description: "d", Data: "kind: step",
			Created: 1, Updated: 1,
		}
	}

	step, stage := newTemplate("build", enum.ResolverTypeStep), newTemplate("build", enum.ResolverTypeStage)
	for _, tpl := range []*types.Template{step, stage} {
		require.NoError(t, ts.Create(ctx, tpl))
		require.NotZero(t, tpl.ID)
	}

	require.ErrorIs(t, ts.Create(ctx, newTemplate("build", enum.ResolverTypeStep)), gitnessstore.ErrDuplicate,
		"identifier is unique per type")

	byID, err := ts.Find(ctx, step.ID)
	require.NoError(t, err)
	require.Equal(t, enum.ResolverTypeStep, byID.Type)

	byType, err := ts.FindByIdentifierAndType(ctx, 1, "build", enum.ResolverTypeStage)
	require.NoError(t, err)
	require.Equal(t, stage.ID, byType.ID)

	_, err = ts.Find(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = ts.FindByIdentifierAndType(ctx, 2, "build", enum.ResolverTypeStage)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	list, err := ts.List(ctx, 1, types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}})
	require.NoError(t, err)
	require.Len(t, list, 2)
	list, err = ts.List(ctx, 1, types.ListQueryFilter{Query: "nothing", Pagination: types.Pagination{Page: 1, Size: 10}})
	require.NoError(t, err)
	require.Empty(t, list)

	count, err := ts.Count(ctx, 1, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(2), count)
	count, err = ts.Count(ctx, 1, types.ListQueryFilter{Query: "nothing"})
	require.NoError(t, err)
	require.Zero(t, count)

	stale := *step
	step.Data = "kind: step\nv: 2"
	require.NoError(t, ts.Update(ctx, step))
	require.Equal(t, int64(1), step.Version)
	require.ErrorIs(t, ts.Update(ctx, &stale), gitnessstore.ErrVersionConflict)

	calls := 0
	updated, err := ts.UpdateOptLock(ctx, &stale, func(tpl *types.Template) error {
		calls++
		tpl.Description = "locked"
		return nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, calls)
	require.Equal(t, "locked", updated.Description)
	require.Equal(t, "kind: step\nv: 2", updated.Data, "the retry starts from the stored row")

	_, err = ts.UpdateOptLock(ctx, updated, func(*types.Template) error { return errBoom })
	require.ErrorIs(t, err, errBoom)

	require.NoError(t, ts.Delete(ctx, step.ID))
	require.NoError(t, ts.DeleteByIdentifierAndType(ctx, 1, "build", enum.ResolverTypeStage))
	count, err = ts.Count(ctx, 1, types.ListQueryFilter{})
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestSettingsStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	st := database.NewSettingsStore(f.db)

	raw := func(s string) json.RawMessage { return json.RawMessage(s) }

	t.Run("upsert and find per scope", func(t *testing.T) {
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeSystem, 0, "Theme", raw(`"dark"`)))
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeSpace, 1, "Theme", raw(`"light"`)))
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeRepo, 1, "Theme", raw(`"blue"`)))

		for scope, want := range map[enum.SettingsScope]string{
			enum.SettingsScopeSystem: `"dark"`,
			enum.SettingsScopeSpace:  `"light"`,
			enum.SettingsScopeRepo:   `"blue"`,
		} {
			got, err := st.Find(ctx, scope, 1, "theme") // The key lookup is case insensitive.
			require.NoError(t, err, scope)
			require.JSONEq(t, want, string(got), scope)
		}
	})

	t.Run("upsert overwrites", func(t *testing.T) {
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeSpace, 1, "THEME", raw(`"solarized"`)))
		got, err := st.Find(ctx, enum.SettingsScopeSpace, 1, "theme")
		require.NoError(t, err)
		require.JSONEq(t, `"solarized"`, string(got))

		// Writing the same value again is a no-op.
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeSpace, 1, "THEME", raw(`"solarized"`)))
	})

	t.Run("not found", func(t *testing.T) {
		_, err := st.Find(ctx, enum.SettingsScopeSpace, 2, "theme")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("unsupported scope", func(t *testing.T) {
		const bogus = enum.SettingsScope("bogus")
		_, err := st.Find(ctx, bogus, 1, "k")
		require.Error(t, err)
		_, err = st.FindMany(ctx, bogus, 1, "k")
		require.Error(t, err)
		require.Error(t, st.Upsert(ctx, bogus, 1, "k", raw(`1`)))
		require.Error(t, st.DeleteMany(ctx, bogus, 1, "k"))
	})

	t.Run("find many", func(t *testing.T) {
		require.NoError(t, st.Upsert(ctx, enum.SettingsScopeRepo, 1, "lang", raw(`"go"`)))

		got, err := st.FindMany(ctx, enum.SettingsScopeRepo, 1, "THEME", "lang", "missing")
		require.NoError(t, err)
		require.Len(t, got, 2)

		none, err := st.FindMany(ctx, enum.SettingsScopeRepo, 1)
		require.NoError(t, err)
		require.Empty(t, none)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, st.Delete(ctx, enum.SettingsScopeRepo, 1, "theme"))
		_, err := st.Find(ctx, enum.SettingsScopeRepo, 1, "theme")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		require.NoError(t, st.DeleteMany(ctx, enum.SettingsScopeRepo, 1, "lang"))
		require.NoError(t, st.DeleteMany(ctx, enum.SettingsScopeRepo, 1), "no keys is a no-op")

		// Other scopes keep their value.
		_, err = st.Find(ctx, enum.SettingsScopeSystem, 0, "theme")
		require.NoError(t, err)
	})
}

func TestFavoriteStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	fs := database.NewFavoriteStore(f.db)

	repoFavorite := &types.FavoriteResource{ID: 1, Type: enum.ResourceTypeRepo}

	require.NoError(t, fs.Create(ctx, userID, repoFavorite))
	require.Error(t, fs.Create(ctx, userID, repoFavorite), "a favorite can only be added once")

	got, err := fs.Map(ctx, userID, enum.ResourceTypeRepo, []int64{1, 2})
	require.NoError(t, err)
	require.Equal(t, map[int64]bool{1: true}, got)

	otherUser, err := fs.Map(ctx, userID+1, enum.ResourceTypeRepo, []int64{1})
	require.NoError(t, err)
	require.Empty(t, otherUser)

	require.NoError(t, fs.Delete(ctx, userID, repoFavorite))
	got, err = fs.Map(ctx, userID, enum.ResourceTypeRepo, []int64{1})
	require.NoError(t, err)
	require.Empty(t, got)

	unsupported := &types.FavoriteResource{ID: 1, Type: enum.ResourceTypeSpace}
	require.Error(t, fs.Create(ctx, userID, unsupported))
	require.Error(t, fs.Delete(ctx, userID, unsupported))
	_, err = fs.Map(ctx, userID, enum.ResourceTypeSpace, []int64{1})
	require.Error(t, err)
}

func TestLFSObjectStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ls := database.NewLFSObjectStore(f.db)

	size, err := ls.GetSizeInKBByRepoID(ctx, 1)
	require.NoError(t, err)
	require.Zero(t, size, "an empty repository has no LFS data")

	small := &types.LFSObject{OID: "aaa", Size: 2048, Created: 1, CreatedBy: userID, RepoID: 1}
	big := &types.LFSObject{OID: "bbb", Size: 3072, Created: 2, CreatedBy: userID, RepoID: 1}
	for _, obj := range []*types.LFSObject{small, big} {
		require.NoError(t, ls.Create(ctx, obj))
		require.NotZero(t, obj.ID)
	}

	require.ErrorIs(t, ls.Create(ctx, &types.LFSObject{OID: "aaa", Size: 1, RepoID: 1, CreatedBy: userID}),
		gitnessstore.ErrDuplicate)

	got, err := ls.Find(ctx, 1, "aaa")
	require.NoError(t, err)
	require.Equal(t, int64(2048), got.Size)

	_, err = ls.Find(ctx, 1, "zzz")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	many, err := ls.FindMany(ctx, 1, []string{"aaa", "bbb", "zzz"})
	require.NoError(t, err)
	require.Len(t, many, 2)

	size, err = ls.GetSizeInKBByRepoID(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, int64(5), size)
}

func TestPublicAccessStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	ps := database.NewPublicAccessStore(f.db)

	for _, typ := range []enum.PublicResourceType{enum.PublicResourceTypeRepo, enum.PublicResourceTypeSpace} {
		exists, err := ps.Find(ctx, typ, 1)
		require.NoError(t, err)
		require.False(t, exists, typ)

		require.NoError(t, ps.Create(ctx, typ, 1))
		exists, err = ps.Find(ctx, typ, 1)
		require.NoError(t, err)
		require.True(t, exists, typ)

		exists, err = ps.Find(ctx, typ, 2)
		require.NoError(t, err)
		require.False(t, exists, typ)

		require.NoError(t, ps.Delete(ctx, typ, 1))
		exists, err = ps.Find(ctx, typ, 1)
		require.NoError(t, err)
		require.False(t, exists, typ)
	}

	const bogus = enum.PublicResourceType("bogus")
	_, err := ps.Find(ctx, bogus, 1)
	require.Error(t, err)
	require.Error(t, ps.Create(ctx, bogus, 1))
	require.Error(t, ps.Delete(ctx, bogus, 1))
}
