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
	"errors"
	"testing"
	"time"

	appstore "github.com/harness/gitness/app/store"
	"github.com/harness/gitness/app/store/database"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"
)

func newPrincipalFixture(t *testing.T) (*sqlx.DB, *database.PrincipalStore) {
	t.Helper()

	db, teardown := setupDB(t)
	t.Cleanup(teardown)

	return db, database.NewPrincipalStore(db, appstore.ToLowerPrincipalUIDTransformation)
}

func TestPrincipalStore_UserLifecycle(t *testing.T) {
	_, ps := newPrincipalFixture(t)
	ctx := context.Background()

	user := &types.User{
		UID: "Jane", Email: "Jane@example.com", DisplayName: "Jane Doe", Admin: true,
		Salt: "salt", Password: "hash", Created: 100, Updated: 100,
	}
	require.NoError(t, ps.CreateUser(ctx, user))
	require.NotZero(t, user.ID)

	t.Run("find by id, uid and email", func(t *testing.T) {
		byID, err := ps.FindUser(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, "Jane Doe", byID.DisplayName)
		require.True(t, byID.Admin)
		require.Equal(t, "hash", byID.Password)

		// The uid transformation lower-cases lookups, so the lookup is case insensitive.
		byUID, err := ps.FindUserByUID(ctx, "JANE")
		require.NoError(t, err)
		require.Equal(t, user.ID, byUID.ID)

		byEmail, err := ps.FindUserByEmail(ctx, "jane@example.com")
		require.NoError(t, err)
		require.Equal(t, user.ID, byEmail.ID)
	})

	t.Run("missing user", func(t *testing.T) {
		_, err := ps.FindUser(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ps.FindUserByUID(ctx, "nobody")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ps.FindUserByEmail(ctx, "nobody@example.com")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("duplicate uid", func(t *testing.T) {
		err := ps.CreateUser(ctx, &types.User{UID: "jane", Email: "other@example.com"})
		require.ErrorIs(t, err, gitnessstore.ErrDuplicate)
	})

	t.Run("update", func(t *testing.T) {
		user.DisplayName = "Jane D."
		user.Blocked = true
		user.Updated = 200
		require.NoError(t, ps.UpdateUser(ctx, user))

		got, err := ps.FindUser(ctx, user.ID)
		require.NoError(t, err)
		require.Equal(t, "Jane D.", got.DisplayName)
		require.True(t, got.Blocked)
		require.Equal(t, int64(200), got.Updated)
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, ps.DeleteUser(ctx, user.ID))
		_, err := ps.FindUser(ctx, user.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}

func TestPrincipalStore_ListAndCountUsers(t *testing.T) {
	_, ps := newPrincipalFixture(t)
	ctx := context.Background()

	for _, u := range []*types.User{
		{UID: "carol", Email: "c@x.io", DisplayName: "Carol", Created: 3, Updated: 30},
		{UID: "alice", Email: "a@x.io", DisplayName: "Alice", Admin: true, Created: 1, Updated: 10},
		{UID: "bob", Email: "B@x.io", DisplayName: "Bob", Created: 2, Updated: 20},
	} {
		require.NoError(t, ps.CreateUser(ctx, u))
	}

	uids := func(users []*types.User) []string {
		out := make([]string, len(users))
		for i, u := range users {
			out[i] = u.UID
		}
		return out
	}

	tests := []struct {
		name   string
		filter types.UserFilter
		want   []string
	}{
		{"default sorts by name asc", types.UserFilter{Page: 1, Size: 10}, []string{"alice", "bob", "carol"}},
		{"name desc", types.UserFilter{Page: 1, Size: 10, Sort: enum.UserAttrName, Order: enum.OrderDesc},
			[]string{"carol", "bob", "alice"}},
		{"created desc", types.UserFilter{Page: 1, Size: 10, Sort: enum.UserAttrCreated, Order: enum.OrderDesc},
			[]string{"carol", "bob", "alice"}},
		{"updated asc", types.UserFilter{Page: 1, Size: 10, Sort: enum.UserAttrUpdated},
			[]string{"alice", "bob", "carol"}},
		{"email asc is case insensitive", types.UserFilter{Page: 1, Size: 10, Sort: enum.UserAttrEmail},
			[]string{"alice", "bob", "carol"}},
		{"uid desc", types.UserFilter{Page: 1, Size: 10, Sort: enum.UserAttrUID, Order: enum.OrderDesc},
			[]string{"carol", "bob", "alice"}},
		{"admin desc", types.UserFilter{Page: 1, Size: 1, Sort: enum.UserAttrAdmin, Order: enum.OrderDesc},
			[]string{"alice"}},
		{"second page", types.UserFilter{Page: 2, Size: 2}, []string{"carol"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ps.ListUsers(ctx, &tt.filter)
			require.NoError(t, err)
			require.Equal(t, tt.want, uids(got))
		})
	}

	total, err := ps.CountUsers(ctx, &types.UserFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(3), total)

	admins, err := ps.CountUsers(ctx, &types.UserFilter{Admin: true})
	require.NoError(t, err)
	require.Equal(t, int64(1), admins)
}

func TestPrincipalStore_FindAndListPrincipals(t *testing.T) {
	_, ps := newPrincipalFixture(t)
	ctx := context.Background()

	require.NoError(t, ps.CreateUser(ctx, &types.User{UID: "alice", Email: "alice@x.io", DisplayName: "Alice A"}))
	require.NoError(t, ps.CreateUser(ctx, &types.User{UID: "bob", Email: "bob@x.io", DisplayName: "Bob B"}))
	svc := &types.Service{UID: "gitness", Email: "svc@x.io", DisplayName: "Gitness"}
	require.NoError(t, ps.CreateService(ctx, svc))

	t.Run("find", func(t *testing.T) {
		p, err := ps.FindByUID(ctx, "ALICE")
		require.NoError(t, err)
		require.Equal(t, enum.PrincipalTypeUser, p.Type)

		byID, err := ps.Find(ctx, p.ID)
		require.NoError(t, err)
		require.Equal(t, p.UID, byID.UID)

		byEmail, err := ps.FindByEmail(ctx, "svc@x.io")
		require.NoError(t, err)
		require.Equal(t, enum.PrincipalTypeService, byEmail.Type)

		_, err = ps.Find(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ps.FindByUID(ctx, "ghost")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ps.FindByEmail(ctx, "ghost@x.io")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("find many", func(t *testing.T) {
		many, err := ps.FindManyByUID(ctx, []string{"alice", "bob", "ghost"})
		require.NoError(t, err)
		require.Len(t, many, 2)

		empty, err := ps.FindManyByUID(ctx, nil)
		require.NoError(t, err)
		require.Empty(t, empty)

		byEmail, err := ps.FindManyByEmail(ctx, []string{"alice@x.io", "bob@x.io", "ghost@x.io"})
		require.NoError(t, err)
		require.Len(t, byEmail, 2)
	})

	t.Run("list filters", func(t *testing.T) {
		all, err := ps.List(ctx, &types.PrincipalFilter{Page: 1, Size: 10})
		require.NoError(t, err)
		require.Len(t, all, 3)

		users, err := ps.List(ctx, &types.PrincipalFilter{
			Page: 1, Size: 10, Types: []enum.PrincipalType{enum.PrincipalTypeUser},
		})
		require.NoError(t, err)
		require.Len(t, users, 2)

		both, err := ps.List(ctx, &types.PrincipalFilter{
			Page: 1, Size: 10,
			Types: []enum.PrincipalType{enum.PrincipalTypeUser, enum.PrincipalTypeService},
		})
		require.NoError(t, err)
		require.Len(t, both, 3)

		byQuery, err := ps.List(ctx, &types.PrincipalFilter{Page: 1, Size: 10, Query: "bob"})
		require.NoError(t, err)
		require.Len(t, byQuery, 1)
		require.Equal(t, "bob", byQuery[0].UID)

		paged, err := ps.List(ctx, &types.PrincipalFilter{Page: 2, Size: 2})
		require.NoError(t, err)
		require.Len(t, paged, 1)
	})
}

func TestPrincipalStore_ServiceLifecycle(t *testing.T) {
	_, ps := newPrincipalFixture(t)
	ctx := context.Background()

	svc := &types.Service{UID: "Gitness", Email: "svc@x.io", DisplayName: "Gitness", Admin: true, Created: 1, Updated: 1}
	require.NoError(t, ps.CreateService(ctx, svc))
	require.NotZero(t, svc.ID)

	require.ErrorIs(t, ps.CreateService(ctx, &types.Service{UID: "gitness", Email: "dup@x.io"}), gitnessstore.ErrDuplicate)

	byID, err := ps.FindService(ctx, svc.ID)
	require.NoError(t, err)
	require.True(t, byID.Admin)

	byUID, err := ps.FindServiceByUID(ctx, "GITNESS")
	require.NoError(t, err)
	require.Equal(t, svc.ID, byUID.ID)

	_, err = ps.FindService(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = ps.FindServiceByUID(ctx, "ghost")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	svc.DisplayName = "Renamed"
	svc.Blocked = true
	require.NoError(t, ps.UpdateService(ctx, svc))
	updated, err := ps.FindService(ctx, svc.ID)
	require.NoError(t, err)
	require.Equal(t, "Renamed", updated.DisplayName)
	require.True(t, updated.Blocked)

	list, err := ps.ListServices(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	count, err := ps.CountServices(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)

	require.NoError(t, ps.DeleteService(ctx, svc.ID))
	count, err = ps.CountServices(ctx)
	require.NoError(t, err)
	require.Zero(t, count)
}

func TestPrincipalStore_ServiceAccountLifecycle(t *testing.T) {
	_, ps := newPrincipalFixture(t)
	ctx := context.Background()

	newSA := func(uid, name string, parentID int64) *types.ServiceAccount {
		return &types.ServiceAccount{
			UID: uid, Email: uid + "@sa.x.io", DisplayName: name,
			ParentType: enum.ParentResourceTypeSpace, ParentID: parentID, Created: 1, Updated: 1,
		}
	}

	sa1, sa2, sa3 := newSA("Bot-One", "Build bot", 10), newSA("bot-two", "Deploy bot", 10), newSA("bot-three", "Other", 20)
	for _, sa := range []*types.ServiceAccount{sa1, sa2, sa3} {
		require.NoError(t, ps.CreateServiceAccount(ctx, sa))
		require.NotZero(t, sa.ID)
	}

	t.Run("duplicate", func(t *testing.T) {
		err := ps.CreateServiceAccount(ctx, newSA("bot-one", "dup", 10))
		require.ErrorIs(t, err, gitnessstore.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		byID, err := ps.FindServiceAccount(ctx, sa1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(10), byID.ParentID)
		require.Equal(t, enum.ParentResourceTypeSpace, byID.ParentType)

		byUID, err := ps.FindServiceAccountByUID(ctx, "BOT-ONE")
		require.NoError(t, err)
		require.Equal(t, sa1.ID, byUID.ID)

		many, err := ps.FindManyServiceAccountByUID(ctx, []string{"bot-one", "BOT-TWO", "ghost"})
		require.NoError(t, err)
		require.Len(t, many, 2)

		_, err = ps.FindServiceAccount(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ps.FindServiceAccountByUID(ctx, "ghost")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("list and count by parent", func(t *testing.T) {
		parents := []*types.ServiceAccountParentInfo{{Type: enum.ParentResourceTypeSpace, ID: 10}}
		filter := &types.PrincipalFilter{Page: 1, Size: 10}

		list, err := ps.ListServiceAccounts(ctx, parents, filter)
		require.NoError(t, err)
		require.Len(t, list, 2)
		count, err := ps.CountServiceAccounts(ctx, parents, filter)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)

		withQuery := &types.PrincipalFilter{Page: 1, Size: 10, Query: "deploy"}
		list, err = ps.ListServiceAccounts(ctx, parents, withQuery)
		require.NoError(t, err)
		require.Len(t, list, 1)
		require.Equal(t, "bot-two", list[0].UID)
		count, err = ps.CountServiceAccounts(ctx, parents, withQuery)
		require.NoError(t, err)
		require.Equal(t, int64(1), count)

		bothParents := []*types.ServiceAccountParentInfo{
			{Type: enum.ParentResourceTypeSpace, ID: 10},
			{Type: enum.ParentResourceTypeSpace, ID: 20},
		}
		list, err = ps.ListServiceAccounts(ctx, bothParents, filter)
		require.NoError(t, err)
		require.Len(t, list, 3)

		none, err := ps.ListServiceAccounts(ctx, nil, filter)
		require.NoError(t, err)
		require.Empty(t, none, "no parents means nothing matches")
	})

	t.Run("update and delete", func(t *testing.T) {
		sa1.DisplayName = "Renamed bot"
		sa1.Blocked = true
		require.NoError(t, ps.UpdateServiceAccount(ctx, sa1))
		got, err := ps.FindServiceAccount(ctx, sa1.ID)
		require.NoError(t, err)
		require.Equal(t, "Renamed bot", got.DisplayName)
		require.True(t, got.Blocked)

		require.NoError(t, ps.DeleteServiceAccount(ctx, sa1.ID))
		_, err = ps.FindServiceAccount(ctx, sa1.ID)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}

func TestPrincipalInfoView(t *testing.T) {
	db, ps := newPrincipalFixture(t)
	ctx := context.Background()

	alice := &types.User{UID: "alice", Email: "alice@x.io", DisplayName: "Alice"}
	bob := &types.User{UID: "bob", Email: "bob@x.io", DisplayName: "Bob"}
	require.NoError(t, ps.CreateUser(ctx, alice))
	require.NoError(t, ps.CreateUser(ctx, bob))

	view := database.NewPrincipalInfoView(db)

	info, err := view.Find(ctx, alice.ID)
	require.NoError(t, err)
	require.Equal(t, "alice", info.UID)
	require.Equal(t, "Alice", info.DisplayName)
	require.Equal(t, enum.PrincipalTypeUser, info.Type)

	_, err = view.Find(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	many, err := view.FindMany(ctx, []int64{alice.ID, bob.ID, 9999})
	require.NoError(t, err)
	require.Len(t, many, 2)

	none, err := view.FindMany(ctx, nil)
	require.NoError(t, err)
	require.Empty(t, none)
}

func TestTokenStore(t *testing.T) {
	db, ps := newPrincipalFixture(t)
	ctx := context.Background()

	owner := &types.User{UID: "owner", Email: "owner@x.io"}
	require.NoError(t, ps.CreateUser(ctx, owner))

	ts := database.NewTokenStore(db)
	now := time.Now()
	expired := now.Add(-time.Hour).UnixMilli()
	valid := now.Add(time.Hour).UnixMilli()

	pat := &types.Token{
		PrincipalID: owner.ID, Type: enum.TokenTypePAT, Identifier: "my-pat", IssuedAt: 1, CreatedBy: owner.ID,
	}
	oldSession := &types.Token{
		PrincipalID: owner.ID, Type: enum.TokenTypeSession, Identifier: "old-session",
		ExpiresAt: &expired, IssuedAt: 1, CreatedBy: owner.ID,
	}
	newSession := &types.Token{
		PrincipalID: owner.ID, Type: enum.TokenTypeSession, Identifier: "new-session",
		ExpiresAt: &valid, IssuedAt: 2, CreatedBy: owner.ID,
	}
	for _, tok := range []*types.Token{pat, oldSession, newSession} {
		require.NoError(t, ts.Create(ctx, tok))
		require.NotZero(t, tok.ID)
	}

	require.ErrorIs(t, ts.Create(ctx, &types.Token{
		PrincipalID: owner.ID, Type: enum.TokenTypePAT, Identifier: "my-pat", CreatedBy: owner.ID,
	}), gitnessstore.ErrDuplicate)

	byID, err := ts.Find(ctx, pat.ID)
	require.NoError(t, err)
	require.Equal(t, "my-pat", byID.Identifier)
	require.Nil(t, byID.ExpiresAt)

	byIdentifier, err := ts.FindByIdentifier(ctx, owner.ID, "new-session")
	require.NoError(t, err)
	require.Equal(t, newSession.ID, byIdentifier.ID)
	require.Equal(t, valid, *byIdentifier.ExpiresAt)

	_, err = ts.Find(ctx, 9999)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	_, err = ts.FindByIdentifier(ctx, owner.ID, "ghost")
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	count, err := ts.Count(ctx, owner.ID, enum.TokenTypeSession)
	require.NoError(t, err)
	require.Equal(t, int64(2), count)

	list, err := ts.List(ctx, owner.ID, enum.TokenTypeSession)
	require.NoError(t, err)
	require.Len(t, list, 2)
	pats, err := ts.List(ctx, owner.ID, enum.TokenTypePAT)
	require.NoError(t, err)
	require.Len(t, pats, 1)

	// Only expired tokens of the requested types are removed.
	deleted, err := ts.DeleteExpiredBefore(ctx, now, []enum.TokenType{enum.TokenTypePAT})
	require.NoError(t, err)
	require.Zero(t, deleted)
	deleted, err = ts.DeleteExpiredBefore(ctx, now, []enum.TokenType{enum.TokenTypeSession})
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)
	deleted, err = ts.DeleteExpiredBefore(ctx, now.Add(2*time.Hour), nil)
	require.NoError(t, err)
	require.Equal(t, int64(1), deleted)

	require.NoError(t, ts.Delete(ctx, pat.ID))
	_, err = ts.Find(ctx, pat.ID)
	require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

	require.False(t, errors.Is(ts.Delete(ctx, pat.ID), gitnessstore.ErrDuplicate))
}
