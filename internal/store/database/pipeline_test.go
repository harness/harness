// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/harness/scm/internal/store"
	"github.com/harness/scm/types"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/jmoiron/sqlx"
)

// pipeline fields to ignore in test comparisons
var pipelineIgnore = cmpopts.IgnoreFields(types.Pipeline{},
	"ID", "Token", "Created", "Updated")

func TestPipeline(t *testing.T) {
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

	if _, err := newUserStoreSeeded(db); err != nil {
		t.Error(err)
		return
	}

	store := NewPipelineStoreSync(NewPipelineStore(db))
	t.Run("create", testPipelineCreate(store))
	t.Run("find", testPipelineFind(store))
	t.Run("list", testPipelineList(store))
	t.Run("update", testPipelineUpdate(store))
	t.Run("delete", testPipelineDelete(store))
}

// this test creates entries in the database and confirms
// the primary keys were auto-incremented.
func testPipelineCreate(store store.PipelineStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.Pipeline{}
		if err := unmarshal("testdata/pipelines.json", &vv); err != nil {
			t.Error(err)
			return
		}
		// create row 1
		v := vv[0]
		// generate a deterministic token for each
		// entry based on the hash of the email.
		v.Token = fmt.Sprintf("%x", v.Slug)
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}
		// create row 2
		v = vv[1]
		v.Token = fmt.Sprintf("%x", v.Slug)
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}

		t.Run("duplicate slug", func(t *testing.T) {
			v.ID = 0
			v.Token = "9afeab83324a53"
			v.Slug = "cassini"
			if err := store.Create(noContext, v); err == nil {
				t.Errorf("Expect duplicate row error")
				return
			}
		})

		t.Run("duplicate token", func(t *testing.T) {
			v.ID = 0
			v.Slug = "voyager"
			v.Token = "63617373696e69"
			if err := store.Create(noContext, v); err == nil {
				t.Errorf("Expect duplicate row error")
				return
			}
		})
	}
}

// this test fetches pipelines from the database by id and key
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
func testPipelineFind(store store.PipelineStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.Pipeline{}
		if err := unmarshal("testdata/pipelines.json", &vv); err != nil {
			t.Error(err)
			return
		}
		want := vv[0]
		want.Token = "63617373696e69"

		// Find row by ID
		got, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if diff := cmp.Diff(got, want, pipelineIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}

		t.Run("token", func(t *testing.T) {
			got, err := store.FindToken(noContext, want.Token)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, pipelineIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("slug", func(t *testing.T) {
			got, err := store.FindSlug(noContext, want.Slug)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, pipelineIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("slug", func(t *testing.T) {
			got, err := store.FindSlug(noContext, want.Slug)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, pipelineIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})
	}
}

// this test fetches a list of pipelines from the database
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
func testPipelineList(store store.PipelineStore) func(t *testing.T) {
	return func(t *testing.T) {
		want := []*types.Pipeline{}
		if err := unmarshal("testdata/pipelines.json", &want); err != nil {
			t.Error(err)
			return
		}
		got, err := store.List(noContext, 2, types.Params{Page: 0, Size: 100})
		if err != nil {
			t.Error(err)
			return
		}
		if len(got) != 2 {
			t.Errorf("Expect 2 pipelines")
		}
		if diff := cmp.Diff(got, want, pipelineIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}
	}
}

// this test updates an pipeline in the database and then fetches
// the pipeline and confirms the column was updated as expected.
func testPipelineUpdate(store store.PipelineStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		before.Updated = time.Now().Unix()
		before.Active = false
		if err := store.Update(noContext, before); err != nil {
			t.Error(err)
			return
		}
		after, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}

		if diff := cmp.Diff(before, after, pipelineIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}
	}
}

// this test deletes an pipeline from the database and then confirms
// subsequent attempts to fetch the deleted pipeline result in
// a sql.ErrNoRows error.
func testPipelineDelete(store store.PipelineStore) func(t *testing.T) {
	return func(t *testing.T) {
		v, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if err := store.Delete(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if _, err := store.Find(noContext, 1); err != sql.ErrNoRows {
			t.Errorf("Expected sql.ErrNoRows got %s", err)
		}
	}
}

// helper function that returns an pipeline store that is seeded
// with pipeline data loaded from a json file.
func newPipelineStoreSeeded(db *sqlx.DB) (store.PipelineStore, error) {
	store := NewPipelineStoreSync(NewPipelineStore(db))
	vv := []*types.Pipeline{}
	if err := unmarshal("testdata/pipelines.json", &vv); err != nil {
		return nil, err
	}
	for _, v := range vv {
		v.Token = fmt.Sprintf("%x", v.Slug)
		if err := store.Create(noContext, v); err != nil {
			return nil, err
		}
	}
	return store, nil
}
