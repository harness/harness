package database

import (
	"database/sql"
	"testing"

	"github.com/drone/drone/server/database/schema"
	"github.com/drone/drone/server/database/testdata"
	"github.com/drone/drone/server/database/testdatabase"
	"github.com/drone/drone/shared/model"
)

// in-memory database instance for unit testing
var db *sql.DB

// setup the test database and test fixtures
func setup() {
	db, _ = testdatabase.Open()
	schema.Load(db)
	testdata.Load(db)
}

// teardown the test database
func teardown() {
	db.Close()
}

func TestUserFind(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	user, err := users.Find(1)
	if err != nil {
		t.Errorf("Want User from ID, got %s", err)
	}

	testUser(t, user)
}

func TestUserFindLogin(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	user, err := users.FindLogin("github.com", "smellypooper")
	if err != nil {
		t.Errorf("Want User from Login, got %s", err)
	}

	testUser(t, user)
}

func TestUserFindToken(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	user, err := users.FindToken("e42080dddf012c718e476da161d21ad5")
	if err != nil {
		t.Errorf("Want User from Token, got %s", err)
	}

	testUser(t, user)
}

func TestUserList(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	all, err := users.List()
	if err != nil {
		t.Errorf("Want Users, got %s", err)
	}

	var got, want = len(all), 4
	if got != want {
		t.Errorf("Want %v Users, got %v", want, got)
	}

	testUser(t, all[0])
}

func TestUserInsert(t *testing.T) {
	setup()
	defer teardown()

	user := model.NewUser("github.com", "winkle", "winkle@caltech.edu")
	users := NewUserManager(db)
	if err := users.Insert(user); err != nil {
		t.Errorf("Want User created, got %s", err)
	}

	var got, want = user.ID, int64(5)
	if want != got {
		t.Errorf("Want User ID %v, got %v", want, got)
	}

	// verify unique remote + remote login constraint
	var err = users.Insert(&model.User{Remote: user.Remote, Login: user.Login, Token: "f71eb4a81a2cca56035dd7f6f2942e41"})
	if err == nil {
		t.Error("Want Token unique constraint violated")
	}

	// verify unique token constraint
	err = users.Insert(&model.User{Remote: "gitlab.com", Login: user.Login, Token: user.Token})
	if err == nil {
		t.Error("Want Token unique constraint violated")
	}
}

func TestUserUpdate(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	user, err := users.Find(4)
	if err != nil {
		t.Errorf("Want User from ID, got %s", err)
	}

	// update the user's access token
	user.Access = "fc47f37716fa04e9dfa9ac7eb22b5718"
	user.Secret = "d1c65427c978f2c9ad4baed72628dba0"
	if err := users.Update(user); err != nil {
		t.Errorf("Want User updated, got %s", err)
	}

	updated, _ := users.Find(4)
	var got, want = updated.Access, user.Access
	if got != want {
		t.Errorf("Want updated Access %s, got %s", want, got)
	}

	got, want = updated.Secret, user.Secret
	if got != want {
		t.Errorf("Want updated Secret %s, got %s", want, got)
	}
}

func TestUserDelete(t *testing.T) {
	setup()
	defer teardown()

	users := NewUserManager(db)
	user, err := users.Find(1)
	if err != nil {
		t.Errorf("Want User from ID, got %s", err)
	}

	// delete the user
	if err := users.Delete(user); err != nil {
		t.Errorf("Want User deleted, got %s", err)
	}

	// check to see if the deleted user is actually gone
	if _, err := users.Find(1); err != sql.ErrNoRows {
		t.Errorf("Want ErrNoRows, got %s", err)
	}
}

// testUser is a helper function that compares the user
// to an expected set of fixed field values.
func testUser(t *testing.T, user *model.User) {
	var got, want = user.Login, "smellypooper"
	if got != want {
		t.Errorf("Want Token %v, got %v", want, got)
	}

	got, want = user.Remote, "github.com"
	if got != want {
		t.Errorf("Want Token %v, got %v", want, got)
	}

	got, want = user.Access, "f0b461ca586c27872b43a0685cbc2847"
	if got != want {
		t.Errorf("Want Access Token %v, got %v", want, got)
	}

	got, want = user.Secret, "976f22a5eef7caacb7e678d6c52f49b1"
	if got != want {
		t.Errorf("Want Token Secret %v, got %v", want, got)
	}

	got, want = user.Name, "Dr. Cooper"
	if got != want {
		t.Errorf("Want Name %v, got %v", want, got)
	}

	got, want = user.Email, "drcooper@caltech.edu"
	if got != want {
		t.Errorf("Want Email %v, got %v", want, got)
	}

	got, want = user.Gravatar, "b9015b0857e16ac4d94a0ffd9a0b79c8"
	if got != want {
		t.Errorf("Want Gravatar %v, got %v", want, got)
	}

	got, want = user.Token, "e42080dddf012c718e476da161d21ad5"
	if got != want {
		t.Errorf("Want Token %v, got %v", want, got)
	}

	var gotBool, wantBool = user.Active, true
	if gotBool != wantBool {
		t.Errorf("Want Active %v, got %v", wantBool, gotBool)
	}

	gotBool, wantBool = user.Admin, true
	if gotBool != wantBool {
		t.Errorf("Want Admin %v, got %v", wantBool, gotBool)
	}

	var gotInt64, wantInt64 = user.ID, int64(1)
	if gotInt64 != wantInt64 {
		t.Errorf("Want ID %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = user.Created, int64(1398065343)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Created %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = user.Updated, int64(1398065344)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Updated %v, got %v", wantInt64, gotInt64)
	}

	gotInt64, wantInt64 = user.Synced, int64(1398065345)
	if gotInt64 != wantInt64 {
		t.Errorf("Want Synced %v, got %v", wantInt64, gotInt64)
	}
}
