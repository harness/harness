package bitbucket

import (
	"testing"
)

func Test_Users(t *testing.T) {

	// FIND the currently authenticated user
	curr, err := client.Users.Current()
	if err != nil {
		t.Error(err)
	}

	// Find the user by Id
	user, err := client.Users.Find(curr.User.Username)
	if err != nil {
		t.Error(err)
	}

	// verify we get back the correct data
	if user.User.Username != curr.User.Username {
		t.Errorf("username [%v]; want [%v]", user.User.Username, curr.User.Username)
	}

}

func Test_UsersGuest(t *testing.T) {

	// FIND the currently authenticated user
	user, err := Guest.Users.Find(testUser)
	if err != nil {
		t.Error(err)
	}

	// verify we get back the correct data
	if user.User.Username != testUser {
		t.Errorf("username [%v]; want [%v]", user.User.Username, testUser)
	}
}

