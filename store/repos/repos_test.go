// Copyright 2019 Drone.IO Inc. All rights reserved.
// Use of this source code is governed by the Drone Non-Commercial License
// that can be found in the LICENSE file.

// +build !oss

package repos

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/drone/drone/core"
	"github.com/drone/drone/store/shared/db"
	"github.com/drone/drone/store/shared/db/dbtest"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

var noContext = context.TODO()

func TestRepo(t *testing.T) {
	conn, err := dbtest.Connect()
	if err != nil {
		t.Error(err)
		return
	}
	defer func() {
		dbtest.Reset(conn)
		dbtest.Disconnect(conn)
	}()

	store := New(conn).(*repoStore)
	t.Run("Create", testRepoCreate(store))
	t.Run("Count", testRepoCount(store))
	t.Run("Find", testRepoFind(store))
	t.Run("FindName", testRepoFindName(store))
	t.Run("List", testRepoList(store))
	t.Run("ListLatest", testRepoListLatest(store))
	t.Run("ListLatestByCounter", testRepoListLatestByCounter(store))
	t.Run("Update", testRepoUpdate(store))
	t.Run("Activate", testRepoActivate(store))
	t.Run("Locking", testRepoLocking(store))
	t.Run("Increment", testRepoIncrement(store))
	t.Run("Delete", testRepoDelete(store))
}

func testRepoCreate(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		out, err := ioutil.ReadFile("testdata/repo.json")
		if err != nil {
			t.Error(err)
			return
		}
		repo := &core.Repository{}
		err = json.Unmarshal(out, repo)
		if err != nil {
			t.Error(err)
			return
		}
		err = repos.Create(noContext, repo)
		if err != nil {
			t.Error(err)
		}
		if got := repo.ID; got == 0 {
			t.Errorf("Want non-zero ID")
		}
		if got, want := repo.Version, int64(1); got != want {
			t.Errorf("Want Version %d, got %d", want, got)
		}

		err = repos.db.Update(func(execer db.Execer, binder db.Binder) error {
			query, args, _ := binder.BindNamed(stmtPermInsert, map[string]interface{}{
				"perm_user_id":  1,
				"perm_repo_uid": repo.UID,
				"perm_read":     true,
				"perm_write":    true,
				"perm_admin":    true,
				"perm_synced":   0,
				"perm_created":  0,
				"perm_updated":  0,
			})
			_, err = execer.Exec(query, args...)
			return err
		})
		if err != nil {
			t.Error(err)
		}
	}
}

func testRepoCount(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		count, err := repos.Count(noContext)
		if err != nil {
			t.Error(err)
		}
		if got, want := count, int64(1); got != want {
			t.Errorf("Want count %d, got %d", want, got)
		}
	}
}

func testRepoFind(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		named, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}
		got, err := repos.Find(noContext, named.ID)
		if err != nil {
			t.Error(err)
			return
		}

		want := &core.Repository{}
		raw, err := ioutil.ReadFile("testdata/repo.json.golden")
		if err != nil {
			t.Error(err)
			return
		}
		err = json.Unmarshal(raw, want)
		if err != nil {
			t.Error(err)
			return
		}

		ignore := cmpopts.IgnoreFields(core.Repository{}, "ID")
		if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
			t.Errorf("Diff: %s", diff)
		}
	}
}

func testRepoFindName(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		got, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}

		want := &core.Repository{}
		raw, err := ioutil.ReadFile("testdata/repo.json.golden")
		if err != nil {
			t.Error(err)
			return
		}
		err = json.Unmarshal(raw, want)
		if err != nil {
			t.Error(err)
			return
		}

		ignore := cmpopts.IgnoreFields(core.Repository{}, "ID")
		if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
			t.Errorf("Diff: %s", diff)
		}
	}
}

func testRepoList(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		repos, err := repos.List(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(repos), 1; got != want {
			t.Errorf("Want Repo count %d, got %d", want, got)
			return
		}

		if err != nil {
			t.Error(err)
			return
		}

		got, want := repos[0], &core.Repository{}
		raw, err := ioutil.ReadFile("testdata/repo.json.golden")
		if err != nil {
			t.Error(err)
			return
		}
		err = json.Unmarshal(raw, want)
		if err != nil {
			t.Error(err)
			return
		}

		ignore := cmpopts.IgnoreFields(core.Repository{}, "ID")
		if diff := cmp.Diff(got, want, ignore); len(diff) != 0 {
			t.Errorf("Diff: %s", diff)
		}
	}
}

func testRepoListLatest(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		repos, err := repos.ListLatest(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := len(repos), 1; got != want {
			t.Errorf("Want Repo count %d, got %d", want, got)
		} else if repos[0].Build != nil {
			t.Errorf("Expect nil build")
		} else {
			t.Run("Fields", testRepo(repos[0]))
		}
	}
}

// testRepoListLatestByCounter proves that queryRepoWithBuildPostgres
// resolves the repo's build via repo_counter, not via insertion order
// or build_id. It inserts the higher-numbered build first (so it gets
// the lower build_id) and the lower-numbered build second (so it gets
// the higher build_id) -- the old per-row subquery ordered by
// build_id DESC would have picked the wrong one.
func testRepoListLatestByCounter(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		if repos.db.Driver() != db.Postgres {
			t.Skip("queryRepoWithBuildPostgres only runs on the postgres driver")
		}

		repo, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}

		insert := func(number int64) {
			err := repos.db.Update(func(execer db.Execer, binder db.Binder) error {
				_, err := execer.Exec(
					`INSERT INTO builds (build_repo_id, build_number, build_status) VALUES ($1, $2, '')`,
					repo.ID, number,
				)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
		}
		defer func() {
			_ = repos.db.Update(func(execer db.Execer, binder db.Binder) error {
				_, err := execer.Exec(`DELETE FROM builds WHERE build_repo_id = $1`, repo.ID)
				return err
			})
		}()

		insert(7) // higher number, inserted (and therefore lower build_id) first
		insert(4) // lower number, inserted (and therefore higher build_id) second

		repo.Counter = 7
		if err := repos.Update(noContext, repo); err != nil {
			t.Error(err)
			return
		}

		list, err := repos.ListLatest(noContext, 1)
		if err != nil {
			t.Error(err)
			return
		}
		if len(list) != 1 || list[0].Build == nil {
			t.Fatalf("want exactly one repo with a build, got %+v", list)
		}
		if got, want := list[0].Build.Number, int64(7); got != want {
			t.Errorf("Want latest build number %d (repo_counter), got %d", want, got)
		}
	}
}

func testRepoUpdate(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}

		version := before.Version
		before.Private = true
		err = repos.Update(noContext, before)
		if err != nil {
			t.Error(err)
			return
		}
		after, err := repos.Find(noContext, before.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := after.Version, version+1; got != want {
			t.Errorf("Want version incremented on update")
		}
		if got, want := before.Private, after.Private; got != want {
			t.Errorf("Want updated Repo private %v, got %v", want, got)
		}
	}
}

func testRepoActivate(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		before, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}
		before.Active = true
		err = repos.Activate(noContext, before)
		if err != nil {
			t.Error(err)
			return
		}
		after, err := repos.Find(noContext, before.ID)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := before.Active, after.Active; got != want {
			t.Errorf("Want updated Repo Active %v, got %v", want, got)
		}
	}
}

func testRepoLocking(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		repo, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}
		repo.Version = 1
		err = repos.Update(noContext, repo)
		if err == nil {
			t.Errorf("Want Optimistic Lock Error, got nil")
		} else if err != db.ErrOptimisticLock {
			t.Errorf("Want Optimistic Lock Error")
		}
	}
}

func testRepoIncrement(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		repo, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			t.Error(err)
			return
		}
		before := repo.Counter
		repo.Version--
		repo, err = repos.Increment(noContext, repo)
		if err != nil {
			t.Error(err)
			return
		}
		if got, want := repo.Counter, before+1; got != want {
			t.Errorf("Want count incremented to %d, got %d", want, got)
		}
	}
}

func testRepoDelete(repos *repoStore) func(t *testing.T) {
	return func(t *testing.T) {
		count, _ := repos.Count(noContext)
		if got, want := count, int64(1); got != want {
			t.Errorf("Want Repo table count %d, got %d", want, got)
			return
		}

		repo, err := repos.FindName(noContext, "octocat", "hello-world")
		if err != nil {
			return
		}

		err = repos.Delete(noContext, repo)
		if err != nil {
			t.Error(err)
		}

		count, _ = repos.Count(noContext)
		if got, want := count, int64(0); got != want {
			t.Errorf("Want Repo table count %d, got %d", want, got)
			return
		}
	}
}

func testRepo(repo *core.Repository) func(t *testing.T) {
	return func(t *testing.T) {
		if got, want := repo.UserID, int64(1); got != want {
			t.Errorf("Want UserID %d, got %d", want, got)
		}
		if got, want := repo.Namespace, "octocat"; got != want {
			t.Errorf("Want Namespace %q, got %q", want, got)
		}
		if got, want := repo.Name, "hello-world"; got != want {
			t.Errorf("Want Name %q, got %q", want, got)
		}
		if got, want := repo.Slug, "octocat/hello-world"; got != want {
			t.Errorf("Want Slug %q, got %q", want, got)
		}
		if got, want := repo.UID, "42"; got != want {
			t.Errorf("Want UID %q, got %q", want, got)
		}
	}
}
