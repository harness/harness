package github

import (
	"testing"
)

func Test_Users(t *testing.T) {

	// Get the currently authenticated user
	curr, err := client.Users.Current()
	if err != nil {
		t.Error(err)
		return
	}
	if curr.Login != testUser {
		t.Errorf("current user login [%v]; want [%v]", curr.Login, testUser)
	}

	// Get the named user
	u, err := client.Users.Find(testUser)
	if err != nil {
		t.Error(err)
		return
	}
	if curr.Login != u.Login {
		t.Errorf("named user login [%v]; want [%v]", curr.Login, u.Login)
	}

}

func Test_UsersGuest(t *testing.T) {

	// Get the named user
	u, err := Guest.Users.Find(testUser)
	if err != nil {
		t.Error(err)
		return
	}
	if testUser != u.Login {
		t.Errorf("named user login [%v]; want [%v]", u.Login, testUser)
	}
}
