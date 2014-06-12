package database

import (
	"database/sql"
	"testing"

	"github.com/drone/drone/shared/model"
)

func TestRepoFind(t *testing.T) {
	setup()
	defer teardown()

	repos := NewRepoManager(db)
	repo, err := repos.Find(1)
	if err != nil {
		t.Errorf("Want Repo from ID, got %s", err)
	}

	testRepo(t, repo)
}

func TestRepoFindName(t *testing.T) {
	setup()
	defer teardown()

	repos := NewRepoManager(db)
	user, err := repos.FindName("github.com", "lhofstadter", "lenwoloppali")
	if err != nil {
		t.Errorf("Want Repo by Name, got %s", err)
	}

	testRepo(t, user)
}

func TestRepoList(t *testing.T) {
	setup()
	defer teardown()

	repos := NewRepoManager(db)
	all, err := repos.List(1)
	if err != nil {
		t.Errorf("Want Repos, got %s", err)
	}

	var got, want = len(all), 2
	if got != want {
		t.Errorf("Want %v Repos, got %v", want, got)
	}

	testRepo(t, all[0])
}

func TestRepoInsert(t *testing.T) {
	setup()
	defer teardown()

	repo, _ := model.NewRepo("github.com", "mrwolowitz", "lenwoloppali")
	repos := NewRepoManager(db)
	if err := repos.Insert(repo); err != nil {
		t.Errorf("Want Repo created, got %s", err)
	}

	// verify unique remote + owner + name login constraint
	var err = repos.Insert(&model.Repo{Host: repo.Host, Owner: repo.Owner, Name: repo.Name})
	if err == nil {
		t.Error("Want unique constraint violated")
	}
}

func TestRepoUpdate(t *testing.T) {
	setup()
	defer teardown()

	repos := NewRepoManager(db)
	repo, err := repos.Find(1)
	if err != nil {
		t.Errorf("Want Repo from ID, got %s", err)
	}

	// update the repo's access token
	repo.Active = false
	repo.Private = false
	repo.Privileged = false
	repo.PostCommit = false
	repo.PullRequest = false
	if err := repos.Update(repo); err != nil {
		t.Errorf("Want Repo updated, got %s", err)
	}

	updated, _ := repos.Find(1)
	var got, want = updated.Active, repo.Active
	if got != want {
		t.Errorf("Want updated Active %v, got %v", want, got)
	}

	got, want = updated.Private, repo.Private
	if got != want {
		t.Errorf("Want updated Private %v, got %v", want, got)
	}

	got, want = updated.Privileged, repo.Privileged
	if got != want {
		t.Errorf("Want updated Privileged %v, got %v", want, got)
	}

	got, want = updated.PostCommit, repo.PostCommit
	if got != want {
		t.Errorf("Want updated PostCommit %v, got %v", want, got)
	}

	got, want = updated.PullRequest, repo.PullRequest
	if got != want {
		t.Errorf("Want updated PullRequest %v, got %v", want, got)
	}
}

func TestRepoDelete(t *testing.T) {
	setup()
	defer teardown()

	repos := NewRepoManager(db)
	repo, err := repos.Find(1)
	if err != nil {
		t.Errorf("Want Repo from ID, got %s", err)
	}

	// delete the repo
	if err := repos.Delete(repo); err != nil {
		t.Errorf("Want Repo deleted, got %s", err)
	}

	// check to see if the deleted repo is actually gone
	if _, err := repos.Find(1); err != sql.ErrNoRows {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

// testRepo is a helper function that compares the repo
// to an expected set of fixed field values.
func testRepo(t *testing.T, repo *model.Repo) {
	var got, want = repo.Remote, "github.com"
	if got != want {
		t.Errorf("Want Remote %v, got %v", want, got)
	}

	got, want = repo.Host, "github.com"
	if got != want {
		t.Errorf("Want Host %v, got %v", want, got)
	}

	got, want = repo.Owner, "lhofstadter"
	if got != want {
		t.Errorf("Want Owner %v, got %v", want, got)
	}

	got, want = repo.Name, "lenwoloppali"
	if got != want {
		t.Errorf("Want Name %v, got %v", want, got)
	}

	got, want = repo.CloneURL, "git://github.com/lhofstadter/lenwoloppali.git"
	if got != want {
		t.Errorf("Want URL %v, got %v", want, got)
	}

	got, want = repo.PublicKey, "publickey"
	if got != want {
		t.Errorf("Want PublicKey %v, got %v", want, got)
	}

	got, want = repo.PrivateKey, "privatekey"
	if got != want {
		t.Errorf("Want PrivateKey %v, got %v", want, got)
	}

	got, want = repo.Params, "params"
	if got != want {
		t.Errorf("Want Params %v, got %v", want, got)
	}

	var gotBool, wantBool = repo.Active, true
	if gotBool != wantBool {
		t.Errorf("Want Active %v, got %v", wantBool, gotBool)
	}

	gotBool, wantBool = repo.Private, true
	if gotBool != wantBool {
		t.Errorf("Want Private %v, got %v", wantBool, gotBool)
	}

	gotBool, wantBool = repo.Privileged, true
	if gotBool != wantBool {
		t.Errorf("Want Privileged %v, got %v", wantBool, gotBool)
	}

	gotBool, wantBool = repo.PostCommit, true
	if gotBool != wantBool {
		t.Errorf("Want PostCommit %v, got %v", wantBool, gotBool)
	}

	gotBool, wantBool = repo.PullRequest, true
	if gotBool != wantBool {
		t.Errorf("Want PullRequest %v, got %v", wantBool, gotBool)
	}

	var gotInt64, wantInt64 = repo.ID, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = repo.Created, int64(1398065343)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Created %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = repo.Updated, int64(1398065344)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Updated %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = repo.Timeout, int64(900)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Timeout %v, got %v", wantInt64, gotInt64)
	}
}
