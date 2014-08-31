package database

import (
	"testing"

	"github.com/drone/drone/shared/model"
	"github.com/jinzhu/gorm"
)

func Test_find(t *testing.T) {
	setup()
	defer teardown()

	manager := NewPermManager(conn.DB).(*permManager)
	perm, err := manager.find(&model.User{Id: 101}, &model.Repo{Id: 200})
	if err != nil {
		t.Errorf("Want permission, got %s", err)
	}

	var got, want = perm.Id, int64(1)
	if got != want {
		t.Errorf("Want ID %d, got %d", got, want)
	}

	got, want = perm.UserId, int64(101)
	if got != want {
		t.Errorf("Want Created %d, got %d", got, want)
	}

	got, want = perm.RepoId, int64(200)
	if got != want {
		t.Errorf("Want Created %d, got %d", got, want)
	}

	got, want = perm.Created, int64(1398065343)
	if got != want {
		t.Errorf("Want Created %d, got %d", got, want)
	}

	got, want = perm.Updated, int64(1398065344)
	if got != want {
		t.Errorf("Want Updated %d, got %d", got, want)
	}

	var gotBool, wantBool = perm.Read, true
	if gotBool != wantBool {
		t.Errorf("Want Read %v, got %v", gotBool, wantBool)
	}

	gotBool, wantBool = perm.Write, true
	if gotBool != wantBool {
		t.Errorf("Want Read %v, got %v", gotBool, wantBool)
	}

	gotBool, wantBool = perm.Admin, true
	if gotBool != wantBool {
		t.Errorf("Want Read %v, got %v", gotBool, wantBool)
	}

	// test that we get the appropriate error message when
	// no permissions are found in the database.
	_, err = manager.find(&model.User{Id: 102}, &model.Repo{Id: 201})
	if err != gorm.RecordNotFound {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

func TestPermFind(t *testing.T) {
	setup()
	defer teardown()

	manager := NewPermManager(conn.DB).(*permManager)

	u := model.User{Id: 101, Admin: false}
	r := model.Repo{Id: 201, Private: false}

	// public repos should always be accessible
	if perm := manager.Find(&u, &r); !perm.Read {
		t.Errorf("Public repos should always be READ accessible")
	}

	// public repos should always be accessible, even to guest users
	if perm := manager.Find(nil, &r); !perm.Read || perm.Write || perm.Admin {
		t.Errorf("Public repos should always be READ accessible, even to nil users")
	}

	// private repos should not be accessible to nil users
	r.Private = true
	if perm := manager.Find(nil, &r); perm.Read || perm.Write || perm.Admin {
		t.Errorf("Private repos should not be READ accessible to nil users")
	}

	// private repos should not be accessible to users without a row in the perm table.
	r.Private = true
	if perm := manager.Find(&u, &r); perm.Read || perm.Write || perm.Admin {
		t.Errorf("Private repos should not be READ accessible to users without a row in the perm table.")
	}

	// private repos should be accessible to admins
	r.Private = true
	u.Admin = true
	if perm := manager.Find(&u, &r); !perm.Read || !perm.Write || !perm.Admin {
		t.Errorf("Private repos should be READ accessible to admins")
	}

	// private repos should be accessible to users with rows in the perm table.
	r.Id = 200
	r.Private = true
	u.Admin = false
	if perm := manager.Find(&u, &r); !perm.Read {
		t.Errorf("Private repos should be READ accessible to users with rows in the perm table.")
	}
}

func TestPermRead(t *testing.T) {
	setup()
	defer teardown()
	var manager = NewPermManager(conn.DB)

	// dummy admin and repo
	u := model.User{Id: 101, Admin: false}
	r := model.Repo{Id: 201, Private: false}

	// public repos should always be accessible
	if read, err := manager.Read(&u, &r); !read || err != nil {
		t.Errorf("Public repos should always be READ accessible")
	}

	// public repos should always be accessible, even to guest users
	if read, err := manager.Read(nil, &r); !read || err != nil {
		t.Errorf("Public repos should always be READ accessible, even to nil users")
	}

	// private repos should not be accessible to nil users
	r.Private = true
	if read, err := manager.Read(nil, &r); read || err != nil {
		t.Errorf("Private repos should not be READ accessible to nil users")
	}

	// private repos should not be accessible to users without a row in the perm table.
	r.Private = true
	if read, _ := manager.Read(&u, &r); read {
		t.Errorf("Private repos should not be READ accessible to users without a row in the perm table.")
	}

	// private repos should be accessible to admins
	r.Private = true
	u.Admin = true
	if read, err := manager.Read(&u, &r); !read || err != nil {
		t.Errorf("Private repos should be READ accessible to admins")
	}

	// private repos should be accessible to users with rows in the perm table.
	r.Id = 200
	r.Private = true
	u.Admin = false
	if read, err := manager.Read(&u, &r); !read || err != nil {
		t.Errorf("Private repos should be READ accessible to users with rows in the perm table.")
	}
}

func TestPermWrite(t *testing.T) {
	setup()
	defer teardown()
	var manager = NewPermManager(conn.DB)

	// dummy admin and repo
	u := model.User{Id: 101, Admin: false}
	r := model.Repo{Id: 201, Private: false}

	// repos should not be accessible to nil users
	r.Private = true
	if write, err := manager.Write(nil, &r); write || err != nil {
		t.Errorf("Repos should not be WRITE accessible to nil users")
	}

	// repos should not be accessible to users without a row in the perm table.
	if write, _ := manager.Write(&u, &r); write {
		t.Errorf("Repos should not be WRITE accessible to users without a row in the perm table.")
	}

	// repos should be accessible to admins
	u.Admin = true
	if write, err := manager.Write(&u, &r); !write || err != nil {
		t.Errorf("Repos should be WRITE accessible to admins")
	}

	// repos should be accessible to users with rows in the perm table.
	r.Id = 200
	u.Admin = false
	if write, err := manager.Write(&u, &r); !write || err != nil {
		t.Errorf("Repos should be WRITE accessible to users with rows in the perm table.")
	}

	// repos should not be accessible to users with a row in the perm table, but write=false
	u.Id = 103
	u.Admin = false
	if write, err := manager.Write(&u, &r); write || err != nil {
		t.Errorf("Repos should not be WRITE accessible to users with perm.Write=false.")
	}
}

func TestPermAdmin(t *testing.T) {
	setup()
	defer teardown()
	var manager = NewPermManager(conn.DB)

	// dummy admin and repo
	u := model.User{Id: 101, Admin: false}
	r := model.Repo{Id: 201, Private: false}

	// repos should not be accessible to nil users
	r.Private = true
	if admin, err := manager.Admin(nil, &r); admin || err != nil {
		t.Errorf("Repos should not be ADMIN accessible to nil users")
	}

	// repos should not be accessible to users without a row in the perm table.
	if admin, _ := manager.Admin(&u, &r); admin {
		t.Errorf("Repos should not be ADMIN accessible to users without a row in the perm table.")
	}

	// repos should be accessible to admins
	u.Admin = true
	if admin, err := manager.Admin(&u, &r); !admin || err != nil {
		t.Errorf("Repos should be ADMIN accessible to admins")
	}

	// repos should be accessible to users with rows in the perm table.
	r.Id = 200
	u.Admin = false
	if admin, err := manager.Admin(&u, &r); !admin || err != nil {
		t.Errorf("Repos should be ADMIN accessible to users with rows in the perm table.")
	}

	// repos should not be accessible to users with a row in the perm table, but admin=false
	u.Id = 103
	u.Admin = false
	if admin, err := manager.Admin(&u, &r); admin || err != nil {
		t.Errorf("Repos should not be ADMIN accessible to users with perm.Admin=false.")
	}
}

func TestPermRevoke(t *testing.T) {
	setup()
	defer teardown()

	// dummy admin and repo
	u := model.User{Id: 101}
	r := model.Repo{Id: 200}

	manager := NewPermManager(conn.DB).(*permManager)
	admin, err := manager.Admin(&u, &r)
	if !admin || err != nil {
		t.Errorf("Want Admin permission, got Admin %v, error %s", admin, err)
	}

	// revoke permissions
	if err := manager.Revoke(&u, &r); err != nil {
		t.Errorf("Want revoked permissions, got %s", err)
	}

	perm, err := manager.find(&u, &r)
	if perm.Admin == true || err != gorm.RecordNotFound {
		t.Errorf("Expected revoked permission, got Admin %v, error %v", perm.Admin, err)
	}
}

func TestPermGrant(t *testing.T) {
	setup()
	defer teardown()

	// dummy admin and repo
	u := model.User{Id: 104}
	r := model.Repo{Id: 200}

	manager := NewPermManager(conn.DB).(*permManager)
	if err := manager.Grant(&u, &r, true, true, true); err != nil {
		t.Errorf("Want permissions granted, got %s", err)
	}

	// add new permissions
	perm, err := manager.find(&u, &r)
	if err != nil {
		t.Errorf("Want permission, got %s", err)
	} else if perm.Read != true {
		t.Errorf("Want Read permission True, got %v", perm.Read)
	} else if perm.Write != true {
		t.Errorf("Want Write permission True, got %v", perm.Write)
	} else if perm.Admin != true {
		t.Errorf("Want Admin permission True, got %v", perm.Admin)
	}

	// update permissions
	if err := manager.Grant(&u, &r, false, false, false); err != nil {
		t.Errorf("Want permissions granted, got %s", err)
	}

	// add new permissions
	perm, err = manager.find(&u, &r)
	if err != nil {
		t.Errorf("Want permission updated, got %s", err)
	} else if perm.Read != false {
		t.Errorf("Want Read permission False, got %v", perm.Read)
	} else if perm.Write != false {
		t.Errorf("Want Write permission False, got %v", perm.Write)
	} else if perm.Admin != false {
		t.Errorf("Want Admin permission False, got %v", perm.Admin)
	}
}
