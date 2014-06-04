package build

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/resource/build/buildtest"
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

	builds := NewManager(db)
	build, err := builds.Find(1)
	if err != nil {
		t.Errorf("Want Build from ID, got %s", err)
	}

	testBuild(t, build)
}

func TestFindNumber(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	build, err := builds.FindNumber(2, 1)
	if err != nil {
		t.Errorf("Want Build from Number, got %s", err)
	}

	testBuild(t, build)
}

func TestFindOutput(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	out, err := builds.FindOutput(2, 1)
	if err != nil {
		t.Errorf("Want Build Output from Number, got %s", err)
	}

	var got, want = string(out), "some output"
	if got != want {
		t.Errorf("Want build output %v, got %v", want, got)
	}
}

func TestList(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	list, err := builds.List(2)
	if err != nil {
		t.Errorf("Want List from CommitID, got %s", err)
	}

	var got, want = len(list), 3
	if got != want {
		t.Errorf("Want List size %v, got %v", want, got)
	}

	testBuild(t, list[0])
}

func TestInsert(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	if err := builds.Insert(&Build{CommitID: 4, Number: 1}); err != nil {
		t.Errorf("Want Build created, got %s", err)
	}

	// verify that it is ok to add same commit ID for incremented build number
	if err := builds.Insert(&Build{CommitID: 4, Number: 2}); err != nil {
		t.Errorf("Want Build created, got %s", err)
	}

	// verify that unique constraint fails when commit ID and build number already exist
	err := builds.Insert(&Build{CommitID: 4, Number: 2})
	if err == nil || !strings.Contains(err.Error(), "commit_id, build_number are not unique") {
		t.Errorf("Want unique constraint violated, got %s", err)
	}
}

func TestUpdate(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	build, err := builds.Find(5)
	if err != nil {
		t.Errorf("Want Build from ID, got %s", err)
	}

	// update the build's access token
	build.Status = "Success"
	build.Finished = time.Now().Unix()
	build.Duration = 999
	if err := builds.Update(build); err != nil {
		t.Errorf("Want Build updated, got %s", err)
	}

	updated, _ := builds.Find(5)
	var got, want = updated.Status, "Success"
	if got != want {
		t.Errorf("Want updated Status %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = updated.ID, build.ID
	if gotInt64 != wantInt64 {
		t.Errorf("Want build ID %v, got %v", wantInt64, gotInt64)
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

func TestFindUpdateOutput(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	build, err := builds.Find(5)
	if err != nil {
		t.Errorf("Want Build from ID, got %s", err)
	}

	if err := builds.UpdateOutput(build, []byte("some output ...")); err != nil {
		t.Errorf("Want Build updated, got %s", err)
	}

	out, err := builds.FindOutput(build.CommitID, build.Number)
	if err != nil {
		t.Errorf("Want Build Output, got %s", err)
	}

	var got, want = string(out), "some output ..."
	if got != want {
		t.Errorf("Want Build Output %v, got %v", want, got)
	}
}

func TestDelete(t *testing.T) {
	setup()
	defer teardown()

	builds := NewManager(db)
	build, err := builds.Find(1)
	if err != nil {
		t.Errorf("Want Build from ID, got %s", err)
	}

	// delete the builds
	if err := builds.Delete(build); err != nil {
		t.Errorf("Want Build deleted, got %s", err)
	}

	// check to see if the deleted build is actually gone
	if _, err := builds.Find(1); err != sql.ErrNoRows {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

// testBuild is a helper function that compares the build
// to an expected set of fixed field values.
func testBuild(t *testing.T, build *Build) {
	var got, want = build.Status, "Success"
	if got != want {
		t.Errorf("Want Status %v, got %v", want, got)
	}

	got, want = build.Matrix, ""
	if got != want {
		t.Errorf("Want Matrix %v, got %v", want, got)
	}

	var gotInt64, wantInt64 = build.ID, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.CommitID, int64(2)
	if gotInt64 != wantInt64 {
		t.Errorf("Want CommitID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = build.Number, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Number %v, got %v", wantInt64, gotInt64)
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
