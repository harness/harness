package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
	"github.com/drone/drone/pkg/model"
)

// TODO unit test to verify unique constraint on Team.Name

// TestGetMember tests the ability to retrieve a Team
// Member from the database by Unique ID.
func TestGetMember(t *testing.T) {
	Setup()
	defer Teardown()

	// get member by user_id and team_id
	member, err := database.GetMember(1, 1)
	if err != nil {
		t.Error(err)
	}

	if member.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, member.ID)
	}

	if member.Name != "Brad Rydzewski" {
		t.Errorf("Exepected Name %s, got %s", "Brad Rydzewski", member.Name)
	}

	if member.Email != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Email %s, got %s", "brad.rydzewski@gmail.com", member.Email)
	}

	if member.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", member.Gravatar)
	}

	if member.Role != model.RoleOwner {
		t.Errorf("Exepected Role %s, got %s", model.RoleOwner, member.Role)
	}
}

func TestIsMember(t *testing.T) {
	Setup()
	defer Teardown()

	ok, err := database.IsMember(1, 1)
	if err != nil {
		t.Error(err)
	}

	if !ok {
		t.Errorf("Expected IsMember to return true, returned false")
	}
}

func TestIsMemberAdmin(t *testing.T) {
	Setup()
	defer Teardown()

	// expecting user is Owner
	if ok, err := database.IsMemberAdmin(1, 1); err != nil {
		t.Error(err)
	} else if !ok {
		t.Errorf("Expected user id 1 IsMemberAdmin to return true, returned false")
	}

	// expecting user is Admin
	if ok, err := database.IsMemberAdmin(2, 1); err != nil {
		t.Error(err)
	} else if !ok {
		t.Errorf("Expected user id 2 IsMemberAdmin to return true, returned false")
	}

	// expecting user is NOT Admin (Write role)
	if ok, err := database.IsMemberAdmin(3, 1); err != nil {
		t.Error(err)
	} else if ok {
		t.Errorf("Expected user id 3 IsMemberAdmin to return false, returned true")
	}
}

func TestDeleteMember(t *testing.T) {
	Setup()
	defer Teardown()

	// delete member by user_id and team_id
	if err := database.DeleteMember(1, 1); err != nil {
		t.Error(err)
	}

	// get member by user_id and team_id
	if _, err := database.GetMember(1, 1); err == nil {
		t.Error(err)
	}

}

func TestListMembers(t *testing.T) {
	Setup()
	defer Teardown()

	// list members by team_id
	members, err := database.ListMembers(1)
	if err != nil {
		t.Error(err)
	}

	// verify team count
	if len(members) != 3 {
		t.Errorf("Exepected %d Team Members in database, got %d", 3, len(members))
		return
	}

	// get the first member in the list and verify
	// fields are being populated correctly
	member := members[0]

	if member.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, member.ID)
	}

	if member.Name != "Brad Rydzewski" {
		t.Errorf("Exepected Name %s, got %s", "Brad Rydzewski", member.Name)
	}

	if member.Email != "brad.rydzewski@gmail.com" {
		t.Errorf("Exepected Email %s, got %s", "brad.rydzewski@gmail.com", member.Email)
	}

	if member.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", member.Gravatar)
	}

	if member.Role != model.RoleOwner {
		t.Errorf("Exepected Role %s, got %s", model.RoleOwner, member.Role)
	}
}
