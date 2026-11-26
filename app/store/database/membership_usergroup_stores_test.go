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
	"testing"
	"time"

	"github.com/harness/gitness/app/store/database"
	gitnesscache "github.com/harness/gitness/cache"
	gitnessstore "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"
	"github.com/harness/gitness/types/enum"

	"github.com/stretchr/testify/require"
)

func TestMembershipStore(t *testing.T) {
	ctx, f := newScopedFixture(t)

	principalStore, spaceStore, spacePathStore, _ := setupStores(t, f.db)

	// Users 2 and 3 join space 1, user 2 also joins space 2.
	alice := &types.User{UID: "alice", Email: "alice@x.io", DisplayName: "Alice"}
	bob := &types.User{UID: "bob", Email: "bob@x.io", DisplayName: "Bob"}
	require.NoError(t, principalStore.CreateUser(ctx, alice))
	require.NoError(t, principalStore.CreateUser(ctx, bob))

	infoCache := gitnesscache.NewExtended[int64, *types.PrincipalInfo](
		database.NewPrincipalInfoView(f.db), time.Minute)
	ms := database.NewMembershipStore(f.db, infoCache, spacePathStore, spaceStore)

	newMembership := func(spaceID, principalID int64, role enum.MembershipRole, created int64) *types.Membership {
		return &types.Membership{
			MembershipKey: types.MembershipKey{SpaceID: spaceID, PrincipalID: principalID},
			CreatedBy:     userID, Created: created, Updated: created, Role: role,
		}
	}

	aliceInSpace1 := newMembership(1, alice.ID, enum.MembershipRoleReader, 10)
	bobInSpace1 := newMembership(1, bob.ID, enum.MembershipRoleContributor, 20)
	aliceInSpace2 := newMembership(2, alice.ID, enum.MembershipRoleExecutor, 30)
	for _, m := range []*types.Membership{aliceInSpace1, bobInSpace1, aliceInSpace2} {
		require.NoError(t, ms.Create(ctx, m))
	}

	t.Run("duplicate membership", func(t *testing.T) {
		dup := newMembership(1, alice.ID, enum.MembershipRoleReader, 1)
		require.ErrorIs(t, ms.Create(ctx, dup), gitnessstore.ErrDuplicate)
	})

	t.Run("find", func(t *testing.T) {
		got, err := ms.Find(ctx, aliceInSpace1.MembershipKey)
		require.NoError(t, err)
		require.Equal(t, enum.MembershipRoleReader, got.Role)
		require.Equal(t, userID, got.CreatedBy)

		withInfo, err := ms.FindUser(ctx, bobInSpace1.MembershipKey)
		require.NoError(t, err)
		require.Equal(t, "bob", withInfo.Principal.UID)
		require.Equal(t, "user_1", withInfo.AddedBy.UID)

		_, err = ms.Find(ctx, types.MembershipKey{SpaceID: 2, PrincipalID: bob.ID})
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = ms.FindUser(ctx, types.MembershipKey{SpaceID: 2, PrincipalID: bob.ID})
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("list and count users of a space", func(t *testing.T) {
		page := types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}

		users, err := ms.ListUsers(ctx, 1, types.MembershipUserFilter{
			ListQueryFilter: page, Sort: enum.MembershipUserSortName,
		})
		require.NoError(t, err)
		require.Len(t, users, 2)
		require.Equal(t, "alice", users[0].Principal.UID)

		desc, err := ms.ListUsers(ctx, 1, types.MembershipUserFilter{
			ListQueryFilter: page, Sort: enum.MembershipUserSortCreated, Order: enum.OrderDesc,
		})
		require.NoError(t, err)
		require.Equal(t, "bob", desc[0].Principal.UID)

		searched, err := ms.ListUsers(ctx, 1, types.MembershipUserFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "ali", Pagination: page.Pagination},
		})
		require.NoError(t, err)
		require.Len(t, searched, 1)

		count, err := ms.CountUsers(ctx, 1, types.MembershipUserFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		count, err = ms.CountUsers(ctx, 1, types.MembershipUserFilter{ListQueryFilter: types.ListQueryFilter{Query: "bo"}})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("list and count spaces of a user", func(t *testing.T) {
		page := types.ListQueryFilter{Pagination: types.Pagination{Page: 1, Size: 10}}

		spaces, err := ms.ListSpaces(ctx, alice.ID, types.MembershipSpaceFilter{
			ListQueryFilter: page, Sort: enum.MembershipSpaceSortUID,
		})
		require.NoError(t, err)
		require.Len(t, spaces, 2)
		require.Equal(t, "space_1", spaces[0].Space.Identifier)

		byCreated, err := ms.ListSpaces(ctx, alice.ID, types.MembershipSpaceFilter{
			ListQueryFilter: page, Sort: enum.MembershipSpaceSortCreated, Order: enum.OrderDesc,
		})
		require.NoError(t, err)
		require.Equal(t, "space_2", byCreated[0].Space.Identifier)

		searched, err := ms.ListSpaces(ctx, alice.ID, types.MembershipSpaceFilter{
			ListQueryFilter: types.ListQueryFilter{Query: "space_2", Pagination: page.Pagination},
		})
		require.NoError(t, err)
		require.Len(t, searched, 1)

		count, err := ms.CountSpaces(ctx, alice.ID, types.MembershipSpaceFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
		count, err = ms.CountSpaces(ctx, bob.ID, types.MembershipSpaceFilter{})
		require.NoError(t, err)
		require.Equal(t, int64(1), count)
	})

	t.Run("update and delete", func(t *testing.T) {
		aliceInSpace1.Role = enum.MembershipRoleContributor
		require.NoError(t, ms.Update(ctx, aliceInSpace1))

		got, err := ms.Find(ctx, aliceInSpace1.MembershipKey)
		require.NoError(t, err)
		require.Equal(t, enum.MembershipRoleContributor, got.Role)
		require.Equal(t, aliceInSpace1.Updated, got.Updated)

		require.NoError(t, ms.Delete(ctx, aliceInSpace1.MembershipKey))
		_, err = ms.Find(ctx, aliceInSpace1.MembershipKey)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})
}

func TestUserGroupStore(t *testing.T) {
	ctx, f := newScopedFixture(t)
	us := database.NewUserGroupStore(f.db)

	newGroup := func(uid, name string) *types.UserGroup {
		return &types.UserGroup{Identifier: uid, Name: name, Description: "d-" + uid, Created: 1, Updated: 1, Scope: 1}
	}

	devs, ops := newGroup("devs", "Developers"), newGroup("ops", "Operations")
	other := newGroup("devs", "Developers of space 2")
	require.NoError(t, us.Create(ctx, 1, devs))
	require.NoError(t, us.Create(ctx, 1, ops))
	require.NoError(t, us.Create(ctx, 2, other))
	require.NotZero(t, devs.ID)

	require.ErrorIs(t, us.Create(ctx, 1, newGroup("devs", "dup")), gitnessstore.ErrDuplicate)

	t.Run("find", func(t *testing.T) {
		got, err := us.Find(ctx, devs.ID)
		require.NoError(t, err)
		require.Equal(t, "Developers", got.Name)
		require.Equal(t, int64(1), got.SpaceID)

		byIdentifier, err := us.FindByIdentifier(ctx, 2, "devs")
		require.NoError(t, err)
		require.Equal(t, other.ID, byIdentifier.ID)

		_, err = us.Find(ctx, 9999)
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
		_, err = us.FindByIdentifier(ctx, 2, "ops")
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)
	})

	t.Run("find many", func(t *testing.T) {
		byIDs, err := us.FindManyByIDs(ctx, []int64{devs.ID, ops.ID, 9999})
		require.NoError(t, err)
		require.Len(t, byIDs, 2)

		mapped, err := us.Map(ctx, []int64{devs.ID, other.ID})
		require.NoError(t, err)
		require.Equal(t, "Developers", mapped[devs.ID].Name)
		require.Equal(t, int64(2), mapped[other.ID].SpaceID)

		_, err = us.Map(ctx, []int64{9999})
		require.ErrorIs(t, err, gitnessstore.ErrResourceNotFound)

		byIdentifiers, err := us.FindManyByIdentifiersAndSpaceID(ctx, []string{"devs", "ops", "missing"}, 1)
		require.NoError(t, err)
		require.Len(t, byIdentifiers, 2)
	})

	t.Run("create or update", func(t *testing.T) {
		require.NoError(t, us.CreateOrUpdate(ctx, 1, &types.UserGroup{
			Identifier: "DEVS", Name: "Engineers", Description: "renamed", Created: 5, Updated: 5, Scope: 1,
		}))
		got, err := us.Find(ctx, devs.ID)
		require.NoError(t, err)
		require.Equal(t, "Engineers", got.Name)
		require.Equal(t, "renamed", got.Description)
		require.Equal(t, int64(5), got.Updated)

		require.NoError(t, us.CreateOrUpdate(ctx, 1, &types.UserGroup{
			Identifier: "qa", Name: "QA", Created: 6, Updated: 6, Scope: 1,
		}))
		created, err := us.FindByIdentifier(ctx, 1, "qa")
		require.NoError(t, err)
		require.Equal(t, "QA", created.Name)
	})
}
