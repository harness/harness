// Copyright 2021 Harness Inc. All rights reserved.
// Use of this source code is governed by the Polyform Free Trial License
// that can be found in the LICENSE.md file for this repository.

package database

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/harness/gitness/internal/store"
	"github.com/harness/gitness/types"
	"github.com/jmoiron/sqlx"
)

// execution fields to ignore in test comparisons
var executionIgnore = cmpopts.IgnoreFields(types.Execution{},
	"ID", "Created", "Updated")

func TestExecution(t *testing.T) {
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

	if _, err := newPipelineStoreSeeded(db); err != nil {
		t.Error(err)
		return
	}

	store := NewExecutionStoreSync(NewExecutionStore(db))
	t.Run("create", testExecutionCreate(store))
	t.Run("find", testExecutionFind(store))
	t.Run("list", testExecutionList(store))
	t.Run("update", testExecutionUpdate(store))
	t.Run("delete", testExecutionDelete(store))
}

// this test creates entries in the database and confirms
// the primary keys were auto-incremented.
func testExecutionCreate(store store.ExecutionStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.Execution{}
		if err := unmarshal("testdata/executions.json", &vv); err != nil {
			t.Error(err)
			return
		}

		// create row 1
		v := vv[0]
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}
		if v.ID == 0 {
			t.Errorf("Want autoincremented primary key")
		}
		// create row 2
		v = vv[1]
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}
		// create row 3
		v = vv[2]
		if err := store.Create(noContext, v); err != nil {
			t.Error(err)
			return
		}

		t.Run("duplicate slug", func(t *testing.T) {
			// reset the ID so that a new row is created
			// using the same slug
			v.ID = 0
			if err := store.Create(noContext, v); err == nil {
				t.Errorf("Expect duplicate row error")
				return
			}
		})
	}
}

// this test fetches executions from the database by id and key
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
func testExecutionFind(store store.ExecutionStore) func(t *testing.T) {
	return func(t *testing.T) {
		vv := []*types.Execution{}
		if err := unmarshal("testdata/executions.json", &vv); err != nil {
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
			if diff := cmp.Diff(got, want, executionIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})

		t.Run("slug", func(t *testing.T) {
			got, err := store.FindSlug(noContext, want.Pipeline, want.Slug)
			if err != nil {
				t.Error(err)
				return
			}
			if diff := cmp.Diff(got, want, executionIgnore); len(diff) != 0 {
				t.Errorf(diff)
				return
			}
		})
	}
}

// this test fetches a list of executions from the database
// and compares to the expected results (sourced from a json file)
// to ensure all columns are correctly mapped.
func testExecutionList(store store.ExecutionStore) func(t *testing.T) {
	return func(t *testing.T) {
		want := []*types.Execution{}
		if err := unmarshal("testdata/executions.json", &want); err != nil {
			t.Error(err)
			return
		}
		got, err := store.List(noContext, 2, types.Params{Size: 25, Page: 0})
		if err != nil {
			t.Error(err)
			return
		}

		if diff := cmp.Diff(got, want[1:], executionIgnore); len(diff) != 0 {
			t.Errorf(diff)
			debug(t, got)
			return
		}
	}
}

// this test updates a execution in the database and then fetches
// the execution and confirms the column was updated as expected.
func testExecutionUpdate(store store.ExecutionStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		before.Desc = "updated description"
		if err := store.Update(noContext, before); err != nil {
			t.Error(err)
			return
		}
		after, err := store.Find(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}

		if diff := cmp.Diff(before, after, executionIgnore); len(diff) != 0 {
			t.Errorf(diff)
			return
		}
	}
}

// this test deletes a execution from the database and then confirms
// subsequent attempts to fetch the deleted execution result in
// a sql.ErrNoRows error.
func testExecutionDelete(store store.ExecutionStore) func(t *testing.T) {
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

// helper function that returns an execution store that is seeded
// with execution data loaded from a json file.
func newExecutionStoreSeeded(db *sqlx.DB) (store.ExecutionStore, error) {
	store := NewExecutionStoreSync(NewExecutionStore(db))
	vv := []*types.Execution{}
	if err := unmarshal("testdata/executions.json", &vv); err != nil {
		return nil, err
	}
	for _, v := range vv {
		if err := store.Create(noContext, v); err != nil {
			return nil, err
		}
	}
	return store, nil
}
