// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package user

import (
	"context"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db/dbtest"
	"github.com/drone/drone/store/shared/encrypt"
)

var noContext = context.TODO()

func TestUser(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	store := New(conn, nil).(*userStore)
	store.enc, _ = encrypt.New("fb4b4d6267c8a5ce8231f8b186dbca92")
	t.Run("Create", testUserCreate(store))
}

func testUserCreate(store *userStore) func(t *testing.T) {
	return func(t *testing.T) {
		user := &core.User{
			Login:   "octocat",
			Email:   "octocat@github.com",
			Avatar:  "https://avatars3.githubusercontent.com/u/583231?v=4",
			Hash:    "MjAxOC0wOC0xMVQxNTo1ODowN1o",
			Token:   "9595fe015ca9b98c41ebf4e7d4e004ee",
			Refresh: "268ef49df64ea8ff79ef11e995d41aed",
		}
		err := store.Create(noContext, user)
		if err != nil {
			t.Error(err)
		}
		if user.ID == 0 {
			t.Errorf("Want user ID assigned, got %d", user.ID)
		}

		t.Run("Count", testUserCount(store))
		t.Run("Find", testUserFind(store, user))
		t.Run("FindLogin", testUserFindLogin(store))
		t.Run("FindToken", testUserFindToken(store))
		t.Run("List", testUserList(store))
		t.Run("Update", testUserUpdate(store, user))
		t.Run("Delete", testUserDelete(store, user))
	}
}

func testUserCount(users *userStore) func(t *testing.T) {
	return func(t *testing.T) {
		count, err := users.Count(noContext)
		if err != nil {
			t.Error(err)
		}
		if got, want := count, int64(1); got != want {
			t.Errorf("Want user table count %d, got %d", want, got)
		}

		count, err = users.CountHuman(noContext)
		if err != nil {
			t.Error(err)
		}
		if got, want := count, int64(1); got != want {
			t.Errorf("Want user table count %d for humans, got %d", want, got)
		}
	}
}

func testUserFind(users *userStore, created *core.User) func(t *testing.T) {
	return func(t *testing.T) {
		user, err := users.Find(noContext, created.ID)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testUser(user))
		}
	}
}

func testUserFindLogin(users *userStore) func(t *testing.T) {
	return func(t *testing.T) {
		user, err := users.FindLogin(noContext, "octocat")
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testUser(user))
		}
	}
}

func testUserFindToken(users *userStore) func(t *testing.T) {
	return func(t *testing.T) {
		user, err := users.FindToken(noContext, "MjAxOC0wOC0xMVQxNTo1ODowN1o")
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testUser(user))
		}
	}
}

func testUserList(users *userStore) func(t *testing.T) {
	return func(t *testing.T) {
		users, err := users.List(noContext)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(users), 1; got != want {
			t.Errorf("Want user count %d, got %d", want, got)
		} else {
			t.Run("Fields", testUser(users[0]))
		}
	}
}

func testUserUpdate(users *userStore, created *core.User) func(t *testing.T) {
	return func(t *testing.T) {
		user := &core.User{
			ID:     created.ID,
			Login:  "octocat",
			Email:  "noreply@github.com",
			Avatar: "https://avatars3.githubusercontent.com/u/583231?v=4",
		}
		err := users.Update(noContext, user)
		if err != nil {
			t.Error(err)
			return
		}
		updated, err := users.Find(noContext, user.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := updated.Email, user.Email; got != want {
			t.Errorf("Want updated user Email %q, got %q", want, got)
		}
	}
}

func testUserDelete(users *userStore, created *core.User) func(t *testing.T) {
	return func(t *testing.T) {
		count, _ := users.Count(noContext)
		if got, want := count, int64(1); got != want {
			t.Errorf("Want user table count %d, got %d", want, got)
			return
		}

		err := users.Delete(noContext, &core.User{ID: created.ID})
		if err != nil {
			t.Error(err)
		}

		count, _ = users.Count(noContext)
		if got, want := count, int64(0); got != want {
			t.Errorf("Want user table count %d, got %d", want, got)
			return
		}
	}
}

func testUser(user *core.User) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := user.Login, "octocat"; got != want {
			t.Errorf("Want user Login %q, got %q", want, got)
		}
		if got, want := user.Email, "octocat@github.com"; got != want {
			t.Errorf("Want user Email %q, got %q", want, got)
		}
		if got, want := user.Avatar, "https://avatars3.githubusercontent.com/u/583231?v=4"; got != want {
			t.Errorf("Want user Avatar %q, got %q", want, got)
		}
		if got, want := user.Token, "9595fe015ca9b98c41ebf4e7d4e004ee"; got != want {
			t.Errorf("Want user Access Token %q, got %q", want, got)
		}
		if got, want := user.Refresh, "268ef49df64ea8ff79ef11e995d41aed"; got != want {
			t.Errorf("Want user Refresh Token %q, got %q", want, got)
		}
	}
}

// The purpose of this unit test is to ensure that plaintext
// data can still be read from the database if encryption is
// added at a later time.
func TestUserCryptoCompat(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	store := New(conn, nil).(*userStore)
	store.enc, _ = encrypt.New("")

	item := &core.User{
		Login:   "octocat",
		Email:   "octocat@github.com",
		Avatar:  "https://avatars3.githubusercontent.com/u/583231?v=4",
		Hash:    "MjAxOC0wOC0xMVQxNTo1ODowN1o",
		Token:   "9595fe015ca9b98c41ebf4e7d4e004ee",
		Refresh: "268ef49df64ea8ff79ef11e995d41aed",
	}

	// create the secret with the secret value stored as plaintext
	err = store.Create(noContext, item)
	if err != nil {
		t.Error(err)
		return
	}
	if item.ID == 0 {
		t.Errorf("Want secret ID assigned, got %d", item.ID)
		return
	}

	// update the store to use encryption
	store.enc, _ = encrypt.New("fb4b4d6267c8a5ce8231f8b186dbca92")
	store.enc.(*encrypt.Aesgcm).Compat = true

	// fetch the secret from the database
	got, err := store.Find(noContext, item.ID)
	if err != nil {
		t.Errorf("cannot retrieve user from database: %s", err)
	} else {
		t.Run("Fields", testUser(got))
	}
}
