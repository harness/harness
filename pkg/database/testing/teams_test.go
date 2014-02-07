package database

import (
	"testing"

	"github.com/drone/drone/pkg/database"
)

// TODO unit test to verify unique constraint on Member.UserID and Member.TeamID

// TestGetTeam tests the ability to retrieve a Team
// from the database by Unique ID.
func TestGetTeam(t *testing.T) {
	Setup()
	defer Teardown()

	team, err := database.GetTeam(1)
	if err != nil {
		t.Error(err)
	}

	if team.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, team.ID)
	}

	if team.Name != "Drone" {
		t.Errorf("Exepected Name %s, got %s", "Drone", team.Name)
	}

	if team.Slug != "drone" {
		t.Errorf("Exepected Slug %s, got %s", "drone", team.Slug)
	}

	if team.Email != "support@drone.io" {
		t.Errorf("Exepected Email %s, got %s", "brad@drone.io", team.Email)
	}

	if team.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", team.Gravatar)
	}
}

// TestGetTeamName tests the ability to retrieve a Team
// from the database by Unique Team Name (aka Slug).
func TestGetTeamSlug(t *testing.T) {
	Setup()
	defer Teardown()

	team, err := database.GetTeamSlug("drone")
	if err != nil {
		t.Error(err)
	}

	if team.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, team.ID)
	}

	if team.Name != "Drone" {
		t.Errorf("Exepected Name %s, got %s", "Drone", team.Name)
	}

	if team.Slug != "drone" {
		t.Errorf("Exepected Slug %s, got %s", "drone", team.Slug)
	}

	if team.Email != "support@drone.io" {
		t.Errorf("Exepected Email %s, got %s", "brad@drone.io", team.Email)
	}

	if team.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", team.Gravatar)
	}
}

// TestUpdateTeam tests the ability to updatee an
// existing Team in the database.
func TestUpdateTeam(t *testing.T) {
	Setup()
	defer Teardown()

	// get the user we plan to update
	team, err := database.GetTeam(1)
	if err != nil {
		t.Error(err)
	}

	// update fields
	team.Email = "brad@drone.io"
	team.Gravatar = "61024896f291303615bcd4f7a0dcfb74"

	// update the database
	if err := database.SaveTeam(team); err != nil {
		t.Error(err)
	}

	// get the updated team
	updatedTeam, err := database.GetTeam(1)
	if err != nil {
		t.Error(err)
	}

	// verify the updated fields
	if team.Email != updatedTeam.Email {
		t.Errorf("Exepected Email %s, got %s", team.Email, updatedTeam.Email)
	}

	if team.Gravatar != updatedTeam.Gravatar {
		t.Errorf("Exepected Gravatar %s, got %s", team.Gravatar, updatedTeam.Gravatar)
	}
}

// Test the ability to delete a Team.
func TestDeleteTeam(t *testing.T) {
	Setup()
	defer Teardown()

	// get the team we plan to update
	if err := database.DeleteTeam(1); err != nil {
		t.Error(err)
	}

	// now try to get the team from the database
	_, err := database.GetTeam(1)
	if err == nil {
		t.Fail()
	}
}

// Test the ability to get a list of Teams
// to which a User belongs.
func TestListTeam(t *testing.T) {
	Setup()
	defer Teardown()

	teams, err := database.ListTeams(1)
	if err != nil {
		t.Error(err)
	}

	// verify team count
	if len(teams) != 3 {
		t.Errorf("Exepected %d teams in database, got %d", 3, len(teams))
		return
	}

	// get the first user in the list and verify
	// fields are being populated correctly
	team := teams[0]

	if team.ID != 1 {
		t.Errorf("Exepected ID %d, got %d", 1, team.ID)
	}

	if team.Name != "Drone" {
		t.Errorf("Exepected Name %s, got %s", "Drone", team.Name)
	}

	if team.Slug != "drone" {
		t.Errorf("Exepected Slug %s, got %s", "drone", team.Slug)
	}

	if team.Email != "support@drone.io" {
		t.Errorf("Exepected Email %s, got %s", "brad@drone.io", team.Email)
	}

	if team.Gravatar != "8c58a0be77ee441bb8f8595b7f1b4e87" {
		t.Errorf("Exepected Gravatar %s, got %s", "8c58a0be77ee441bb8f8595b7f1b4e87", team.Gravatar)
	}
}
