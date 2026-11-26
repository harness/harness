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

package database_test

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/cache"
	"github.com/harness/gitness/app/store/database"
	"github.com/harness/gitness/app/store/database/migrate"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

const (
	st5Sha1       = "1111111111111111111111111111111111111111"
	st5Sha2       = "2222222222222222222222222222222222222222"
	st5SecondUser = int64(2)
)

var errSt5Cache = errors.New("st5 cache failure")

// st5Env is a migrated SQLite database with foreign keys enabled, a user, a root space and two repositories.
type st5Env struct {
	db             *sqlx.DB
	principalStore *database.PrincipalStore
	spaceStore     *database.SpaceStore
	spacePathStore store.SpacePathStore
	repoStore      *database.RepoStore
	repoIDs        []int64
}

func st5NewEnv(ctx context.Context, t *testing.T) *st5Env {
	t.Helper()

	db, err := sqlx.Connect(st5SqliteDriver, "file:"+filepath.Join(t.TempDir(), "st5.db")+"?_foreign_keys=on")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, migrate.Migrate(ctx, db))
	_, err = db.ExecContext(ctx, `PRAGMA foreign_keys = ON`)
	require.NoError(t, err)

	principalStore := database.NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	spacePathStore := database.NewSpacePathStore(db, store.ToLowerSpacePathTransformation)
	evictor := cache.NewEvictor[*types.SpaceCore]("st5", "st5-topic", nil)
	spacePathCache := cache.New(ctx, spacePathStore, store.ToLowerSpacePathTransformation, evictor, time.Minute)
	spaceStore := database.NewSpaceStore(db, spacePathCache, spacePathStore)
	repoStore := database.NewRepoStore(db, spacePathCache, spacePathStore, spaceStore)
	env := &st5Env{
		db: db, principalStore: principalStore, spaceStore: spaceStore,
		spacePathStore: spacePathStore, repoStore: repoStore,
	}

	createUser(ctx, t, principalStore)
	require.NoError(t, principalStore.CreateUser(ctx,
		&types.User{ID: st5SecondUser, UID: "user_2", Email: "user2@example.com"}))
	createSpace(ctx, t, spaceStore, spacePathStore, userID, 1, 0)

	for i := range 2 {
		id := int64(i + 1)
		identifier := "st5repo_" + strconv.FormatInt(id, 10)
		repo := types.Repository{
			ID: id, Identifier: identifier, GitUID: identifier, ParentID: 1, Tags: json.RawMessage{},
		}
		require.NoError(t, repoStore.Create(ctx, &repo))
		env.repoIDs = append(env.repoIDs, repo.ID)
	}

	return env
}

// st5PrincipalCache resolves principal info for the check store.
type st5PrincipalCache struct {
	store.PrincipalInfoCache

	err error
}

func (c st5PrincipalCache) Map(_ context.Context, keys []int64) (map[int64]*types.PrincipalInfo, error) {
	if c.err != nil {
		return nil, c.err
	}
	res := make(map[int64]*types.PrincipalInfo, len(keys))
	for _, k := range keys {
		res[k] = &types.PrincipalInfo{ID: k, UID: "user_" + strconv.FormatInt(k, 10)}
	}

	return res, nil
}

func st5UpsertCheck(
	ctx context.Context, t *testing.T, s *database.CheckStore, repoID int64, sha, uid string,
	status enum.CheckStatus, bypassedBy *int64,
) types.Check {
	t.Helper()

	c := types.Check{
		CreatedBy: userID, Created: 1000, Updated: 1000, RepoID: repoID, CommitSHA: sha, Identifier: uid,
		Status: status, BypassedByID: bypassedBy,
		Metadata: json.RawMessage("{}"), Payload: types.CheckPayload{Data: json.RawMessage("{}")},
	}
	require.NoError(t, s.Upsert(ctx, &c))

	return c
}

func TestSt5CheckStore(t *testing.T) {
	ctx := context.Background()
	env := st5NewEnv(ctx, t)
	s := database.NewCheckStore(env.db, st5PrincipalCache{})
	repo, other := env.repoIDs[0], env.repoIDs[1]

	bypasser := st5SecondUser
	st5UpsertCheck(ctx, t, s, repo, st5Sha1, "lint", enum.CheckStatusSuccess, nil)
	st5UpsertCheck(ctx, t, s, repo, st5Sha1, "build", enum.CheckStatusFailure, &bypasser)
	st5UpsertCheck(ctx, t, s, repo, st5Sha1, "unit", enum.CheckStatusPending, nil)
	st5UpsertCheck(ctx, t, s, repo, st5Sha2, "lint", enum.CheckStatusRunning, nil)
	st5UpsertCheck(ctx, t, s, other, st5Sha1, "deploy", enum.CheckStatusError, nil)

	t.Run("find and upsert update", func(t *testing.T) {
		got, err := s.FindByIdentifier(ctx, repo, st5Sha1, "lint")
		require.NoError(t, err)
		require.Equal(t, enum.CheckStatusSuccess, got.Status)

		updated := st5UpsertCheck(ctx, t, s, repo, st5Sha1, "lint", enum.CheckStatusFailureIgnored, nil)
		require.Equal(t, got.ID, updated.ID)
		got, err = s.FindByIdentifier(ctx, repo, st5Sha1, "lint")
		require.NoError(t, err)
		require.Equal(t, enum.CheckStatusFailureIgnored, got.Status)

		_, err = s.FindByIdentifier(ctx, repo, st5Sha1, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("count", func(t *testing.T) {
		n, err := s.Count(ctx, repo, st5Sha1, types.CheckListOptions{})
		require.NoError(t, err)
		require.Equal(t, 3, n)

		n, err = s.Count(ctx, repo, st5Sha1, types.CheckListOptions{ListQueryFilter: types.ListQueryFilter{Query: "uil"}})
		require.NoError(t, err)
		require.Equal(t, 1, n)
	})

	t.Run("list maps principals", func(t *testing.T) {
		list, err := s.List(ctx, repo, st5Sha1, types.CheckListOptions{})
		require.NoError(t, err)
		require.Len(t, list, 3)

		byUID := map[string]types.Check{}
		for _, c := range list {
			byUID[c.Identifier] = c
			require.NotNil(t, c.ReportedBy)
			require.Equal(t, userID, c.ReportedBy.ID)
		}
		require.NotNil(t, byUID["build"].BypassedBy)
		require.Equal(t, st5SecondUser, byUID["build"].BypassedBy.ID)
		require.Nil(t, byUID["unit"].BypassedBy)

		page, err := s.List(ctx, repo, st5Sha1, types.CheckListOptions{
			ListQueryFilter: types.ListQueryFilter{Pagination: types.Pagination{Page: 2, Size: 2}},
		})
		require.NoError(t, err)
		require.Len(t, page, 1)

		filtered, err := s.List(ctx, repo, st5Sha1, types.CheckListOptions{
			ListQueryFilter: types.ListQueryFilter{Query: "lin"},
		})
		require.NoError(t, err)
		require.Len(t, filtered, 1)
		require.Equal(t, "lint", filtered[0].Identifier)
	})

	t.Run("list principal cache failure", func(t *testing.T) {
		failing := database.NewCheckStore(env.db, st5PrincipalCache{err: errSt5Cache})
		_, err := failing.List(ctx, repo, st5Sha1, types.CheckListOptions{})
		require.ErrorIs(t, err, errSt5Cache)
	})

	t.Run("list recent", func(t *testing.T) {
		got, err := s.ListRecent(ctx, repo, types.CheckRecentOptions{})
		require.NoError(t, err)
		require.Equal(t, []string{"build", "lint", "unit"}, got)

		got, err = s.ListRecent(ctx, repo, types.CheckRecentOptions{Query: "nit"})
		require.NoError(t, err)
		require.Equal(t, []string{"unit"}, got)

		got, err = s.ListRecent(ctx, repo, types.CheckRecentOptions{Since: 1 << 60})
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("list recent space", func(t *testing.T) {
		got, err := s.ListRecentSpace(ctx, []int64{1}, types.CheckRecentOptions{})
		require.NoError(t, err)
		require.Equal(t, []string{"build", "deploy", "lint", "unit"}, got)

		got, err = s.ListRecentSpace(ctx, []int64{1}, types.CheckRecentOptions{Query: "dep"})
		require.NoError(t, err)
		require.Equal(t, []string{"deploy"}, got)

		got, err = s.ListRecentSpace(ctx, []int64{999}, types.CheckRecentOptions{})
		require.NoError(t, err)
		require.Empty(t, got)
	})

	t.Run("list results", func(t *testing.T) {
		res, err := s.ListResults(ctx, repo, st5Sha1)
		require.NoError(t, err)
		require.Len(t, res, 3)
		for _, r := range res {
			if r.Identifier == "build" {
				require.NotNil(t, r.BypassedBy)
				require.Equal(t, st5SecondUser, *r.BypassedBy)
			}
		}
	})

	t.Run("result summary rejects invalid commit sha", func(t *testing.T) {
		st5UpsertCheck(ctx, t, s, repo, "not-a-sha", "lint", enum.CheckStatusSuccess, nil)
		_, err := s.ResultSummary(ctx, repo, []string{"not-a-sha"}, nil)
		require.ErrorContains(t, err, "invalid commit SHA")
	})
}

func TestSt5LinkedRepoStore(t *testing.T) {
	ctx := context.Background()
	env := st5NewEnv(ctx, t)
	s := database.NewLinkedRepoStore(env.db)

	newLinked := func(repoID int64, providerID, connectorPath string, lastSync int64) *types.LinkedRepo {
		return &types.LinkedRepo{
			RepoID: repoID, Created: 1, Updated: 1, LastFullSync: lastSync,
			ConnectorPath: connectorPath, ConnectorIdentifier: "conn", ConnectorRepo: "acme/widgets",
			ProviderRepoID: providerID, ProviderType: "github",
		}
	}
	first, second := env.repoIDs[0], env.repoIDs[1]
	require.NoError(t, s.Create(ctx, newLinked(first, "100", "account.acme", 20)))
	require.NoError(t, s.Create(ctx, newLinked(second, "100", "account.acme/org", 10)))

	t.Run("create duplicate fails", func(t *testing.T) {
		require.ErrorIs(t, s.Create(ctx, newLinked(first, "1", "p", 0)), gitness_store.ErrDuplicate)
	})

	t.Run("find missing", func(t *testing.T) {
		_, err := s.Find(ctx, 999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("update bumps version and detects conflicts", func(t *testing.T) {
		linked, err := s.Find(ctx, first)
		require.NoError(t, err)

		stale := *linked
		linked.ProviderRepoID = "200"
		linked.LastFullSync = 99
		require.NoError(t, s.Update(ctx, linked))
		require.EqualValues(t, 1, linked.Version)

		got, err := s.Find(ctx, first)
		require.NoError(t, err)
		require.Equal(t, "200", got.ProviderRepoID)
		require.EqualValues(t, 99, got.LastFullSync)

		stale.ProviderRepoID = "300"
		require.ErrorIs(t, s.Update(ctx, &stale), gitness_store.ErrVersionConflict)
	})

	t.Run("update opt lock retries on conflict", func(t *testing.T) {
		stale := types.LinkedRepo{RepoID: second, Version: 5}
		calls := 0
		res, err := s.UpdateOptLock(ctx, &stale, func(l *types.LinkedRepo) error {
			calls++
			l.ConnectorRepo = "acme/renamed"
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, "acme/renamed", res.ConnectorRepo)

		got, err := s.Find(ctx, second)
		require.NoError(t, err)
		require.Equal(t, "acme/renamed", got.ConnectorRepo)
	})

	t.Run("update opt lock propagates mutate errors", func(t *testing.T) {
		_, err := s.UpdateOptLock(ctx, &types.LinkedRepo{RepoID: second}, func(*types.LinkedRepo) error {
			return errSt5Cache
		})
		require.ErrorIs(t, err, errSt5Cache)
	})

	t.Run("update opt lock fails when the repo vanished", func(t *testing.T) {
		_, err := s.UpdateOptLock(ctx, &types.LinkedRepo{RepoID: 999, Version: 3}, func(*types.LinkedRepo) error {
			return nil
		})
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("list orders by last sync", func(t *testing.T) {
		list, err := s.List(ctx, 10)
		require.NoError(t, err)
		require.Len(t, list, 2)
		require.Equal(t, second, list[0].RepoID)
		require.Equal(t, first, list[1].RepoID)

		list, err = s.List(ctx, 1)
		require.NoError(t, err)
		require.Len(t, list, 1)
	})

	t.Run("list by provider id", func(t *testing.T) {
		list, err := s.ListByProviderID(ctx, "account.acme", "github", "100", types.Pagination{Page: 1, Size: 10})
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, second, list[0].RepoID)

		list, err = s.ListByProviderID(ctx, "account.acme", "gitlab", "100", types.Pagination{})
		require.NoError(t, err)
		require.Empty(t, list)
	})
}
