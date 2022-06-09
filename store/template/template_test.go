// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

//go:build !oss
// +build !oss

package template

import (
	"context"
	"database/sql"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db/dbtest"
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

	store := New(conn).(*templateStore)
	t.Run("TestTemplates", testTemplateCreate(store))
}

func testTemplateCreate(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Template{
			Id:        1,
			Name:      "my_template",
			Namespace: "my_org",
			Data:      "some_template_data",
			Created:   1,
			Updated:   2,
		}
		err := store.Create(noContext, item)
		if err != nil {
			t.Error(err)
		}
		if item.Id == 0 {
			t.Errorf("Want template Id assigned, got %d", item.Id)
		}

		t.Run("CreateSameNameDiffOrg", testCreateWithSameNameDiffOrg(store))
		t.Run("CreateSameNameSameOrgShouldError", testCreateSameNameSameOrgShouldError(store))
		t.Run("Find", testTemplateFind(store, item))
		t.Run("FindName", testTemplateFindName(store))
		t.Run("ListAll", testTemplateListAll(store))
		t.Run("List", testTemplateList(store))
		t.Run("Update", testTemplateUpdate(store))
		t.Run("Delete", testTemplateDelete(store))
	}
}

func testCreateWithSameNameDiffOrg(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Template{
			Id:        1,
			Name:      "my_template",
			Namespace: "my_org2",
			Data:      "some_template_data",
			Created:   1,
			Updated:   2,
		}
		err := store.Create(noContext, item)
		if err != nil {
			t.Error(err)
		}
		if item.Id == 0 {
			t.Errorf("Want template Id assigned, got %d", item.Id)
		}
	}
}

func testCreateSameNameSameOrgShouldError(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		item := &core.Template{
			Id:        3,
			Name:      "my_template",
			Namespace: "my_org2",
			Data:      "some_template_data",
			Created:   1,
			Updated:   2,
		}
		err := store.Create(noContext, item)
		if err == nil {
			t.Error(err)
		}
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

func testTemplateFindName(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		item, err := store.FindName(noContext, "my_template", "my_org")
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
		if got, want := item.Namespace, "my_org"; got != want {
			t.Errorf("Want template org %q, got %q", want, got)
		}
	}
}

func testTemplate2(item *core.Template) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := item.Name, "my_template"; got != want {
			t.Errorf("Want template name %q, got %q", want, got)
		}
		if got, want := item.Data, "some_template_data"; got != want {
			t.Errorf("Want template data %q, got %q", want, got)
		}
		if got, want := item.Namespace, "my_org2"; got != want {
			t.Errorf("Want template org %q, got %q", want, got)
		}
	}
}

func testTemplateListAll(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		list, err := store.ListAll(noContext)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(list), 2; got != want {
			t.Errorf("Want count %d, got %d", want, got)
		} else {
			t.Run("Fields", testTemplate(list[0]))
			t.Run("Fields", testTemplate2(list[1]))
		}
	}
}

func testTemplateList(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		list, err := store.List(noContext, "my_org")
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

func testTemplateUpdate(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.FindName(noContext, "my_template", "my_org")
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

func testTemplateDelete(store *templateStore) func(t *testing.T) {
	return func(t *testing.T) {
		secret, err := store.FindName(noContext, "my_template", "my_org")
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
