package database

import (
	"database/sql"
	"testing"
	"time"

	"github.com/drone/drone/shared/model"
)

func TestCommitFind(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
	commit, err := commits.Find(3)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	testCommit(t, commit)
}

func TestCommitFindSha(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
	commit, err := commits.FindSha(2, "master", "7253f6545caed41fb8f5a6fcdb3abc0b81fa9dbf")
	if err != nil {
		t.Errorf("Want Commit from SHA, got %s", err)
	}

	testCommit(t, commit)
}

func TestCommitFindLatest(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
	commit, err := commits.FindLatest(2, "master")
	if err != nil {
		t.Errorf("Want Latest Commit, got %s", err)
	}

	testCommit(t, commit)
}

func TestCommitFindOutput(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
	out, err := commits.FindOutput(1)
	if err != nil {
		t.Errorf("Want Commit stdout, got %s", err)
	}

	var want, got = "sample console output", string(out)
	if want != got {
		t.Errorf("Want stdout %v, got %v", want, got)
	}
}

func TestCommitList(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
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

func TestCommitListBranch(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
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

func TestCommitListBranches(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
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

func TestCommitInsert(t *testing.T) {
	setup()
	defer teardown()

	commit := model.Commit{RepoID: 3, Branch: "foo", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"}
	commits := NewCommitManager(db)
	if err := commits.Insert(&commit); err != nil {
		t.Errorf("Want Commit created, got %s", err)
	}

	// verify that it is ok to add same sha for different branch
	var err = commits.Insert(&model.Commit{RepoID: 3, Branch: "bar", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"})
	if err != nil {
		t.Errorf("Want Commit created, got %s", err)
	}

	// verify unique remote + remote id constraint
	err = commits.Insert(&model.Commit{RepoID: 3, Branch: "bar", Sha: "85f8c029b902ed9400bc600bac301a0aadb144ac"})
	if err == nil {
		t.Error("Want unique constraint violated")
	}

}

func TestCommitUpdate(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
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

func TestCommitDelete(t *testing.T) {
	setup()
	defer teardown()

	commits := NewCommitManager(db)
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
func testCommit(t *testing.T, commit *model.Commit) {
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
