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
	"time"

	"github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

// Shared string values used by the rst* tests.
const (
	rstValRenamed = "renamed"
	rstValChanged = "changed"
	rstValUpdated = "updated"
	rstValStale   = "stale"
	rstValOptLock = "opt lock"
)

// rstFixture bundles an SQLite database with the space, space path and repo stores wired together.
type rstFixture struct {
	db         *sqlx.DB
	principals *database.PrincipalStore
	spaces     *database.SpaceStore
	paths      store.SpacePathStore
	repos      *database.RepoStore
	clock      int64
}

// rstNewFixture creates a migrated database with a single user (id 1) and returns the stores.
func rstNewFixture(t *testing.T) (context.Context, *rstFixture) {
	t.Helper()

	db, teardown := setupDB(t)
	t.Cleanup(teardown)

	principals, spaces, paths, repos := setupStores(t, db)
	ctx := context.Background()
	createUser(ctx, t, principals)

	return ctx, &rstFixture{db: db, principals: principals, spaces: spaces, paths: paths, repos: repos}
}

// rstMkSpace creates a space under parentID (0 for root) together with its primary path segment.
func (f *rstFixture) rstMkSpace(ctx context.Context, t *testing.T, parentID int64, ident string) *types.Space {
	t.Helper()

	f.clock++
	sp := &types.Space{
		Identifier: ident, ParentID: parentID, CreatedBy: userID, Description: "desc " + ident,
		Created: 1000 + f.clock, Updated: 1000 + f.clock,
	}
	require.NoError(t, f.spaces.Create(ctx, sp))
	require.NoError(t, f.paths.InsertSegment(ctx, &types.SpacePathSegment{
		Identifier: ident, ParentID: parentID, SpaceID: sp.ID, IsPrimary: true, CreatedBy: userID,
	}))

	return sp
}

// rstMkRepo creates an active repo under spaceID.
func (f *rstFixture) rstMkRepo(ctx context.Context, t *testing.T, spaceID int64, ident string) *types.Repository {
	t.Helper()

	repo := &types.Repository{
		Identifier: ident, GitUID: "uid-" + ident, ParentID: spaceID, CreatedBy: userID,
		Tags: json.RawMessage(`{}`),
	}
	require.NoError(t, f.repos.Create(ctx, repo))

	return repo
}

// rstSpaceTree builds: alpha{beta{gamma}, epsilon}, delta. Returns spaces keyed by identifier.
func (f *rstFixture) rstSpaceTree(ctx context.Context, t *testing.T) map[string]*types.Space {
	t.Helper()

	tree := map[string]*types.Space{}
	tree["alpha"] = f.rstMkSpace(ctx, t, 0, "alpha")
	tree["beta"] = f.rstMkSpace(ctx, t, tree["alpha"].ID, "beta")
	tree["gamma"] = f.rstMkSpace(ctx, t, tree["beta"].ID, "gamma")
	tree["epsilon"] = f.rstMkSpace(ctx, t, tree["alpha"].ID, "epsilon")
	tree["delta"] = f.rstMkSpace(ctx, t, 0, "delta")

	return tree
}

func rstSpaceIDs(spaces []*types.Space) []int64 {
	ids := make([]int64, len(spaces))
	for i, s := range spaces {
		ids[i] = s.ID
	}

	return ids
}

func TestRstSpaceStore_FindAndPaths(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)

	t.Run("find by id", func(t *testing.T) {
		got, err := f.spaces.Find(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.Equal(t, "gamma", got.Identifier)
		require.Equal(t, "alpha/beta/gamma", got.Path)
		require.Equal(t, tree["beta"].ID, got.ParentID)
	})

	t.Run("find missing", func(t *testing.T) {
		_, err := f.spaces.Find(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find by ref", func(t *testing.T) {
		tests := []struct {
			name string
			ref  string
			want string
		}{
			{"numeric id", "2", "beta"},
			{"root path", "alpha", "alpha"},
			{"nested path", "alpha/beta/gamma", "gamma"},
			{"case insensitive path", "ALPHA/Beta", "beta"},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := f.spaces.FindByRef(ctx, tt.ref)
				require.NoError(t, err)
				require.Equal(t, tt.want, got.Identifier)
			})
		}

		_, err := f.spaces.FindByRef(ctx, "alpha/nope")
		require.Error(t, err)
	})

	t.Run("find by ref case insensitive", func(t *testing.T) {
		id, err := f.spaces.FindByRefCaseInsensitive(ctx, "ALPHA")
		require.NoError(t, err)
		require.Equal(t, tree["alpha"].ID, id)

		id, err = f.spaces.FindByRefCaseInsensitive(ctx, "alpha/beta/Gamma")
		require.NoError(t, err)
		require.Equal(t, tree["gamma"].ID, id)

		_, err = f.spaces.FindByRefCaseInsensitive(ctx, "missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = f.spaces.FindByRefCaseInsensitive(ctx, "")
		require.Error(t, err)
	})

	t.Run("find by ids", func(t *testing.T) {
		got, err := f.spaces.FindByIDs(ctx, tree["alpha"].ID, tree["delta"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []int64{tree["alpha"].ID, tree["delta"].ID}, rstSpaceIDs(got))

		got, err = f.spaces.FindByIDs(ctx)
		require.NoError(t, err)
		require.Empty(t, got)
	})
}

func TestRstSpaceStore_Hierarchy(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)

	t.Run("root space", func(t *testing.T) {
		root, err := f.spaces.GetRootSpace(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.Equal(t, tree["alpha"].ID, root.ID)
	})

	t.Run("all root spaces", func(t *testing.T) {
		roots, err := f.spaces.GetAllRootSpaces(ctx, &types.SpaceFilter{Order: enum.OrderAsc})
		require.NoError(t, err)
		require.Equal(t, []int64{tree["alpha"].ID, tree["delta"].ID}, rstSpaceIDs(roots))

		// the default order is descending.
		roots, err = f.spaces.GetAllRootSpaces(ctx, &types.SpaceFilter{})
		require.NoError(t, err)
		require.Equal(t, []int64{tree["delta"].ID, tree["alpha"].ID}, rstSpaceIDs(roots))

		roots, err = f.spaces.GetAllRootSpaces(ctx, &types.SpaceFilter{
			Query: "del", Sort: enum.SpaceAttrCreated, Order: enum.OrderDesc,
		})
		require.NoError(t, err)
		require.Equal(t, []int64{tree["delta"].ID}, rstSpaceIDs(roots))
	})

	t.Run("ancestors", func(t *testing.T) {
		ids, err := f.spaces.GetAncestorIDs(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []int64{tree["gamma"].ID, tree["beta"].ID, tree["alpha"].ID}, ids)

		level, err := f.spaces.GetTreeLevel(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.Equal(t, int64(3), level)

		level, err = f.spaces.GetTreeLevel(ctx, tree["delta"].ID)
		require.NoError(t, err)
		require.Equal(t, int64(1), level)

		ancestors, err := f.spaces.GetAncestors(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, ids, rstSpaceIDs(ancestors))

		data, err := f.spaces.GetAncestorsData(ctx, tree["beta"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []types.SpaceParentData{
			{ID: tree["beta"].ID, Identifier: "beta", ParentID: tree["alpha"].ID},
			{ID: tree["alpha"].ID, Identifier: "alpha", ParentID: 0},
		}, data)
	})

	t.Run("descendants", func(t *testing.T) {
		ids, err := f.spaces.GetDescendantsIDs(ctx, tree["alpha"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []int64{
			tree["alpha"].ID, tree["beta"].ID, tree["gamma"].ID, tree["epsilon"].ID,
		}, ids)

		data, err := f.spaces.GetDescendantsData(ctx, tree["beta"].ID)
		require.NoError(t, err)
		require.ElementsMatch(t, []types.SpaceParentData{
			{ID: tree["beta"].ID, Identifier: "beta", ParentID: tree["alpha"].ID},
			{ID: tree["gamma"].ID, Identifier: "gamma", ParentID: tree["beta"].ID},
		}, data)
	})
}

func TestRstSpaceStore_CreateUpdate(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)

	t.Run("nil space", func(t *testing.T) {
		require.Error(t, f.spaces.Create(ctx, nil))
		require.Error(t, f.spaces.Update(ctx, nil))
	})

	t.Run("unknown parent violates foreign key", func(t *testing.T) {
		err := f.spaces.Create(ctx, &types.Space{Identifier: "orphan", ParentID: 9999, CreatedBy: userID})
		require.ErrorIs(t, err, gitness_store.ErrForeignKeyViolation)
	})

	t.Run("update bumps version and keeps path", func(t *testing.T) {
		sp, err := f.spaces.Find(ctx, tree["epsilon"].ID)
		require.NoError(t, err)

		sp.Description = rstValChanged
		require.NoError(t, f.spaces.Update(ctx, sp))
		require.Equal(t, int64(1), sp.Version)
		require.Equal(t, "alpha/epsilon", sp.Path)

		got, err := f.spaces.Find(ctx, sp.ID)
		require.NoError(t, err)
		require.Equal(t, rstValChanged, got.Description)
		require.Equal(t, sp.Version, got.Version)
	})

	t.Run("update with stale version conflicts", func(t *testing.T) {
		stale, err := f.spaces.Find(ctx, tree["delta"].ID)
		require.NoError(t, err)
		fresh := *stale

		stale.Description = "first"
		require.NoError(t, f.spaces.Update(ctx, stale))

		fresh.Description = "second"
		require.ErrorIs(t, f.spaces.Update(ctx, &fresh), gitness_store.ErrVersionConflict)
	})

	t.Run("update root space", func(t *testing.T) {
		require.NoError(t, f.spaces.UpdateRootSpace(ctx, nil, 1, "alpha"))

		ids := []int64{tree["beta"].ID, tree["gamma"].ID}
		require.NoError(t, f.spaces.UpdateRootSpace(ctx, ids, tree["alpha"].ID, "alpha"))

		for _, id := range ids {
			got, err := f.spaces.Find(ctx, id)
			require.NoError(t, err)
			require.Equal(t, tree["alpha"].ID, got.RootSpaceID)
			require.Equal(t, "alpha", got.RootSpaceIdentifier)
		}

		other, err := f.spaces.Find(ctx, tree["delta"].ID)
		require.NoError(t, err)
		require.Zero(t, other.RootSpaceID)
	})

	t.Run("update opt lock", func(t *testing.T) {
		sp, err := f.spaces.Find(ctx, tree["gamma"].ID)
		require.NoError(t, err)

		// a concurrent writer bumps the version so the first attempt conflicts and is retried.
		concurrent := *sp
		concurrent.Description = "concurrent"
		require.NoError(t, f.spaces.Update(ctx, &concurrent))

		calls := 0
		updated, err := f.spaces.UpdateOptLock(ctx, sp, func(s *types.Space) error {
			calls++
			s.Description = "via opt lock"
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 2, calls)
		require.Equal(t, "via opt lock", updated.Description)

		boom := errors.New("mutate failed")
		_, err = f.spaces.UpdateOptLock(ctx, sp, func(*types.Space) error { return boom })
		require.ErrorIs(t, err, boom)
	})

	t.Run("find for update", func(t *testing.T) {
		got, err := f.spaces.FindForUpdate(ctx, tree["alpha"].ID)
		require.NoError(t, err)
		require.Equal(t, "alpha", got.Identifier)
	})
}

func TestRstSpaceStore_SoftDeleteRestorePurge(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)

	deletedAt := time.Now().UnixMilli()
	gamma, err := f.spaces.Find(ctx, tree["gamma"].ID)
	require.NoError(t, err)

	require.NoError(t, f.spaces.SoftDelete(ctx, gamma, deletedAt))
	// the service removes the primary path of a soft deleted space.
	require.NoError(t, f.paths.DeletePrimarySegment(ctx, gamma.ID))

	t.Run("hidden from active lookups", func(t *testing.T) {
		_, err := f.spaces.Find(ctx, gamma.ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		ids, err := f.spaces.GetDescendantsIDs(ctx, tree["beta"].ID)
		require.NoError(t, err)
		// the recursive CTE does not filter on deletion.
		require.Contains(t, ids, gamma.ID)
	})

	t.Run("find by ref and deleted at", func(t *testing.T) {
		got, err := f.spaces.FindByRefAndDeletedAt(ctx, "alpha/beta/gamma", deletedAt)
		require.NoError(t, err)
		require.Equal(t, gamma.ID, got.ID)
		require.Equal(t, "alpha/beta/gamma", got.Path)
		require.NotNil(t, got.Deleted)

		got, err = f.spaces.FindByRefAndDeletedAt(ctx, "2", deletedAt)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
		require.Nil(t, got)

		got, err = f.spaces.FindByRefAndDeletedAt(ctx, "3", deletedAt)
		require.NoError(t, err)
		require.Equal(t, gamma.ID, got.ID)

		_, err = f.spaces.FindByRefAndDeletedAt(ctx, "alpha/beta/gamma", deletedAt+1)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = f.spaces.FindByRefAndDeletedAt(ctx, "", deletedAt)
		require.Error(t, err)
	})

	t.Run("find root by path and deleted at", func(t *testing.T) {
		delta, err := f.spaces.Find(ctx, tree["delta"].ID)
		require.NoError(t, err)
		require.NoError(t, f.spaces.SoftDelete(ctx, delta, deletedAt))

		got, err := f.spaces.FindByRefAndDeletedAt(ctx, "delta", deletedAt)
		require.NoError(t, err)
		require.Equal(t, delta.ID, got.ID)
	})

	t.Run("update opt lock rejects deleted spaces", func(t *testing.T) {
		deleted, err := f.spaces.FindByRefAndDeletedAt(ctx, "3", deletedAt)
		require.NoError(t, err)

		_, err = f.spaces.UpdateOptLock(ctx, deleted, func(*types.Space) error { return nil })
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("restore with new identifier", func(t *testing.T) {
		deleted, err := f.spaces.FindByRefAndDeletedAt(ctx, "3", deletedAt)
		require.NoError(t, err)

		newIdent := "gamma2"
		restored, err := f.spaces.Restore(ctx, deleted, &newIdent, nil)
		require.NoError(t, err)
		require.Nil(t, restored.Deleted)
		require.Equal(t, newIdent, restored.Identifier)

		got, err := f.spaces.Find(ctx, gamma.ID)
		require.NoError(t, err)
		require.Equal(t, newIdent, got.Identifier)

		// restoring an active space is rejected.
		_, err = f.spaces.Restore(ctx, got, nil, nil)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("restore into new parent", func(t *testing.T) {
		epsilon, err := f.spaces.Find(ctx, tree["epsilon"].ID)
		require.NoError(t, err)
		require.NoError(t, f.spaces.SoftDelete(ctx, epsilon, deletedAt))

		deleted, err := f.spaces.FindByRefAndDeletedAt(ctx, "alpha/epsilon", deletedAt)
		require.NoError(t, err)

		parent := tree["beta"].ID
		restored, err := f.spaces.Restore(ctx, deleted, nil, &parent)
		require.NoError(t, err)
		require.Equal(t, parent, restored.ParentID)
	})

	t.Run("purge", func(t *testing.T) {
		delta, err := f.spaces.FindByRefAndDeletedAt(ctx, "delta", deletedAt)
		require.NoError(t, err)

		// wrong deleted timestamp is a no-op
		require.NoError(t, f.spaces.Purge(ctx, delta.ID, rstPtrInt64(deletedAt+1)))
		_, err = f.spaces.FindByRefAndDeletedAt(ctx, "delta", deletedAt)
		require.NoError(t, err)

		require.NoError(t, f.spaces.Purge(ctx, delta.ID, &deletedAt))
		_, err = f.spaces.FindByRefAndDeletedAt(ctx, "delta", deletedAt)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		// active purge removes an active space
		require.NoError(t, f.paths.DeletePathsAndDescendandPaths(ctx, tree["gamma"].ID))
		require.NoError(t, f.spaces.Purge(ctx, tree["gamma"].ID, nil))
		_, err = f.spaces.Find(ctx, tree["gamma"].ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})
}

func rstPtrInt64(v int64) *int64 { return &v }

func TestRstSpaceStore_CountAndList(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)
	alpha := tree["alpha"].ID

	deletedAt := time.Now().UnixMilli()
	delta, err := f.spaces.Find(ctx, tree["delta"].ID)
	require.NoError(t, err)
	require.NoError(t, f.spaces.SoftDelete(ctx, delta, deletedAt))

	type tc struct {
		name   string
		parent int64
		filter types.SpaceFilter
		want   []string
	}
	tests := []tc{
		{"direct children", alpha, types.SpaceFilter{}, []string{"beta", "epsilon"}},
		{"direct children sorted desc", alpha, types.SpaceFilter{Sort: enum.SpaceAttrIdentifier, Order: enum.OrderDesc},
			[]string{"epsilon", "beta"}},
		{"direct children by created desc", alpha, types.SpaceFilter{Sort: enum.SpaceAttrCreated, Order: enum.OrderDesc},
			[]string{"epsilon", "beta"}},
		{"query", alpha, types.SpaceFilter{Query: "eps"}, []string{"epsilon"}},
		{"paged", alpha, types.SpaceFilter{Page: 2, Size: 1}, []string{"epsilon"}},
		{"recursive", alpha, types.SpaceFilter{Recursive: true}, []string{"beta", "epsilon", "gamma"}},
		{"recursive with query", alpha, types.SpaceFilter{Recursive: true, Query: "amm"}, []string{"gamma"}},
		{"recursive updated", alpha, types.SpaceFilter{Recursive: true, Sort: enum.SpaceAttrUpdated},
			[]string{"beta", "gamma", "epsilon"}},
		{"none under leaf", tree["gamma"].ID, types.SpaceFilter{}, []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.filter.Order == enum.OrderDefault {
				tt.filter.Order = enum.OrderAsc
			}
			got, err := f.spaces.List(ctx, tt.parent, &tt.filter)
			require.NoError(t, err)

			idents := make([]string, 0, len(got))
			for _, s := range got {
				idents = append(idents, s.Identifier)
			}
			require.Equal(t, tt.want, idents)

			if tt.filter.Size > 0 {
				return
			}
			count, err := f.spaces.Count(ctx, tt.parent, &types.SpaceFilter{
				Recursive: tt.filter.Recursive, Query: tt.filter.Query,
			})
			require.NoError(t, err)
			require.Equal(t, int64(len(tt.want)), count)
		})
	}

	t.Run("deleted filters", func(t *testing.T) {
		// delta is a root space; list deleted spaces under it is empty, so move a child under it.
		child := f.rstMkSpace(ctx, t, alpha, "zeta")
		fresh, err := f.spaces.Find(ctx, child.ID)
		require.NoError(t, err)
		require.NoError(t, f.spaces.SoftDelete(ctx, fresh, deletedAt))

		got, err := f.spaces.List(ctx, alpha, &types.SpaceFilter{DeletedAt: &deletedAt})
		require.NoError(t, err)
		require.Equal(t, []int64{child.ID}, rstSpaceIDs(got))

		got, err = f.spaces.List(ctx, alpha, &types.SpaceFilter{
			DeletedBeforeOrAt: &deletedAt, Sort: enum.SpaceAttrDeleted,
		})
		require.NoError(t, err)
		require.Equal(t, []int64{child.ID}, rstSpaceIDs(got))

		before := deletedAt - 1000
		count, err := f.spaces.Count(ctx, alpha, &types.SpaceFilter{DeletedBeforeOrAt: &before})
		require.NoError(t, err)
		require.Zero(t, count)

		// roots listing honours the deleted filter as well
		roots, err := f.spaces.GetAllRootSpaces(ctx, &types.SpaceFilter{DeletedAt: &deletedAt})
		require.NoError(t, err)
		require.Equal(t, []int64{delta.ID}, rstSpaceIDs(roots))
	})
}

func TestRstSpacePathStore(t *testing.T) {
	ctx, f := rstNewFixture(t)
	tree := f.rstSpaceTree(ctx, t)

	t.Run("find primary by space id", func(t *testing.T) {
		p, err := f.paths.FindPrimaryBySpaceID(ctx, tree["gamma"].ID)
		require.NoError(t, err)
		require.Equal(t, "alpha/beta/gamma", p.Value)
		require.True(t, p.IsPrimary)
		require.Equal(t, tree["gamma"].ID, p.SpaceID)

		_, err = f.paths.FindPrimaryBySpaceID(ctx, 9999)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("find by path", func(t *testing.T) {
		p, err := f.paths.FindByPath(ctx, "ALPHA/Beta")
		require.NoError(t, err)
		require.Equal(t, "alpha/beta", p.Value)
		require.True(t, p.IsPrimary)
		require.Equal(t, tree["beta"].ID, p.SpaceID)

		_, err = f.paths.FindByPath(ctx, "alpha/missing")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = f.paths.FindByPath(ctx, "")
		require.Error(t, err)
	})

	t.Run("non primary alias", func(t *testing.T) {
		require.NoError(t, f.paths.InsertSegment(ctx, &types.SpacePathSegment{
			Identifier: "legacy", ParentID: tree["alpha"].ID, SpaceID: tree["epsilon"].ID,
			IsPrimary: false, CreatedBy: userID,
		}))

		p, err := f.paths.FindByPath(ctx, "alpha/legacy")
		require.NoError(t, err)
		require.False(t, p.IsPrimary)
		require.Equal(t, tree["epsilon"].ID, p.SpaceID)
	})

	t.Run("duplicate segment", func(t *testing.T) {
		err := f.paths.InsertSegment(ctx, &types.SpacePathSegment{
			Identifier: "BETA", ParentID: tree["alpha"].ID, SpaceID: tree["gamma"].ID, CreatedBy: userID,
		})
		require.ErrorIs(t, err, gitness_store.ErrDuplicate)
	})

	t.Run("delete primary segment", func(t *testing.T) {
		require.NoError(t, f.paths.DeletePrimarySegment(ctx, tree["delta"].ID))
		_, err := f.paths.FindByPath(ctx, "delta")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)
	})

	t.Run("delete paths and descendants", func(t *testing.T) {
		require.NoError(t, f.paths.DeletePathsAndDescendandPaths(ctx, tree["beta"].ID))

		_, err := f.paths.FindByPath(ctx, "alpha/beta")
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		_, err = f.paths.FindPrimaryBySpaceID(ctx, tree["gamma"].ID)
		require.ErrorIs(t, err, gitness_store.ErrResourceNotFound)

		// siblings and ancestors remain
		_, err = f.paths.FindByPath(ctx, "alpha/epsilon")
		require.NoError(t, err)
	})
}
