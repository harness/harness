package bitbucket

import (
	"testing"
)

func Test_Teams(t *testing.T) {

	teams, err := client.Teams.List()
	if err != nil {
		t.Error(err)
		return
	}
	
	if len(teams) == 0 {
		t.Errorf("Returned an empty list of teams. Expected at least one result")
		return
	}

	if len(teams) == 1 {
		if teams[0].Name != testUser {
			t.Errorf("expected team name [%s], got [%s]", testUser, teams[0].Name)
		}
		if teams[0].Role != TeamRoleAdmin {
			t.Errorf("expected team role [%s], got [%s]", TeamRoleAdmin, teams[0].Role)
		}
	}
}
