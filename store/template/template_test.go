// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package template

import (
	"context"
	"database/sql"
	"github.com/drone/drone/core"
	"github.com/drone/drone/store/repos"
	"github.com/drone/drone/store/shared/db/dbtest"
	"testing"
)

var noContext = context.TODO()

func TestTemplate(t *testing.T) {
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

	store := New(conn).(*templateStore)
	t.Run("Create", testTemplateCreate(store, repos, repo))
}

func testTemplateCreate(store *templateStore, repos core.RepositoryStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Template{
			Id:      repo.ID,
			Name:    "my_template",
			Data:    "some_template_data",
			Created: 1,
			Updated: 2,
		}
		err := store.Create(noContext, item)
		if err != nil {
			t.Error(err)
		}
		if item.Id == 0 {
			t.Errorf("Want template Id assigned, got %d", item.Id)
		}

		t.Run("Find", testTemplateFind(store, item))
		t.Run("FindName", testTemplateFindName(store, repo))
		t.Run("List", testTemplateList(store, repo))
		t.Run("Update", testTemplateUpdate(store, repo))
		t.Run("Delete", testTemplateDelete(store, repo))
	}
}

func testTemplateFind(store *templateStore, template *core.Template) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.Find(noContext, template.Id)
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testTemplate(item))
		}
	}
}

func testTemplateFindName(store *templateStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.FindName(noContext, repo.ID, "my_template")
		if err != nil {
			t.Error(err)
		} else {
			t.Run("Fields", testTemplate(item))
		}
	}
}

func testTemplate(item *core.Template) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := item.Name, "my_template"; got != want {
			t.Errorf("Want template name %q, got %q", want, got)
		}
		if got, want := item.Data, "some_template_data"; got != want {
			t.Errorf("Want template data %q, got %q", want, got)
		}
	}
}

func testTemplateList(store *templateStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		list, err := store.List(noContext, repo.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(list), 1; got != want {
			t.Errorf("Want count %d, got %d", want, got)
		} else {
			t.Run("Fields", testTemplate(list[0]))
		}
	}
}

func testTemplateUpdate(store *templateStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.FindName(noContext, repo.ID, "my_template")
		if err != nil {
			t.Error(err)
			return
		}
		err = store.Update(noContext, before)
		if err != nil {
			t.Error(err)
			return
		}
		after, err := store.Find(noContext, before.Id)
		if err != nil {
			t.Error(err)
			return
		}
		if after == nil {
			t.Fail()
		}
	}
}

func testTemplateDelete(store *templateStore, repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		secret, err := store.FindName(noContext, repo.ID, "my_template")
		if err != nil {
			t.Error(err)
			return
		}
		err = store.Delete(noContext, secret)
		if err != nil {
			t.Error(err)
			return
		}
		_, err = store.Find(noContext, secret.Id)
		if got, want := sql.ErrNoRows, err; got != want {
			t.Errorf("Want sql.ErrNoRows, got %v", got)
			return
		}
	}
}
