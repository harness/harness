// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jmoiron/sqlx"
)

// user fields to ignore in test comparisons
var userIgnore = cmpopts.IgnoreFields(types.User{},
	"ID", "Salt", "Created", "Updated")

func TestUser(t *testing.T) {
	db, err := connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer db.Close()
	if err := seed(db); err != nil {
		t.Error(err)
		return
	}

	store := NewUserStoreSync(NewUserStore(db))
	t.Run("create", testUserCreate(store))
	t.Run("duplicate", testUserDuplicate(store))
	t.Run("count", testUserCount(store))
	t.Run("find", testUserFind(store))
	t.Run("list", testUserList(store))
	t.Run("update", testUserUpdate(store))
	t.Run("delete", testUserDelete(store))
}

// this test creates entries in the database and confirms
// the primary keys were auto-incremented.
func testUserCreate(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
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
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}
		// create row 2
		v = vv[1]
		v.Salt = fmt.Sprintf("%x", v.Email)
		if err := store.Create(noContext, v); err != nil {
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
func testUserDuplicate(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.User{}
		if err := unmarshal("testdata/users.json", &vv); err != nil {
			t.Error(err)
			return
		}
		if err := store.Create(noContext, vv[0]); err == nil {
			t.Errorf("Expect unique index violation")
		}
	}
}

// this test counts the number of users in the database
// and compares to the expected count.
func testUserCount(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		got, err := store.Count(noContext)
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
func testUserFind(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.User{}
		if err := unmarshal("testdata/users.json", &vv); err != nil {
			t.Error(err)
			return
		}
		want := vv[0]

		t.Run("id", func(t *testing.T) {
			got, err := store.Find(noContext, 1)
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
			got, err := store.FindEmail(noContext, want.Email)
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
			got, err := store.FindEmail(noContext, strings.ToUpper(want.Email))
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("key/id", func(t *testing.T) {
			got, err := store.FindKey(noContext, "1")
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, userIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("key/email", func(t *testing.T) {
			got, err := store.FindKey(noContext, want.Email)
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
func testUserList(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		want := []*types.User{}
		if err := unmarshal("testdata/users.json", &want); err != nil {
			t.Error(err)
			return
		}
		got, err := store.List(noContext, &types.UserFilter{Page: 0, Size: 100})
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
func testUserUpdate(store store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		before.Updated = time.Now().Unix()
		before.Authed = time.Now().Unix()
		if err := store.Update(noContext, before); err != nil {
			t.Error(err)
			return
		}
		after, err := store.Find(noContext, 1)
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
func testUserDelete(s store.UserStore) func(t *testing.T) {
	return func(t *testing.T) {
		v, err := s.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if err := s.Delete(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if _, err := s.Find(noContext, 1); err != store.ErrResourceNotFound {
			t.Errorf("Expected sql.ErrNoRows got %s", err)
		}
	}
}

// helper function that returns an user store that is seeded
// with user data loaded from a json file.
func newUserStoreSeeded(db *sqlx.DB) (store.UserStore, error) {
	store := NewUserStoreSync(NewUserStore(db))
	vv := []*types.User{}
	if err := unmarshal("testdata/users.json", &vv); err != nil {
		return nil, err
	}
	for _, v := range vv {
		v.Salt = fmt.Sprintf("%x", v.Email)
		if err := store.Create(noContext, v); err != nil {
			return nil, err
		}
	}
	return store, nil
}
