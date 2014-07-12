package database

import (
	"database/sql"
	"testing"

	"github.com/drone/drone/shared/model"
)

func TestRemoteFind(t *testing.T) {
	setup()
	defer teardown()

	remotes := NewRemoteManager(db)
	remote, err := remotes.Find(1)
	if err != nil {
		t.Errorf("Want Remote from ID, got %s", err)
	}

	testRemote(t, remote)
}

func TestRemoteFindHost(t *testing.T) {
	setup()
	defer teardown()

	remotes := NewRemoteManager(db)
	remote, err := remotes.FindHost("github.drone.io")
	if err != nil {
		t.Errorf("Want Remote from Host, got %s", err)
	}

	testRemote(t, remote)
}

func TestRemoteList(t *testing.T) {
	setup()
	defer teardown()

	remotes := NewRemoteManager(db)
	all, err := remotes.List()
	if err != nil {
		t.Errorf("Want Remotes, got %s", err)
	}

	var got, want = len(all), 2
	if got != want {
		t.Errorf("Want %v remotes, got %v", want, got)
	}

	testRemote(t, all[0])
}

func TestRemoteInsert(t *testing.T) {
	setup()
	defer teardown()

	remote := &model.Remote{ID: 0, Type: "bitbucket.org", Host: "bitbucket.org", URL: "https://bitbucket.org", API: "https://bitbucket.org", Client: "abc", Secret: "123", Open: false}
	remotes := NewRemoteManager(db)
	if err := remotes.Insert(remote); err != nil {
		t.Errorf("Want Remote created, got %s", err)
	}

	var got, want = remote.ID, int64(3)
	if want != got {
		t.Errorf("Want Remote ID %v, got %v", want, got)
	}

	// verify unique remote name constraint
	var err = remotes.Insert(&model.Remote{Type: "bitbucket.org", Host: "butbucket.com"})
	if err == nil {
		t.Error("Want Type unique constraint violated")
	}

}

func TestRemoteUpdate(t *testing.T) {
	setup()
	defer teardown()

	remotes := NewRemoteManager(db)
	remote, err := remotes.Find(1)
	if err != nil {
		t.Errorf("Want Remote from ID, got %s", err)
	}

	// update the remote's address
	remote.Client = "abc"
	remote.Secret = "123"
	remote.Host = "git.drone.io"
	remote.URL = "https://git.drone.io"
	remote.API = "https://git.drone.io/v3/api"
	if err := remotes.Update(remote); err != nil {
		t.Errorf("Want Remote updated, got %s", err)
	}

	updated, _ := remotes.Find(1)
	var got, want = remote.Host, remote.Host
	if got != want {
		t.Errorf("Want updated Host %s, got %s", want, got)
	}

	got, want = updated.Client, remote.Client
	if got != want {
		t.Errorf("Want updated Client %s, got %s", want, got)
	}

	got, want = updated.Secret, remote.Secret
	if got != want {
		t.Errorf("Want updated Secret %s, got %s", want, got)
	}

	got, want = updated.Host, remote.Host
	if got != want {
		t.Errorf("Want updated Host %s, got %s", want, got)
	}

	got, want = updated.Host, remote.Host
	if got != want {
		t.Errorf("Want updated Host %s, got %s", want, got)
	}

	got, want = updated.URL, remote.URL
	if got != want {
		t.Errorf("Want updated URL %s, got %s", want, got)
	}

	got, want = updated.API, remote.API
	if got != want {
		t.Errorf("Want updated API %s, got %s", want, got)
	}
}

func TestRemoteDelete(t *testing.T) {
	setup()
	defer teardown()

	remotes := NewRemoteManager(db)
	remote, err := remotes.Find(1)
	if err != nil {
		t.Errorf("Want Remote from ID, got %s", err)
	}

	// delete the remote
	if err := remotes.Delete(remote); err != nil {
		t.Errorf("Want Remote deleted, got %s", err)
	}

	// check to see if the deleted remote is actually gone
	if _, err := remotes.Find(1); err != sql.ErrNoRows {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

func testRemote(t *testing.T, remote *model.Remote) {

	var got, want = remote.Host, "github.drone.io"
	if got != want {
		t.Errorf("Want Host %v, got %v", want, got)
	}

	got, want = remote.Type, "enterprise.github.com"
	if got != want {
		t.Errorf("Want Type %v, got %v", want, got)
	}

	got, want = remote.URL, "https://github.drone.io"
	if got != want {
		t.Errorf("Want URL %v, got %v", want, got)
	}

	got, want = remote.API, "https://github.drone.io/v3/api"
	if got != want {
		t.Errorf("Want API %v, got %v", want, got)
	}

	got, want = remote.Client, "f0b461ca586c27872b43a0685cbc2847"
	if got != want {
		t.Errorf("Want Access Token %v, got %v", want, got)
	}

	got, want = remote.Secret, "976f22a5eef7caacb7e678d6c52f49b1"
	if got != want {
		t.Errorf("Want Token Secret %v, got %v", want, got)
	}

	var gotBool, wantBool = remote.Open, true
	if gotBool != wantBool {
		t.Errorf("Want Open %v, got %v", wantBool, gotBool)
	}

	var gotInt64, wantInt64 = remote.ID, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}
}
