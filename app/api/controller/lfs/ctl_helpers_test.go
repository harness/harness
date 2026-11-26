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

package lfs

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/auth"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var errCtlAuthz = errors.New("authz failure")

type ctlAuthMode int

const (
	ctlAllow ctlAuthMode = iota
	ctlDeny
	ctlError
)

// ctlAuthorizer is a fake authz.Authorizer that allows, denies or fails every check.
type ctlAuthorizer struct {
	mode ctlAuthMode
}

func (a ctlAuthorizer) decide() (bool, error) {
	switch a.mode {
	case ctlAllow:
		return true, nil
	case ctlError:
		return false, errCtlAuthz
	case ctlDeny:
		return false, nil
	}
	return false, nil
}

func (a ctlAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.decide()
}

func (a ctlAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.decide()
}

func (a ctlAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	ok, err := a.decide()
	res := make([]bool, len(checks))
	for i := range res {
		res[i] = ok
	}
	return res, err
}

// ctlBase bundles a migrated SQLite database with the stores and finders built on top of it.
type ctlBase struct {
	db             *sqlx.DB
	tx             dbtx.Transactor
	principals     *database.PrincipalStore
	spaces         *database.SpaceStore
	spacePaths     store.SpacePathStore
	repos          *database.RepoStore
	memberships    *database.MembershipStore
	spaceFinder    refcache.SpaceFinder
	repoFinder     refcache.RepoFinder
	spacePathCache store.SpacePathCache
	admin          *types.User
}

func ctlNewBase(t *testing.T) *ctlBase {
	t.Helper()
	ctx := t.Context()

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(t.Context(), `PRAGMA foreign_keys = ON;`)
	require.NoError(t, err)
	_, _ = db.ExecContext(t.Context(), "PRAGMA busy_timeout = 5000;")
	require.NoError(t, migrate.Migrate(ctx, db))

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", nil)
	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", nil)
	spacePathCache := cache.New(ctx, spacePathStore, store.ToLowerSpacePathTransformation, spaceEvictor, 0)
	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)
	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(db))
	membershipStore := database.NewMembershipStore(db, pCache, spacePathStore, spaceStore)

	spaceIDCache := cache.NewSpaceIDCache(ctx, spaceStore, spaceEvictor, 0)
	spaceCICache := cache.NewSpaceCaseInsensitiveCache(ctx, spaceStore, spaceEvictor, 0)
	repoIDCache := cache.NewRepoIDCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)
	repoRefCache := cache.NewRepoRefCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)

	b := &ctlBase{
		db:             db,
		tx:             dbtx.New(db),
		principals:     principalStore,
		spaces:         spaceStore,
		spacePaths:     spacePathStore,
		repos:          repoStore,
		memberships:    membershipStore,
		spacePathCache: spacePathCache,
		spaceFinder:    refcache.NewSpaceFinder(spaceIDCache, spacePathCache, spaceCICache, spaceEvictor),
		repoFinder:     refcache.NewRepoFinder(repoStore, spacePathCache, repoIDCache, repoRefCache, repoEvictor),
	}
	b.admin = b.createUser(t, "admin", true)
	return b
}

func (b *ctlBase) createUser(t *testing.T, uid string, admin bool) *types.User {
	t.Helper()
	u := &types.User{
		UID: uid, Email: uid + "@example.com", DisplayName: uid, Salt: "salt", Admin: admin,
	}
	require.NoError(t, b.principals.CreateUser(t.Context(), u))
	return u
}

// session returns a session for the admin user.
func (b *ctlBase) session() *auth.Session {
	return &auth.Session{Principal: *b.admin.ToPrincipal()}
}

// createSpace persists a space (and its path segment); parentID 0 creates a root space.
func (b *ctlBase) createSpace(t *testing.T, identifier string, parentID int64) *types.Space {
	t.Helper()
	ctx := t.Context()

	space := &types.Space{Identifier: identifier, ParentID: parentID, CreatedBy: b.admin.ID}
	require.NoError(t, b.spaces.Create(ctx, space))
	require.NoError(t, b.spacePaths.InsertSegment(ctx, &types.SpacePathSegment{
		Identifier: space.Identifier, SpaceID: space.ID, ParentID: parentID, IsPrimary: true, CreatedBy: b.admin.ID,
	}))
	return space
}

// createRepo persists a repository below the given root space.
func (b *ctlBase) createRepo(t *testing.T, space *types.Space, identifier string) *types.Repository {
	t.Helper()

	repo := &types.Repository{
		Identifier:          identifier,
		ParentID:            space.ID,
		CreatedBy:           b.admin.ID,
		GitUID:              "git-" + identifier,
		State:               enum.RepoStateActive,
		RootSpaceID:         space.ID,
		RootSpaceIdentifier: space.Identifier,
		Tags:                []byte("{}"),
		DefaultBranch:       "main",
	}
	require.NoError(t, b.repos.Create(t.Context(), repo))
	return repo
}
