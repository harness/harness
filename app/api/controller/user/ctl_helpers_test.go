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

package user

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/harness/gitness/app/auth"
	userevents "github.com/harness/gitness/app/events/user"
	"github.com/harness/gitness/app/services/refcache"
	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	"github.com/harness/gitness/events"
	"github.com/harness/gitness/store/database/dbtx"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/check"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

var (
	errCtlAuthz = errors.New("authz failure")
)

type ctlAuthMode int

const (
	ctlAllow ctlAuthMode = iota
	ctlDeny
	ctlError
)

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

type ctlEnv struct {
	db         *sqlx.DB
	principal  *database.PrincipalStore
	tokens     *database.TokenStore
	favorites  *database.FavoriteStore
	repos      *database.RepoStore
	spaces     *database.SpaceStore
	spacePaths store.SpacePathStore
	ctrl       *Controller
}

func ctlNewEnv(t *testing.T) *ctlEnv {
	t.Helper()
	ctx := t.Context()

	// Use a cheap bcrypt cost to keep the tests fast.
	orig := hashPassword
	hashPassword = func(pw []byte, _ int) ([]byte, error) { return bcrypt.GenerateFromPassword(pw, bcrypt.MinCost) }
	t.Cleanup(func() { hashPassword = orig })

	db, err := sqlx.Connect("sqlite3", filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	_, err = db.ExecContext(t.Context(), `PRAGMA foreign_keys = ON;`)
	require.NoError(t, err)
	_, _ = db.ExecContext(t.Context(), "PRAGMA busy_timeout = 5000;")
	require.NoError(t, migrate.Migrate(ctx, db))

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	spacePathCache := cache.New(
		ctx, spacePathStore, store.ToLowerSpacePathTransformation,
		cache.NewEvictor[*types.SpaceCore]("ns", "space", nil), 0)
	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)
	pCache := cache.ProvidePrincipalInfoCache(database.NewPrincipalInfoView(db))
	membershipStore := database.NewMembershipStore(db, pCache, spacePathStore, spaceStore)
	tokenStore := database.NewTokenStore(db)
	favoriteStore := database.NewFavoriteStore(db)

	repoEvictor := cache.NewEvictor[*types.RepositoryCore]("ns", "repo", nil)
	spaceEvictor := cache.NewEvictor[*types.SpaceCore]("ns", "space", nil)
	repoIDCache := cache.NewRepoIDCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)
	repoRefCache := cache.NewRepoRefCache(ctx, repoStore, spaceEvictor, repoEvictor, 0)
	repoFinder := refcache.NewRepoFinder(repoStore, spacePathCache, repoIDCache, repoRefCache, repoEvictor)

	reporter := ctlNewReporter(t)

	ctrl := NewController(
		dbtx.New(db),
		check.PrincipalUIDDefault,
		ctlAuthorizer{mode: ctlAllow},
		principalStore,
		tokenStore,
		membershipStore,
		database.NewPublicKeyStore(db),
		database.NewPublicKeySubKeyStore(db),
		database.NewGitSignatureResultStore(db),
		reporter,
		repoFinder,
		favoriteStore,
	)

	return &ctlEnv{
		db:         db,
		principal:  principalStore,
		tokens:     tokenStore,
		favorites:  favoriteStore,
		repos:      repoStore,
		spaces:     spaceStore,
		spacePaths: spacePathStore,
		ctrl:       ctrl,
	}
}

func ctlNewReporter(t *testing.T) *userevents.Reporter {
	t.Helper()
	sys, err := events.ProvideSystem(
		events.Config{Mode: events.ModeInMemory, Namespace: "ctl", MaxStreamLength: 100},
		nil, events.NewNoopCollector())
	require.NoError(t, err)
	reporter, err := userevents.NewReporter(sys)
	require.NoError(t, err)
	return reporter
}

// ctlSession returns a session for the given user.
func ctlSession(u *types.User) *auth.Session {
	return &auth.Session{Principal: *u.ToPrincipal()}
}

func ctlCreateUser(t *testing.T, e *ctlEnv, uid string) *types.User {
	t.Helper()
	u, err := e.ctrl.CreateNoAuth(t.Context(), &CreateInput{
		UID:         uid,
		Email:       uid + "@example.com",
		DisplayName: "User " + uid,
		Password:    "password123",
	}, false)
	require.NoError(t, err)
	return u
}

// ctlCreateRepo persists a root space (with its path segment) and a repository inside of it.
func ctlCreateRepo(t *testing.T, e *ctlEnv, owner *types.User, identifier string) *types.Repository {
	t.Helper()
	ctx := t.Context()

	space := &types.Space{Identifier: "space-" + identifier, CreatedBy: owner.ID}
	require.NoError(t, e.spaces.Create(ctx, space))
	require.NoError(t, e.spacePaths.InsertSegment(ctx, &types.SpacePathSegment{
		Identifier: space.Identifier, SpaceID: space.ID, IsPrimary: true, CreatedBy: owner.ID,
	}))

	repo := &types.Repository{
		Identifier:          identifier,
		ParentID:            space.ID,
		CreatedBy:           owner.ID,
		GitUID:              "git-" + identifier,
		State:               enum.RepoStateActive,
		RootSpaceID:         space.ID,
		Tags:                []byte("{}"),
		RootSpaceIdentifier: space.Identifier,
	}
	require.NoError(t, e.repos.Create(ctx, repo))
	return repo
}
