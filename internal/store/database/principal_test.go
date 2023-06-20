// Copyright 2022 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/internal/store"
	gitness_store "github.com/harness/gitness/store"
	"github.com/harness/gitness/types"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

// user fields to ignore in test comparisons.
var userIgnore = cmpopts.IgnoreFields(types.User{},
	"ID", "Salt", "Created", "Updated")

func TestUser(t *testing.T) {
	db, err := connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func(db *sqlx.DB) {
		_ = db.Close()
	}(db)
	if err = seed(db); err != nil {
		t.Error(err)
		return
	}

	principalStore := NewPrincipalStore(db, store.ToLowerPrincipalUIDTransformation)
	t.Run("create", testUserCreate(principalStore))
	t.Run("duplicate", testUserDuplicate(principalStore))
	t.Run("count", testUserCount(principalStore))
	t.Run("find", testUserFind(principalStore))
	t.Run("list", testUserList(principalStore))
	t.Run("update", testUserUpdate(principalStore))
	t.Run("delete", testUserDelete(principalStore))
}

// this test creates entries in the database and confirms
// the primary keys were auto-incremented.
func testUserCreate(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		vv := []*types.User{}
		if err := unmarshal("testdata/users.json", &vv); err != nil {
			t.Error(err)
			return
		}
		// create row 1
		v := vv[0]
		// generate a deterministic token for each
		// entry based on the hash of the email.
		v.Salt = fmt.Sprintf("%x", v.Email)
		if err := store.CreateUser(ctx, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}
		// create row 2
		v = vv[1]
		v.Salt = fmt.Sprintf("%x", v.Email)
		if err := store.CreateUser(ctx, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}
	}
}

// this test attempts to create an entry in the database using
// a duplicate email to verify that unique email constraints are
// being enforced.
func testUserDuplicate(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.User{}
		if err := unmarshal("testdata/users.json", &vv); err != nil {
			t.Error(err)
			return
		}
		if err := store.CreateUser(context.Background(), vv[0]); err == nil {
			t.Errorf("Expect unique index violation")
		}
	}
}

// this test counts the number of users in the database
// and compares to the expected count.
func testUserCount(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		got, err := store.CountUsers(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		if want := int64(2); got != want {
			t.Errorf("Want user count %d, got %d", want, got)
		}
	}
}

// this test fetches users from the database by id and key
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
//
//nolint:gocognit // test method, keep for now
func testUserFind(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		vv := []*types.User{}
		if err := unmarshal("testdata/users.json", &vv); err != nil {
			t.Error(err)
			return
		}
		want := vv[0]

		t.Run("id", func(t *testing.T) {
			got, err := store.FindUser(ctx, 1)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("uid", func(t *testing.T) {
			got, err := store.FindUserByUID(ctx, "jane21")
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("email", func(t *testing.T) {
			got, err := store.FindUserByEmail(ctx, want.Email)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("email/nocase", func(t *testing.T) {
			got, err := store.FindUserByEmail(ctx, strings.ToUpper(want.Email))
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})
	}
}

// this test fetches a list of users from the database
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
func testUserList(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		want := []*types.User{}
		if err := unmarshal("testdata/users.json", &want); err != nil {
			t.Error(err)
			return
		}
		got, err := store.ListUsers(context.Background(), &types.UserFilter{Page: 0, Size: 100})
		if err != nil {
			t.Error(err)
			return
		}
		if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}
	}
}

// this test updates an user in the database and then fetches
// the user and confirms the column was updated as expected.
func testUserUpdate(store store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		before, err := store.FindUser(ctx, 1)
		if err != nil {
			t.Error(err)
			return
		}
		before.Updated = time.Now().Unix()
		if err = store.UpdateUser(ctx, before); err != nil {
			t.Error(err)
			return
		}
		after, err := store.FindUser(ctx, 1)
		if err != nil {
			t.Error(err)
			return
		}

		if diff := cmp.Diff(before, after, userIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}
	}
}

// this test deletes an user from the database and then confirms
// subsequent attempts to fetch the deleted user result in
// a sql.ErrNoRows error.
func testUserDelete(s store.PrincipalStore) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		_, err := s.Find(ctx, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if err = s.DeleteUser(ctx, 1); err != nil {
			t.Error(err)
			return
		}
		if _, err = s.Find(ctx, 1); !errors.Is(err, gitness_store.ErrResourceNotFound) {
			t.Errorf("Expected store.ErrResourceNotFound got %s", err)
		}
	}
}
