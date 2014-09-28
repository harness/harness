package database

import (
	//"database/sql"
	"testing"
	"time"

	"github.com/drone/drone/shared/model"
)

func TestBuildFind(t *testing.T) {
	setup()
	defer teardown()

	builds := NewBuildManager(db)
	build, err := builds.Find(1, 3)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	testBuild(t, build)
}

func TestBuildFindCommit(t *testing.T) {
	setup()
	defer teardown()

	builds := NewBuildManager(db)
	list, err := builds.FindCommit(3)
	if err != nil {
		t.Errorf("Want List from CommitID, got %s", err)
	}

	var got, want = len(list), 1
	if got != want {
		t.Errorf("Want List size %v, got %v", want, got)
	}

	testBuild(t, list[0])
}

func TestBuildUpdate(t *testing.T) {
	setup()
	defer teardown()

	builds := NewBuildManager(db)
	build, err := builds.Find(1, 3)
	if err != nil {
		t.Errorf("Want Commit from ID, got %s", err)
	}

	build.Status = "Success"
	build.Finished = time.Now().Unix()
	build.Duration = 999
	if err := builds.Update(build); err != nil {
		t.Errorf("Want Commit updated, got %s", err)
	}

	updated, _ := builds.Find(1, 3)
	var got, want = updated.Status, "Success"
	if got != want {
		t.Errorf("Want updated Status %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = updated.ID, build.ID
	if gotInt64 != wantInt64 {
		t.Errorf("Want build ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = updated.CommitID, build.CommitID
	if gotInt64 != wantInt64 {
		t.Errorf("Want commit ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = updated.Duration, build.Duration
	if gotInt64 != wantInt64 {
		t.Errorf("Want updated Duration %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = updated.Finished, build.Finished
	if gotInt64 != wantInt64 {
		t.Errorf("Want updated Finished %v, got %v", wantInt64, gotInt64)
	}
}

func TestBuildInsert(t *testing.T) {
	setup()
	defer teardown()

	build := model.Build{CommitID: 2, Index: 2}
	builds := NewBuildManager(db)
	if err := builds.Insert(&build); err != nil {
		t.Errorf("Want Build created, got %s", err)
	}

	// verify that it is ok to add same sha for different branch
	var err = builds.Insert(&model.Build{CommitID: 2, Index: 3})
	if err != nil {
		t.Errorf("Want Build created, got %s", err)
	}

	// verify unique remote + remote id constraint
	err = builds.Insert(&model.Build{CommitID: 1, Index: 1})
	if err == nil {
		t.Error("Want unique constraint violated")
	}
}

// testCommit is a helper function that compares the commit
// to an expected set of fixed field values.
func testBuild(t *testing.T, build *model.Build) {
	var got, want = build.Status, "Success"
	if got != want {
		t.Errorf("Want Status %v, got %v", want, got)
	}

	got, want = string(build.Output), "sample console output"
	if got != want {
		t.Errorf("Want Output %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = build.CommitID, int64(3)
	if gotInt64 != wantInt64 {
		t.Errorf("Want CommitID %v, got %v", want, got)
	}

	gotInt64, wantInt64 = build.Index, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Index %v, got %v", want, got)
	}

	gotInt64, wantInt64 = build.Created, int64(1398065343)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Created %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.Updated, int64(1398065344)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Updated %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.Started, int64(1398065345)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Started %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.Finished, int64(1398069999)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Finished %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.Duration, int64(854)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Duration %v, got %v", wantInt64, gotInt64)
	}
}
