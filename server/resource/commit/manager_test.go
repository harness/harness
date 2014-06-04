package commit

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/resource/commit/committest"
	_ "github.com/mattn/go-sqlite3"
)

// in-memory database instance for unit testing
var db *sql.DB

// setup the test database and test fixtures
func setup() {
	db, _ = sql.Open("sqlite3", ":memory:")
	database.Load(db)
	committest.Load(db)
}

// teardown the test database
func teardown() {
	db.Close()
}

func TestFind(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	commit, err := commits.Find(3)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	testCommit(t, commit)
}

func TestFindSha(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	commit, err := commits.FindSha(2, "master", "7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf")
	if err != nil {
		t.Errorf("Want Commit from SHA, got %s", err)
	}

	testCommit(t, commit)
}

func TestFindLatest(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	commit, err := commits.FindLatest(2, "master")
	if err != nil {
		t.Errorf("Want Latest Commit, got %s", err)
	}

	testCommit(t, commit)
}

func TestList(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	list, err := commits.List(2)
	if err != nil {
		t.Errorf("Want List from RepoID, got %s", err)
	}

	var got, want = len(list), 3
	if got != want {
		t.Errorf("Want List size %v, got %v", want, got)
	}

	testCommit(t, list[0])
}

func TestListBranch(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	list, err := commits.ListBranch(2, "master")
	if err != nil {
		t.Errorf("Want List from RepoID, got %s", err)
	}

	var got, want = len(list), 2
	if got != want {
		t.Errorf("Want List size %v, got %v", want, got)
	}

	testCommit(t, list[0])
}

func TestListBranches(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	list, err := commits.ListBranches(2)
	if err != nil {
		t.Errorf("Want Branch List from RepoID, got %s", err)
	}

	var got, want = len(list), 2
	if got != want {
		t.Errorf("Want List size %v, got %v", want, got)
	}

	testCommit(t, list[1])
}

func TestInsert(t *testing.T) {
	setup()
	defer teardown()

	commit := Commit{RepoID: 3, Branch: "foo", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"}
	commits := NewManager(db)
	if err := commits.Insert(&commit); err != nil {
		t.Errorf("Want Commit created, got %s", err)
	}

	// verify that it is ok to add same sha for different branch
	var err = commits.Insert(&Commit{RepoID: 3, Branch: "bar", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"})
	if err != nil {
		t.Errorf("Want Commit created, got %s", err)
	}

	// verify unique remote + remote id constraint
	err = commits.Insert(&Commit{RepoID: 3, Branch: "bar", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"})
	if err == nil || !strings.Contains(err.Error(), "commit_sha, commit_branch, repo_id are not unique") {
		t.Errorf("Want unique constraint violated, got %s", err)
	}

}

func TestUpdate(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	commit, err := commits.Find(5)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	// update the commit's access token
	commit.Status = "Success"
	commit.Finished = time.Now().Unix()
	commit.Duration = 999
	if err := commits.Update(commit); err != nil {
		t.Errorf("Want Commit updated, got %s", err)
	}

	updated, _ := commits.Find(5)
	var got, want = updated.Status, "Success"
	if got != want {
		t.Errorf("Want updated Status %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = updated.ID, commit.ID
	if gotInt64 != wantInt64 {
		t.Errorf("Want commit ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = updated.Duration, commit.Duration
	if gotInt64 != wantInt64 {
		t.Errorf("Want updated Duration %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = updated.Finished, commit.Finished
	if gotInt64 != wantInt64 {
		t.Errorf("Want updated Finished %v, got %v", wantInt64, gotInt64)
	}
}

func TestDelete(t *testing.T) {
	setup()
	defer teardown()

	commits := NewManager(db)
	commit, err := commits.Find(1)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	// delete the commit
	if err := commits.Delete(commit); err != nil {
		t.Errorf("Want Commit deleted, got %s", err)
	}

	// check to see if the deleted commit is actually gone
	if _, err := commits.Find(1); err != sql.ErrNoRows {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

// testCommit is a helper function that compares the commit
// to an expected set of fixed field values.
func testCommit(t *testing.T, commit *Commit) {
	var got, want = commit.Status, "Success"
	if got != want {
		t.Errorf("Want Status %v, got %v", want, got)
	}

	got, want = commit.Sha, "7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf"
	if got != want {
		t.Errorf("Want Sha %v, got %v", want, got)
	}

	got, want = commit.Branch, "master"
	if got != want {
		t.Errorf("Want Branch %v, got %v", want, got)
	}

	got, want = commit.PullRequest, "5"
	if got != want {
		t.Errorf("Want PullRequest %v, got %v", want, got)
	}

	got, want = commit.Author, "drcooper@caltech.edu"
	if got != want {
		t.Errorf("Want Author %v, got %v", want, got)
	}

	got, want = commit.Gravatar, "ab23a88a3ed77ecdfeb894c0eaf2817a"
	if got != want {
		t.Errorf("Want Gravatar %v, got %v", want, got)
	}

	got, want = commit.Timestamp, "Wed Apr 23 01:02:38 2014 -0700"
	if got != want {
		t.Errorf("Want Timestamp %v, got %v", want, got)
	}

	got, want = commit.Message, "a commit message"
	if got != want {
		t.Errorf("Want Message %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = commit.ID, int64(3)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.RepoID, int64(2)
	if gotInt64 != wantInt64 {
		t.Errorf("Want RepoID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.Created, int64(1398065343)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Created %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.Updated, int64(1398065344)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Updated %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.Started, int64(1398065345)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Started %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.Finished, int64(1398069999)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Finished %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = commit.Duration, int64(854)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Duration %v, got %v", wantInt64, gotInt64)
	}
}
