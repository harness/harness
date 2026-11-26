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

package githook

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
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

var errCxrAuthz = errors.New("authz failure")

type cxrAuthMode int

const (
	cxrAllow cxrAuthMode = iota
	cxrDeny
	cxrError
)

// cxrAuthorizer is a fake authz.Authorizer that allows, denies or fails every check.
type cxrAuthorizer struct {
	mode cxrAuthMode
}

func (a cxrAuthorizer) decide() (bool, error) {
	switch a.mode {
	case cxrAllow:
		return true, nil
	case cxrError:
		return false, errCxrAuthz
	case cxrDeny:
		return false, nil
	}
	return false, nil
}

func (a cxrAuthorizer) Check(
	context.Context, *auth.Session, *types.Scope, *types.Resource, enum.Permission,
) (bool, error) {
	return a.decide()
}

func (a cxrAuthorizer) CheckAll(context.Context, *auth.Session, ...types.PermissionCheck) (bool, error) {
	return a.decide()
}

func (a cxrAuthorizer) CheckMany(
	_ context.Context, _ *auth.Session, checks ...types.PermissionCheck,
) ([]bool, error) {
	ok, err := a.decide()
	res := make([]bool, len(checks))
	for i := range res {
		res[i] = ok
	}
	return res, err
}

// cxrBase bundles a migrated SQLite database with stores and finders built on top of it.
type cxrBase struct {
	db          *sqlx.DB
	principals  *database.PrincipalStore
	spaces      *database.SpaceStore
	spacePaths  store.SpacePathStore
	repos       *database.RepoStore
	spaceFinder refcache.SpaceFinder
	repoFinder  refcache.RepoFinder
	admin       *types.User
}

func cxrNewBase(ctx context.Context, t *testing.T) *cxrBase {
	t.Helper()

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(ctx, `PRAGMA foreign_keys = ON;`)
	require.NoError(t, err)
	_, _ = db.ExecContext(ctx, "PRAGMA busy_timeout = 5000;")
	require.NoError(t, migrate.Migrate(ctx, db))

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", nil)
	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", nil)
	spacePathCache := cache.New(ctx, spacePathStore, store.ToLowerSpacePathTransformation, spaceEvictor, 0)
	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)

	spaceIDCache := cache.NewSpaceIDCache(ctx, spaceStore, spaceEvictor, 0)
	spaceCICache := cache.NewSpaceCaseInsensitiveCache(ctx, spaceStore, spaceEvictor, 0)
	repoIDCache := cache.NewRepoIDCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)
	repoRefCache := cache.NewRepoRefCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)

	b := &cxrBase{
		db:          db,
		principals:  principalStore,
		spaces:      spaceStore,
		spacePaths:  spacePathStore,
		repos:       repoStore,
		spaceFinder: refcache.NewSpaceFinder(spaceIDCache, spacePathCache, spaceCICache, spaceEvictor),
		repoFinder:  refcache.NewRepoFinder(repoStore, spacePathCache, repoIDCache, repoRefCache, repoEvictor),
	}
	b.admin = b.createUser(ctx, t, "admin", true)
	return b
}

func (b *cxrBase) createUser(ctx context.Context, t *testing.T, uid string, admin bool) *types.User {
	t.Helper()
	u := &types.User{
		UID: uid, Email: uid + "@example.com", DisplayName: uid, Salt: "salt", Admin: admin,
	}
	require.NoError(t, b.principals.CreateUser(ctx, u))
	return u
}

// session returns a session for the admin user.
func (b *cxrBase) session() *auth.Session {
	return &auth.Session{Principal: *b.admin.ToPrincipal()}
}

// createSpace persists a root space (and its path segment).
func (b *cxrBase) createSpace(ctx context.Context, t *testing.T, identifier string) *types.Space {
	t.Helper()

	space := &types.Space{Identifier: identifier, CreatedBy: b.admin.ID}
	require.NoError(t, b.spaces.Create(ctx, space))
	space.RootSpaceID, space.RootSpaceIdentifier = space.ID, space.Identifier
	require.NoError(t, b.spaces.UpdateRootSpace(ctx, []int64{space.ID}, space.ID, space.Identifier))
	require.NoError(t, b.spacePaths.InsertSegment(ctx, &types.SpacePathSegment{
		Identifier: space.Identifier, SpaceID: space.ID, IsPrimary: true, CreatedBy: b.admin.ID,
	}))
	return space
}

// createRepo persists a repository below the given root space.
func (b *cxrBase) createRepo(
	ctx context.Context, t *testing.T, space *types.Space, identifier string,
) *types.Repository {
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
	require.NoError(t, b.repos.Create(ctx, repo))
	return repo
}
