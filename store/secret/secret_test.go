// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package secret

import (
	"context"
	"database/sql"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/repos"
	"github.com/drone/drone/store/shared/db/dbtest"
	"github.com/drone/drone/store/shared/encrypt"
)

var noContext = context.TODO()

func TestSecret(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	// seeds the database with a dummy repository.
	repo := &core.Repository{UID: "1", Slug: "octocat/hello-world"}
	repos := repos.New(conn)
	if err := repos.Create(noContext, repo); err != nil {
		t.Error(err)
	}

	store := New(conn, nil).(*secretStore)
	store.enc, _ = encrypt.New("fb4b4d6267c8a5ce8231f8b186dbca92")
	t.Run("Create", testSecretCreate(store, repos, repo))
}

func testSecretCreate(store *secretStore, repos core.RepositoryStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Secret{
			RepoID: repo.ID,
			Name:   "password",
			Data:   "correct-horse-battery-staple",
		}
		err := store.Create(noContext, item)
		if err != nil {
			t.Error(err)
		}
		if item.ID == 0 {
			t.Errorf("Want secret ID assigned, got %d", item.ID)
		}

		t.Run("Find", testSecretFind(store, item))
		t.Run("FindName", testSecretFindName(store, repo))
		t.Run("List", testSecretList(store, repo))
		t.Run("Update", testSecretUpdate(store, repo))
		t.Run("Delete", testSecretDelete(store, repo))
		t.Run("Fkey", testSecretForeignKey(store, repos, repo))
	}
}

func testSecretFind(store *secretStore, secret *core.Secret) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.Find(noContext, secret.ID)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testSecret(item))
		}
	}
}

func testSecretFindName(store *secretStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.FindName(noContext, repo.ID, "password")
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testSecret(item))
		}
	}
}

func testSecretList(store *secretStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		list, err := store.List(noContext, repo.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(list), 1; got != want {
			t.Errorf("Want count %d, got %d", want, got)
		} else {
			t.Run("Fields", testSecret(list[0]))
		}
	}
}

func testSecretUpdate(store *secretStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.FindName(noContext, repo.ID, "password")
		if err != nil {
			t.Error(err)
			return
		}
		err = store.Update(noContext, before)
		if err != nil {
			t.Error(err)
			return
		}
		after, err := store.Find(noContext, before.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if after == nil {
			t.Fail()
		}
	}
}

func testSecretDelete(store *secretStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		secret, err := store.FindName(noContext, repo.ID, "password")
		if err != nil {
			t.Error(err)
			return
		}
		err = store.Delete(noContext, secret)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = store.Find(noContext, secret.ID)
		if got, want := sql.ErrNoRows, err; got != want {
			t.Errorf("Want sql.ErrNoRows, got %v", got)
			return
		}
	}
}

func testSecretForeignKey(store *secretStore, repos core.RepositoryStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Secret{
			RepoID: repo.ID,
			Name:   "password",
			Data:   "correct-horse-battery-staple",
		}
		store.Create(noContext, item)
		before, _ := store.List(noContext, repo.ID)
		if len(before) == 0 {
			t.Errorf("Want non-empty secret list")
			return
		}

		err := repos.Delete(noContext, repo)
		if err != nil {
			t.Error(err)
			return
		}
		after, _ := store.List(noContext, repo.ID)
		if len(after) != 0 {
			t.Errorf("Want empty secret list")
		}
	}
}

func testSecret(item *core.Secret) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := item.Name, "password"; got != want {
			t.Errorf("Want secret name %q, got %q", want, got)
		}
		if got, want := item.Data, "correct-horse-battery-staple"; got != want {
			t.Errorf("Want secret data %q, got %q", want, got)
		}
	}
}
