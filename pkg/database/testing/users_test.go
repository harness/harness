package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

// TODO unit test to verify unique constraint on User.Username
// TODO unit test to verify unique constraint on User.Email

// TestGetUser tests the ability to retrieve a User
// from the database by Unique ID.
func TestGetUser(t *testing.T) {
	Setup()
	defer Teardown()

	u, err := database.GetUser(1)
	if err != nil {
		t.Error(err)
	}

	if u.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, u.ID)
	}

	if u.Password != "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS" {
		t.Errorf("Exepected Password %s, got %s", "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS", u.Password)
	}

	if u.Token != "123" {
		t.Errorf("Exepected Token %s, got %s", "123", u.Token)
	}

	if u.Name != "Brad Rydzewski" {
		t.Errorf("Exepected Name %s, got %s", "Brad Rydzewski", u.Name)
	}

	if u.Email != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Email %s, got %s", "brad.rydzewski@gmail.com", u.Email)
	}

	if u.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", u.Gravatar)
	}
}

// TestGetUseEmail tests the ability to retrieve a User
// from the database by Email address.
func TestGetUserEmail(t *testing.T) {
	Setup()
	defer Teardown()

	u, err := database.GetUserEmail("brad.rydzewski@gmail.com")
	if err != nil {
		t.Error(err)
	}

	if u.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, u.ID)
	}

	if u.Password != "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS" {
		t.Errorf("Exepected Password %s, got %s", "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS", u.Password)
	}

	if u.Token != "123" {
		t.Errorf("Exepected Token %s, got %s", "123", u.Token)
	}

	if u.Name != "Brad Rydzewski" {
		t.Errorf("Exepected Name %s, got %s", "Brad Rydzewski", u.Name)
	}

	if u.Email != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Email %s, got %s", "brad.rydzewski@gmail.com", u.Email)
	}

	if u.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", u.Gravatar)
	}
}

// TestUpdateUser tests the ability to updatee an
// existing User in the database.
func TestUpdateUser(t *testing.T) {
	Setup()
	defer Teardown()

	// get the user we plan to update
	user, err := database.GetUser(1)
	if err != nil {
		t.Error(err)
	}

	// update fields
	user.Email = "brad@drone.io"
	user.Password = "password"

	// update the database
	if err := database.SaveUser(user); err != nil {
		t.Error(err)
	}

	// get the updated user
	updatedUser, err := database.GetUser(1)
	if err != nil {
		t.Error(err)
	}

	// verify the updated fields
	if user.Email != updatedUser.Email {
		t.Errorf("Exepected Email %s, got %s", user.Email, updatedUser.Email)
	}

	if user.Password != updatedUser.Password {
		t.Errorf("Exepected Password %s, got %s", user.Email, updatedUser.Password)
	}
}

// Deletes an existing User account.
func TestDeleteUser(t *testing.T) {
	Setup()
	defer Teardown()

	// get the user we plan to update
	if err := database.DeleteUser(1); err != nil {
		t.Error(err)
	}

	// now try to get the user from the database
	_, err := database.GetUser(1)
	if err == nil {
		t.Fail()
	}
}

// Returns a list of all Users.
func TestListUsers(t *testing.T) {
	Setup()
	defer Teardown()

	users, err := database.ListUsers()
	if err != nil {
		t.Error(err)
	}

	// verify user count
	if len(users) != 4 {
		t.Errorf("Exepected %d users in database, got %d", 4, len(users))
		return
	}

	// get the first user in the list and verify
	// fields are being populated correctly
	u := users[0]

	if u.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, u.ID)
	}

	if u.Password != "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS" {
		t.Errorf("Exepected Password %s, got %s", "$2a$10$b8d63QsTL38vx7lj0HEHfOdbu1PCAg6Gfca74UavkXooIBx9YxopS", u.Password)
	}

	if u.Token != "123" {
		t.Errorf("Exepected Token %s, got %s", "123", u.Token)
	}

	if u.Name != "Brad Rydzewski" {
		t.Errorf("Exepected Name %s, got %s", "Brad Rydzewski", u.Name)
	}

	if u.Email != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Email %s, got %s", "brad.rydzewski@gmail.com", u.Email)
	}

	if u.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", u.Gravatar)
	}
}
